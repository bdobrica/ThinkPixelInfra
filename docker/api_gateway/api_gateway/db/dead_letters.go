package db

import (
	"database/sql"
	"fmt"
	"time"

	"api_gateway/logger"
)

// DeadLetterRecord represents a failed message stored in the dead letter queue
type DeadLetterRecord struct {
	ID            int        `json:"id"`
	SiteID        int32      `json:"site_id"`
	DocID         int32      `json:"doc_id"`
	Subject       string     `json:"subject"`
	ErrorMessage  string     `json:"error_message"`
	PayloadJSON   string     `json:"payload_json"`
	RetryCount    int        `json:"retry_count"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	ReprocessedAt *time.Time `json:"reprocessed_at,omitempty"`
}

// InsertDeadLetter stores a failed message in the dead letter queue table
// Accepts individual fields instead of protobuf object to avoid import cycles
func InsertDeadLetter(siteID int32, docID int32, subject string, errorMsg string, payloadJSON string, retryCount int32) error {
	db, err := GetDBConnection()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	query := `
		INSERT INTO wp_thinkpixel_dead_letters
		(site_id, doc_id, subject, error_message, payload_json, retry_count, status)
		VALUES (?, ?, ?, ?, ?, ?, 'pending')
	`

	result, err := db.Exec(
		query,
		siteID,
		docID,
		subject,
		errorMsg,
		payloadJSON,
		retryCount,
	)

	if err != nil {
		return fmt.Errorf("failed to insert dead letter: %w", err)
	}

	dlqID, err := result.LastInsertId()
	if err != nil {
		logger.Warningf("Could not get last insert ID for DLQ record: %v", err)
	} else {
		logger.Infof("Stored message in DLQ (id=%d, site_id=%d, doc_id=%d, subject=%s)",
			dlqID, siteID, docID, subject)
	}

	return nil
}

// GetPendingDeadLetters retrieves all pending dead letter records
func GetPendingDeadLetters(limit int) ([]DeadLetterRecord, error) {
	db, err := GetDBConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	query := `
		SELECT id, site_id, doc_id, subject, error_message, payload_json,
		       retry_count, status, created_at, updated_at, reprocessed_at
		FROM wp_thinkpixel_dead_letters
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT ?
	`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query dead letters: %w", err)
	}
	defer rows.Close()

	var records []DeadLetterRecord
	for rows.Next() {
		var record DeadLetterRecord
		var reprocessedAt sql.NullTime

		err := rows.Scan(
			&record.ID,
			&record.SiteID,
			&record.DocID,
			&record.Subject,
			&record.ErrorMessage,
			&record.PayloadJSON,
			&record.RetryCount,
			&record.Status,
			&record.CreatedAt,
			&record.UpdatedAt,
			&reprocessedAt,
		)
		if err != nil {
			logger.Warningf("Failed to scan dead letter record: %v", err)
			continue
		}

		if reprocessedAt.Valid {
			record.ReprocessedAt = &reprocessedAt.Time
		}

		records = append(records, record)
	}

	return records, nil
}

// UpdateDeadLetterStatus updates the status of a dead letter record
func UpdateDeadLetterStatus(id int, status string) error {
	db, err := GetDBConnection()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	query := `
		UPDATE wp_thinkpixel_dead_letters
		SET status = ?,
		    updated_at = CURRENT_TIMESTAMP,
		    reprocessed_at = CASE WHEN ? = 'reprocessed' THEN CURRENT_TIMESTAMP ELSE reprocessed_at END
		WHERE id = ?
	`

	result, err := db.Exec(query, status, status, id)
	if err != nil {
		return fmt.Errorf("failed to update dead letter status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no dead letter found with id=%d", id)
	}

	logger.Infof("Updated DLQ record %d status to %s", id, status)
	return nil
}

// GetDeadLetterStats returns statistics about dead letter queue
func GetDeadLetterStats() (map[string]int, error) {
	db, err := GetDBConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status = 'reprocessed' THEN 1 ELSE 0 END) as reprocessed,
			SUM(CASE WHEN status = 'discarded' THEN 1 ELSE 0 END) as discarded
		FROM wp_thinkpixel_dead_letters
	`

	var stats struct {
		Total       int
		Pending     int
		Reprocessed int
		Discarded   int
	}

	scanErr := db.QueryRow(query).Scan(&stats.Total, &stats.Pending, &stats.Reprocessed, &stats.Discarded)
	if scanErr != nil {
		return nil, fmt.Errorf("failed to get DLQ stats: %w", scanErr)
	}

	return map[string]int{
		"total":       stats.Total,
		"pending":     stats.Pending,
		"reprocessed": stats.Reprocessed,
		"discarded":   stats.Discarded,
	}, nil
}
