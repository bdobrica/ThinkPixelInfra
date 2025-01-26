package db

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"api_gateway/config"
	"api_gateway/logger"
	"api_gateway/utils"

	_ "github.com/go-sql-driver/mysql"
)

var (
	dbInstance *sql.DB
	initOnce   sync.Once
)

type APIKeyDetails struct {
	ID               int
	IndexingNode     string
	ExpiresAt        time.Time
	MaxSearchResults int
	Model            string
	ChunkSize        int
	ChunkOverlap     int
}

// GetDBConnection provides a singleton instance of the database connection
func GetDBConnection() (*sql.DB, error) {
	var err error
	initOnce.Do(func() {
		dsn := config.GetEnv("API_GATEWAY_DB_DSN", "thinkpixel:thinkpixel@tcp(mysql:3306)/thinkpixel")
		dsn += "?parseTime=true"
		dbInstance, err = sql.Open("mysql", dsn)
		if err != nil {
			return
		}
		// Test the connection to ensure it's valid
		err = dbInstance.Ping()
	})
	if err != nil {
		return nil, err
	}
	return dbInstance, nil
}

// CloseDBConnection closes the database connection
func CloseDBConnection() error {
	if dbInstance != nil {
		return dbInstance.Close()
	}
	return nil
}

// GetAPIKeyDetails retrieves API key details from the database
func GetAPIKeyDetails(hashedKey string) (APIKeyDetails, error) {
	dbConn, err := GetDBConnection()
	if err != nil {
		return APIKeyDetails{}, err
	}

	query := `
		SELECT id, indexing_node, expires_at, max_search_results, model, chunk_size, chunk_overlap
		FROM wp_thinkpixel_sites
		WHERE api_key = ? AND status = 'active' AND (expires_at IS NULL OR expires_at > NOW())`

	row := dbConn.QueryRow(query, hashedKey)

	var keyDetails APIKeyDetails
	if err := row.Scan(&keyDetails.ID, &keyDetails.IndexingNode, &keyDetails.ExpiresAt, &keyDetails.MaxSearchResults, &keyDetails.Model, &keyDetails.ChunkSize, &keyDetails.ChunkOverlap); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return APIKeyDetails{}, errors.New("invalid API key")
		}
		return APIKeyDetails{}, fmt.Errorf("database query error %v", err)
	}

	return keyDetails, nil
}

// GetAPIKeyDetailsByID retrieves API key details from the database by API Key ID
func GetAPIKeyDetailsByID(siteId int) (APIKeyDetails, error) {
	dbConn, err := GetDBConnection()
	if err != nil {
		return APIKeyDetails{}, err
	}

	query := `
		SELECT id, indexing_node, expires_at, max_search_results, model, chunk_size, chunk_overlap
		FROM wp_thinkpixel_sites
		WHERE id = ? AND status = 'active' AND (expires_at IS NULL OR expires_at > NOW())`

	row := dbConn.QueryRow(query, siteId)

	var id int
	var indexingNode string
	var expiresAt sql.NullTime
	var maxSearchResults int
	var model string
	var chunkSize int
	var chunkOverlap int
	if err := row.Scan(&id, &indexingNode, &expiresAt, &maxSearchResults, &model, &chunkSize, &chunkOverlap); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return APIKeyDetails{}, errors.New("invalid API key")
		}
		return APIKeyDetails{}, fmt.Errorf("database query error %v", err)
	}

	return APIKeyDetails{}, nil
}

// StoreRegistrationData stores registration data in the database
func StoreRegistrationData(domain, path, validationToken string, validationTokenExpiresAt time.Time, estimatedPages, averagePageSize, stDevPageSize int) error {
	dbConn, err := GetDBConnection()
	if err != nil {
		return err
	}

	logger.Infof("Storing registration data for domain %s and path %s: token=%s, expires=%s, estimatedPages=%d, averagePageSize=%d, stDevPageSize=%d", domain, path, validationToken, validationTokenExpiresAt, estimatedPages, averagePageSize, stDevPageSize)

	query := `
        INSERT INTO wp_thinkpixel_sites (domain, path, validation_token, validation_token_expires_at, validation_method, validation_status, estimated_pages, average_page_size, st_dev_page_size, status)
        VALUES (?, ?, ?, ?, 'http', 'pending', ?, ?, ?, 'suspended')`

	_, err = dbConn.Exec(query, domain, path, validationToken, validationTokenExpiresAt, estimatedPages, averagePageSize, stDevPageSize)
	if err != nil {
		return fmt.Errorf("failed to store registration data: %v", err)
	}

	return nil
}

// GetValidationTokenDetails retrieves validation token details for pending status
func getValidationTokenDetails(domain, path, validationStatus string) (string, time.Time, error) {
	dbConn, err := GetDBConnection()
	if err != nil {
		return "", time.Now(), err
	}

	query := `
		SELECT validation_token, validation_token_expires_at
		FROM wp_thinkpixel_sites
		WHERE domain = ? AND path = ? AND validation_status = ?`

	row := dbConn.QueryRow(query, domain, path, validationStatus)

	var (
		validationToken          string
		validationTokenExpiresAt sql.NullTime
	)
	if err := row.Scan(&validationToken, &validationTokenExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", time.Now(), fmt.Errorf("no %s validation token found", validationStatus)
		}
		return "", time.Now(), fmt.Errorf("database query error %v", err)
	}

	return validationToken, validationTokenExpiresAt.Time, nil
}

