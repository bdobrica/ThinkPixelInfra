package db

import (
	"database/sql"
	"errors"
	"time"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	"api_gateway/config"
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
