package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
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

// GetDBConnection provides a singleton instance of the database connection
func GetDBConnection() (*sql.DB, error) {
	var err error
	initOnce.Do(func() {
		dsn := config.GetEnv("API_GATEWAY_DB_DSN", "thinkpixel:thinkpixel@tcp(mysql:3306)/thinkpixel")
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
func GetAPIKeyDetails(hashedKey string) (int, string, time.Time, int, error) {
	dbConn, err := GetDBConnection()
	if err != nil {
		return 0, "", time.Time{}, 0, err
	}

	query := `
		SELECT id, indexing_node, expires_at, max_search_results
		FROM wp_thinkpixel_sites
		WHERE api_key = ? AND status = 'active' AND (expires_at IS NULL OR expires_at > NOW())`

	row := dbConn.QueryRow(query, hashedKey)

	var id int
	var indexingNode string
	var expiresAt sql.NullTime
	var maxSearchResults int
	if err := row.Scan(&id, &indexingNode, &expiresAt, &maxSearchResults); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", time.Time{}, 0, errors.New("Invalid API key")
		}
		return 0, "", time.Time{}, 0, errors.New("Database query error")
	}

	return id, indexingNode, expiresAt.Time, maxSearchResults, nil
}

// GetAPIKeyDetailsByID retrieves API key details from the database by API Key ID
func GetAPIKeyDetailsByID(siteId int) (int, string, time.Time, int, error) {
	dbConn, err := GetDBConnection()
	if err != nil {
		return 0, "", time.Time{}, 0, err
	}

	query := `
		SELECT id, indexing_node, expires_at
		FROM wp_thinkpixel_sites
		WHERE id = ? AND status = 'active' AND (expires_at IS NULL OR expires_at > NOW())`

	row := dbConn.QueryRow(query, siteId)

	var id int
	var indexingNode string
	var expiresAt sql.NullTime
	var maxSearchResults int
	if err := row.Scan(&id, &indexingNode, &expiresAt, &maxSearchResults); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", time.Time{}, 0, errors.New("Invalid API key")
		}
		return 0, "", time.Time{}, 0, errors.New("Database query error")
	}

	return id, indexingNode, expiresAt.Time, maxSearchResults, nil
}

// StoreRegistrationData stores registration data in the database
func StoreRegistrationData(domain string, path string, requestSalt string, validationToken string, estimatedPages int, averagePageSize int, stDevPageSize int) error {
	dbConn, err := GetDBConnection()
	if err != nil {
		return err
	}

	logger.Infof("Storing registration data for domain %s and path %s: requestSalt=%s, token=%s, estimatedPages=%d, averagePageSize=%d, stDevPageSize=%d", domain, path, requestSalt, validationToken, estimatedPages, averagePageSize, stDevPageSize)

	query := `
		INSERT INTO wp_thinkpixel_sites (domain, path, request_salt, validation_token, validation_method, validation_status, estimated_pages, average_page_size, st_dev_page_size, status)
		VALUES (?, ?, ?, ?, 'http', 'pending', ?, ?, ?, 'suspended')`

	_, err = dbConn.Exec(query, domain, path, requestSalt, validationToken, estimatedPages, averagePageSize, stDevPageSize)
	if err != nil {
		logger.Errorf("Failed to store registration data: %v", err)
		return errors.New("Failed to store registration data: " + err.Error())
	}

	return nil
}

// GetValidationTokenDetails retrieves validation token details for pending status
func getValidationTokenDetails(domain, path, validationStatus string) (string, string, error) {
	dbConn, err := GetDBConnection()
	if err != nil {
		return "", "", err
	}

	query := `
		SELECT validation_token, request_salt
		FROM wp_thinkpixel_sites
		WHERE domain = ? AND path = ? AND validation_status = ?`

	row := dbConn.QueryRow(query, domain, path, validationStatus)

	var (
		validationToken string
		requestSalt     string
	)
	if err := row.Scan(&validationToken, &requestSalt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", fmt.Errorf("No %s validation token found", validationStatus)
		}
		return "", "", errors.New("Database query error")
	}

	return validationToken, requestSalt, nil
}

func GetPendingTokenDetails(domain, path string) (string, string, error) {
	return getValidationTokenDetails(domain, path, "pending")
}

func GetFailedTokenDetails(domain, path string) (string, string, error) {
	return getValidationTokenDetails(domain, path, "failed")
}

func GetVerifiedTokenDetails(domain, path string) (string, string, error) {
	return getValidationTokenDetails(domain, path, "verified")
}

// ActivateAPIKey updates the API key and status to active for a given validation token
func ActivateAPIKey(validationToken string, apiKey string) error {
	dbConn, err := GetDBConnection()
	if err != nil {
		return err
	}

	validityDaysStr := config.GetEnv("API_GATEWAY_API_KEY_VALIDITY", "30")
	validityDays, err := strconv.Atoi(validityDaysStr)
	if err != nil {
		return errors.New("Failed to parse API_GATEWAY_API_KEY_VALIDITY: " + err.Error())
	}

	expiresAt := time.Now().AddDate(0, 0, validityDays)
	query := `
		UPDATE wp_thinkpixel_sites
		SET api_key = ?, status = 'active', updated_at = NOW(), verified_at = NOW(), expires_at = ?
		WHERE validation_token = ? AND validation_status = 'verified'`

	result, err := dbConn.Exec(query, apiKey, expiresAt, validationToken)
	if err != nil {
		return errors.New("Failed to activate API key: " + err.Error())
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return errors.New("Failed to retrieve affected rows: " + err.Error())
	}
	if affected == 0 {
		return errors.New("No matching record found to activate")
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
		return errors.New("Failed to verify token: " + err.Error())
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return errors.New("Failed to retrieve affected rows: " + err.Error())
	}
	if affected == 0 {
		return errors.New("Token is already verified or not found")
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
		return errors.New("Failed to mark token as failed: " + err.Error())
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return errors.New("Failed to retrieve affected rows: " + err.Error())
	}
	if affected == 0 {
		return errors.New("Token is already verified or not found")
	}

	return nil
}

// ResetValidationStatus resets the validation status for a given domain and path
func ResetValidationStatus(domain, path, requestSalt string) error {
	dbConn, err := GetDBConnection()
	if err != nil {
		return err
	}

	query := `
        UPDATE wp_thinkpixel_sites
        SET validation_status = 'pending', request_salt = ?, updated_at = NOW()
        WHERE domain = ? AND path = ? AND validation_status = 'failed'`

	result, err := dbConn.Exec(query, requestSalt, domain, path)
	if err != nil {
		return errors.New("Failed to reset validation status: " + err.Error())
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return errors.New("Failed to retrieve affected rows: " + err.Error())
	}
	if affected == 0 {
		return errors.New("No matching record found to reset")
	}

	return nil
}