func GetPendingTokenDetails(domain, path string) (string, time.Time, error) {
	return getValidationTokenDetails(domain, path, "pending")
}

func GetFailedTokenDetails(domain, path string) (string, time.Time, error) {
	return getValidationTokenDetails(domain, path, "failed")
}

func GetVerifiedTokenDetails(domain, path string) (string, time.Time, error) {
	return getValidationTokenDetails(domain, path, "verified")
}

// ActivateAPIKey updates the API key and status to active for a given validation token
func ActivateAPIKey(domain, path, apiKey string) error {
	dbConn, err := GetDBConnection()
	if err != nil {
		return err
	}

	validityStr := config.GetEnv("API_GATEWAY_API_KEY_VALIDITY", "720h")
	validity, err := time.ParseDuration(validityStr)
	if err != nil {
		return fmt.Errorf("failed to parse API_GATEWAY_API_KEY_VALIDITY: %v", err)
	}

	hashedApiKey := utils.HashString(apiKey)
	expiresAt := time.Now().Add(validity)
	query := `
		UPDATE wp_thinkpixel_sites
		SET api_key = ?, status = 'active', updated_at = NOW(), verified_at = NOW(), expires_at = ?
		WHERE domain = ? AND path = ? AND validation_status = 'verified'`

	result, err := dbConn.Exec(query, hashedApiKey, expiresAt, domain, path)
	if err != nil {
		return fmt.Errorf("failed to activate API key: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve affected rows: %v", err)
	}
	if affected == 0 {
		return errors.New("no matching record found to activate")
	} else {
		logger.Infof("API key activated for %s%s", domain, path)
	}

	// Retrieve the ID of the API key row you just activated, since it's the foreign key in wp_thinkpixel_index_requests.
	// This can be done in multiple ways; for instance, if you store the auto-increment in a local variable
	// or re-query the row after updating. For example:
	var (
		siteId, estimatedPages, averagePageSize, stDevPageSize int
	)
	lookupQuery := `
        SELECT id, estimated_pages, average_page_size, st_dev_page_size
        FROM wp_thinkpixel_sites
        WHERE api_key = ?
        LIMIT 1
    `
	err = dbConn.QueryRow(lookupQuery, apiKey).Scan(&siteId, &estimatedPages, &averagePageSize, &stDevPageSize)
	if err != nil {
		// Not strictly fatal here, but you may choose to handle differently
		logger.Errorf("Failed to retrieve site_id: %v", err)
		return err
	}

	// Estimate the memory needed for the API key and assign it to a Redis master
	var requestedBytes uint64 = utils.EstimateMemory(estimatedPages, averagePageSize, stDevPageSize)

	logger.Infof("Estimated memory for site %d: %d Mb", siteId, requestedBytes>>20)

	// Call the concurrency-safe function to assign a Redis master, if possible.
	// Do not fail activation if it fails to find capacity. That is handled in the function’s logic.
	if err := AssignToIndexingNode(siteId, requestedBytes); err != nil {
		// You might want to log this error but not necessarily fail activation
		logger.Errorf("Failed to assign Redis node: %v", err)
		// Decide if you want to return an error or continue
	}

	return nil
}

// VerifyToken marks a validation token as verified
func VerifyToken(validationToken, domain, path string) error {
	dbConn, err := GetDBConnection()
	if err != nil {
		return err
	}

	query := `
		UPDATE wp_thinkpixel_sites
		SET validation_status = 'verified', verified_at = NOW(), updated_at = NOW()
		WHERE validation_token = ? AND domain = ? AND path = ? AND validation_status IN ('pending', 'failed')`

	result, err := dbConn.Exec(query, validationToken, domain, path)
	if err != nil {
		return fmt.Errorf("failed to verify token: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve affected rows: %v", err)
	}
	if affected == 0 {
		return errors.New("token is already verified or not found")
	}

	return nil
}

// FailToken marks a validation token as failed
func FailToken(validationToken, domain, path string) error {
	dbConn, err := GetDBConnection()
	if err != nil {
		return err
	}

	query := `
		UPDATE wp_thinkpixel_sites
		SET validation_status = 'failed', verified_at = NOW(), updated_at = NOW()
		WHERE validation_token = ? AND domain = ? AND path = ? AND validation_status IN ('pending', 'failed')`

	result, err := dbConn.Exec(query, validationToken, domain, path)
	if err != nil {
		return fmt.Errorf("failed to mark token as failed: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve affected rows: %v", err)
	}
	if affected == 0 {
		return errors.New("token is already verified or not found")
	}

	return nil
}

// ResetValidationStatus resets the validation status for a given domain and path
func ResetValidationStatus(domain, path, newToken string, newTokenExpiresAt time.Time) error {
	dbConn, err := GetDBConnection()
	if err != nil {
		return err
	}

	query := `
        UPDATE wp_thinkpixel_sites
        SET validation_token = ?, validation_token_expires_at = ?, validation_status = 'pending', updated_at = NOW()
        WHERE domain = ? AND path = ? AND validation_status = 'failed'`

	result, err := dbConn.Exec(query, newToken, newTokenExpiresAt, domain, path)
	if err != nil {
		return fmt.Errorf("failed to reset validation status: %v", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to retrieve affected rows: %v", err)
	}
	if affected == 0 {
		return errors.New("no matching record found to reset")
	}

	return nil
}
