package db

import (
	"api_gateway/config"
	"fmt"
	"strconv"
	"strings"
)

func getIndexingNodeTypeById(siteId int) (string, error) {
	dbConn, err := GetDBConnection()
	if err != nil {
		return "", err
	}

	query := `
        SELECT indexing_node_type
        FROM wp_thinkpixel_sites
        WHERE id = ?
    `
	row := dbConn.QueryRow(query, siteId)

	var indexingNodeType string
	if err := row.Scan(&indexingNodeType); err != nil {
		return "", err
	}

	return strings.ToLower(indexingNodeType), nil
}

func AssignToIndexingNode(siteId int, requestedMemoryBytes uint64) error {
	// Get the maximum memory allowed for the site from environment variable
	siteMaxMemory := config.GetEnv("API_GATEWAY_SITE_MAX_MEMORY", "1000000000")
	siteMaxMemoryBytes, err := strconv.ParseUint(siteMaxMemory, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid site max memory %s: %w", siteMaxMemory, err)
	}

	// Check if the requested memory is within the allowed range
	if requestedMemoryBytes < 1 || requestedMemoryBytes > siteMaxMemoryBytes {
		return fmt.Errorf("requested memory %d is out of range", requestedMemoryBytes)
	}

	// Check if the site ID is valid
	if siteId <= 0 {
		return fmt.Errorf("invalid site ID %d", siteId)
	}

	// Get the node type for the given site ID
	indexingNodeType, err := getIndexingNodeTypeById(siteId)
	if err != nil {
		return fmt.Errorf("failed to get node type for site ID %d: %w", siteId, err)
	}

	switch indexingNodeType {
	case "qdrant":
		return assignToQdrantIndexingNode(siteId, requestedMemoryBytes)
	case "redis":
		return assignToRedisIndexingNode(siteId, requestedMemoryBytes)
	default:
		return fmt.Errorf("unknown node type %s for site ID %d", indexingNodeType, siteId)
	}
}
