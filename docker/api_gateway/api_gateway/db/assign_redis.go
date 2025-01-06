package db

import (
	"database/sql"
	"errors"
	"fmt"

	"api_gateway/logger"
)

// AssignToRedisMaster tries to find an active Redis master
// with enough free capacity to handle the requestedMemoryBytes.
// If found, it assigns the node by creating a record in wp_client_requests
// and updating the master’s assigned_memory_bytes.
// If not found, it logs an error, inserts a 'rejected' (or 'pending') request,
// but does NOT fail overall activation.
//
// It uses a transaction + SELECT FOR UPDATE for concurrency safety.
func AssignToRedisMaster(apiKeyID int, requestedMemoryBytes uint64) error {
	dbConn, err := GetDBConnection()
	if err != nil {
		return err
	}

	// Start a transaction.
	tx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	defer func() {
		// If we exit the function via a panic or an error,
		// we want to make sure the transaction is rolled back.
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // re-throw panic after rollback
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// 1) Attempt to find a suitable master with enough remaining capacity.
	//    We do ORDER BY so we pick the "best fit" or "first fit" approach.
	//    The FOR UPDATE ensures we lock the row until the end of the transaction.
	query := `
        SELECT id, assigned_memory_bytes, master_name, sentinel_name
        FROM wp_redis_masters
        WHERE status = 'active'
          AND max_capacity_bytes - GREATEST(used_memory_bytes, assigned_memory_bytes) >= ?
        ORDER BY (max_capacity_bytes - GREATEST(used_memory_bytes, assigned_memory_bytes)) ASC
        LIMIT 1
        FOR UPDATE
    `
	row := tx.QueryRow(query, requestedMemoryBytes)

	var (
		masterID, currentAssigned int
		masterName, sentinelName  string
	)
	if scanErr := row.Scan(&masterID, &currentAssigned, &masterName, &sentinelName); scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			// No capacity found on any active node. Log, but do not fail activation.
			logger.Warningf("[WARN] No Redis master node can accommodate %d bytes for API key %d",
				requestedMemoryBytes, apiKeyID,
			)

			// Create a wp_client_requests record with 'rejected' (or 'pending') status
			// to indicate the request was made but could not be fulfilled.
			insertReq := `
                INSERT INTO wp_client_requests (api_key_id, requested_memory_bytes, status)
                VALUES (?, ?, 'rejected')
            `
			if _, err = tx.Exec(insertReq, apiKeyID, requestedMemoryBytes); err != nil {
				logger.Errorf("Failed to insert rejected client request: %v", err)
				return err
			}

			// Do not return an error here, because we don't want to fail the overall activation.
			return nil
		}
		// Other DB errors
		logger.Errorf("Failed to scan Redis master node: %v", scanErr)
		return scanErr
	}

	// 2) We found a master. Update its assigned_memory_bytes to reserve capacity for this request.
	updateMaster := `
        UPDATE wp_redis_masters
        SET assigned_memory_bytes = assigned_memory_bytes + ?
        WHERE id = ?
    `
	if _, err = tx.Exec(updateMaster, requestedMemoryBytes, masterID); err != nil {
		logger.Errorf("Failed to update assigned_memory_bytes on Redis master %d: %v", masterID, err)
		return err
	}

	// 3) Insert the client request record, marking it as 'assigned'.
	insertReq := `
        INSERT INTO wp_client_requests (
            api_key_id,
            requested_memory_bytes,
            status,
            assigned_node_id,
            assigned_at
        ) VALUES (?, ?, 'assigned', ?, NOW())
    `
	if _, err = tx.Exec(insertReq, apiKeyID, requestedMemoryBytes, masterID); err != nil {
		logger.Errorf("Failed to insert assigned client request: %v", err)
		return err
	}

	// 4) Update the ApiKey record to reflect the assigned node.
	updateKey := `
        UPDATE wp_api_keys
        SET redis_server = ?
        WHERE id = ?
    `
	redisServer := fmt.Sprintf("%s:%d/%s", sentinelName, 26379, masterName)
	if _, err = tx.Exec(updateKey, redisServer, apiKeyID); err != nil {
		logger.Errorf("Failed to update Redis server on API key %d: %v", apiKeyID, err)
		return err
	}

	// Transaction will commit in deferred function if there are no errors.
	return nil
}
