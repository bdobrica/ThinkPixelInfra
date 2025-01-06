package db

import (
	"database/sql"
	"errors"
	"strconv"
	"sync"
	"time"
    "fmt"

	"api_gateway/config"
    "api_gateway/logger"
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
		SELECT id, redis_server, expires_at, max_search_results
		FROM wp_api_keys
		WHERE api_key = ? AND status = 'active' AND (expires_at IS NULL OR expires_at > NOW())`

	row := dbConn.QueryRow(query, hashedKey)

	var id int
	var redisServer string
	var expiresAt sql.NullTime
	var maxSearchResults int
	if err := row.Scan(&id, &redisServer, &expiresAt, &maxSearchResults); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", time.Time{}, 0, errors.New("Invalid API key")
		}
		return 0, "", time.Time{}, 0, errors.New("Database query error")
	}

	return id, redisServer, expiresAt.Time, maxSearchResults, nil
}

// GetAPIKeyDetailsByID retrieves API key details from the database by API Key ID
func GetAPIKeyDetailsByID(apiKeyID int) (int, string, time.Time, int, error) {
	dbConn, err := GetDBConnection()
	if err != nil {
		return 0, "", time.Time{}, 0, err
	}

	query := `
		SELECT id, redis_server, expires_at
		FROM wp_api_keys
		WHERE id = ? AND status = 'active' AND (expires_at IS NULL OR expires_at > NOW())`

	row := dbConn.QueryRow(query, apiKeyID)

	var id int
	var redisServer string
	var expiresAt sql.NullTime
	var maxSearchResults int
	if err := row.Scan(&id, &redisServer, &expiresAt, &maxSearchResults); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", time.Time{}, 0, errors.New("Invalid API key")
		}
		return 0, "", time.Time{}, 0, errors.New("Database query error")
	}

	return id, redisServer, expiresAt.Time, maxSearchResults, nil
}

// StoreRegistrationData stores registration data in the database
func StoreRegistrationData(domain string, path string, requestSalt string, verificationToken string, estimatedPages int, averagePageSize int, stDevPageSize int) error {
	dbConn, err := GetDBConnection()
	if err != nil {
		return err
	}

    logger.Infof("Storing registration data for domain %s and path %s: requestSalt=%s, token=%s, estimatedPages=%d, averagePageSize=%d, stDevPageSize=%d", domain, path, requestSalt, verificationToken, estimatedPages, averagePageSize, stDevPageSize)

	query := `
		INSERT INTO wp_api_keys (domain, path, request_salt, verification_token, verification_method, verification_status, estimated_pages, average_page_size, st_dev_page_size, status)
		VALUES (?, ?, ?, ?, 'http', 'pending', ?, ?, ?, 'suspended')`

	_, err = dbConn.Exec(query, domain, path, requestSalt, verificationToken, estimatedPages, averagePageSize, stDevPageSize)
	if err != nil {
        logger.Errorf("Failed to store registration data: %v", err)
		return errors.New("Failed to store registration data: " + err.Error())
	}

	return nil
}

// GetVerificationTokenDetails retrieves verification token details for pending status
func getVerificationTokenDetails(domain, path, verificationStatus string) (string, string, error) {
	dbConn, err := GetDBConnection()
	if err != nil {
		return "", "", err
	}

	query := `
		SELECT verification_token, request_salt
		FROM wp_api_keys
		WHERE domain = ? AND path = ? AND verification_status = ?`

	row := dbConn.QueryRow(query, domain, path, verificationStatus)

	var (
        verificationToken string
        requestSalt string
    )
	if err := row.Scan(&verificationToken, &requestSalt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", fmt.Errorf("No %s verification token found", verificationStatus)
		}
		return "", "", errors.New("Database query error")
	}

	return verificationToken, requestSalt, nil
}

func GetPendingTokenDetails(domain, path string) (string, string, error) {
    return getVerificationTokenDetails(domain, path, "pending")
}

func GetFailedTokenDetails(domain, path string) (string, string, error) {
    return getVerificationTokenDetails(domain, path, "failed")
}

func GetVerifiedTokenDetails(domain, path string) (string, string, error) {
    return getVerificationTokenDetails(domain, path, "verified")
}

// ActivateAPIKey updates the API key and status to active for a given verification token
func ActivateAPIKey(verificationToken string, apiKey string) error {
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
		UPDATE wp_api_keys
		SET api_key = ?, status = 'active', updated_at = NOW(), verified_at = NOW(), expires_at = ?
		WHERE verification_token = ? AND verification_status = 'verified'`

	result, err := dbConn.Exec(query, apiKey, expiresAt, verificationToken)
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

	return nil
}

// VerifyToken marks a verification token as verified
func VerifyToken(verificationToken, domain, path string) error {
	dbConn, err := GetDBConnection()
	if err != nil {
		return err
	}

	query := `
		UPDATE wp_api_keys
		SET verification_status = 'verified', verified_at = NOW(), updated_at = NOW()
		WHERE verification_token = ? AND domain = ? AND path = ? AND verification_status IN ('pending', 'failed')`

	result, err := dbConn.Exec(query, verificationToken, domain, path)
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

// FailToken marks a verification token as failed
func FailToken(verificationToken, domain, path string) error {
	dbConn, err := GetDBConnection()
	if err != nil {
		return err
	}

	query := `
		UPDATE wp_api_keys
		SET verification_status = 'failed', verified_at = NOW(), updated_at = NOW()
		WHERE verification_token = ? AND domain = ? AND path = ? AND verification_status IN ('pending', 'failed')`

	result, err := dbConn.Exec(query, verificationToken, domain, path)
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

// ResetVerificationStatus resets the verification status for a given domain and path
func ResetVerificationStatus(domain, path, requestSalt string) error {
    dbConn, err := GetDBConnection()
    if err != nil {
        return err
    }

    query := `
        UPDATE wp_api_keys
        SET verification_status = 'pending', request_salt = ?, updated_at = NOW()
        WHERE domain = ? AND path = ? AND verification_status = 'failed'`

    result, err := dbConn.Exec(query, requestSalt, domain, path)
    if err != nil {
        return errors.New("Failed to reset verification status: " + err.Error())
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
