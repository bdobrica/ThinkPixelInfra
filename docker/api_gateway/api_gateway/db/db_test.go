package db

import (
	"regexp"
	"sync"
	"testing"

	"api_gateway/utils"

	"github.com/DATA-DOG/go-sqlmock"
)

func setupMockDB(t *testing.T) sqlmock.Sqlmock {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}

	dbInstance = sqlDB
	initOnce = sync.Once{}
	initOnce.Do(func() {})

	t.Cleanup(func() {
		dbInstance = nil
		initOnce = sync.Once{}
		_ = sqlDB.Close()
	})

	return mock
}

func TestActivateAPIKeySkipsAssignmentWhenIndexingNodeExists(t *testing.T) {
	mock := setupMockDB(t)

	const (
		domain = "example.com"
		path   = "/docs"
		apiKey = "test-api-key"
	)

	hashedAPIKey := utils.HashString(apiKey)

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE wp_thinkpixel_sites
		SET api_key = ?, status = 'active', updated_at = NOW(), verified_at = NOW(), expires_at = ?
		WHERE domain = ? AND path = ? AND validation_status = 'verified'`)).
		WithArgs(hashedAPIKey, sqlmock.AnyArg(), domain, path).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, estimated_pages, average_page_size, st_dev_page_size, indexing_node
		FROM wp_thinkpixel_sites
		WHERE api_key = ?
		LIMIT 1
	`)).
		WithArgs(hashedAPIKey).
		WillReturnRows(sqlmock.NewRows([]string{"id", "estimated_pages", "average_page_size", "st_dev_page_size", "indexing_node"}).
			AddRow(42, 100, 2048, 256, "redis-sentinel-01:26379/redis-master-01"))

	if err := ActivateAPIKey(domain, path, apiKey); err != nil {
		t.Fatalf("ActivateAPIKey() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestActivateAPIKeyAssignsNodeWhenIndexingNodeMissing(t *testing.T) {
	mock := setupMockDB(t)

	const (
		domain           = "example.com"
		path             = "/docs"
		apiKey           = "test-api-key"
		siteID           = 42
		estimatedPages   = 100
		averagePageSize  = 2048
		stDevPageSize    = 256
		assignedNodeID   = 7
		assignedNodeName = "redis-master-01"
		assignedSentinel = "redis-sentinel-01"
		indexingNodeType = "redis"
	)

	hashedAPIKey := utils.HashString(apiKey)
	requestedBytes := utils.EstimateMemory(estimatedPages, averagePageSize, stDevPageSize)
	if requestedBytes == 0 {
		t.Fatal("requestedBytes should be greater than zero")
	}

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE wp_thinkpixel_sites
		SET api_key = ?, status = 'active', updated_at = NOW(), verified_at = NOW(), expires_at = ?
		WHERE domain = ? AND path = ? AND validation_status = 'verified'`)).
		WithArgs(hashedAPIKey, sqlmock.AnyArg(), domain, path).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, estimated_pages, average_page_size, st_dev_page_size, indexing_node
		FROM wp_thinkpixel_sites
		WHERE api_key = ?
		LIMIT 1
	`)).
		WithArgs(hashedAPIKey).
		WillReturnRows(sqlmock.NewRows([]string{"id", "estimated_pages", "average_page_size", "st_dev_page_size", "indexing_node"}).
			AddRow(siteID, estimatedPages, averagePageSize, stDevPageSize, nil))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT indexing_node_type
		FROM wp_thinkpixel_sites
		WHERE id = ?
	`)).
		WithArgs(siteID).
		WillReturnRows(sqlmock.NewRows([]string{"indexing_node_type"}).AddRow(indexingNodeType))

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, assigned_memory_bytes, node_name, sentinel_name
		FROM wp_thinkpixel_index_nodes
		WHERE status = 'active'
		  AND node_type = 'redis'
		  AND max_capacity_bytes - GREATEST(used_memory_bytes, assigned_memory_bytes) >= ?
		ORDER BY (max_capacity_bytes - GREATEST(used_memory_bytes, assigned_memory_bytes)) ASC
		LIMIT 1
		FOR UPDATE
	`)).
		WithArgs(int64(requestedBytes)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "assigned_memory_bytes", "node_name", "sentinel_name"}).
			AddRow(assignedNodeID, 1024, assignedNodeName, assignedSentinel))

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE wp_thinkpixel_index_nodes
		SET assigned_memory_bytes = assigned_memory_bytes + ?
		WHERE id = ?
	`)).
		WithArgs(int64(requestedBytes), assignedNodeID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO wp_thinkpixel_index_requests (
		    site_id,
		    requested_memory_bytes,
		    status,
		    assigned_node_id,
		    assigned_at
		) VALUES (?, ?, 'assigned', ?, NOW())
	`)).
		WithArgs(siteID, int64(requestedBytes), assignedNodeID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE wp_thinkpixel_sites
		SET indexing_node = ?
		WHERE id = ?
	`)).
		WithArgs(assignedSentinel+":26379/"+assignedNodeName, siteID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	if err := ActivateAPIKey(domain, path, apiKey); err != nil {
		t.Fatalf("ActivateAPIKey() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestActivateAPIKeyAssignsQdrantNodeWhenIndexingNodeMissing(t *testing.T) {
	mock := setupMockDB(t)

	const (
		domain           = "example.com"
		path             = "/docs"
		apiKey           = "test-api-key"
		siteID           = 43
		estimatedPages   = 120
		averagePageSize  = 1024
		stDevPageSize    = 128
		assignedNodeID   = 8
		assignedNodeName = "qdrant-node-01"
		indexingNodeType = "qdrant"
	)

	hashedAPIKey := utils.HashString(apiKey)
	requestedBytes := utils.EstimateMemory(estimatedPages, averagePageSize, stDevPageSize)
	if requestedBytes == 0 {
		t.Fatal("requestedBytes should be greater than zero")
	}

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE wp_thinkpixel_sites
		SET api_key = ?, status = 'active', updated_at = NOW(), verified_at = NOW(), expires_at = ?
		WHERE domain = ? AND path = ? AND validation_status = 'verified'`)).
		WithArgs(hashedAPIKey, sqlmock.AnyArg(), domain, path).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, estimated_pages, average_page_size, st_dev_page_size, indexing_node
		FROM wp_thinkpixel_sites
		WHERE api_key = ?
		LIMIT 1
	`)).
		WithArgs(hashedAPIKey).
		WillReturnRows(sqlmock.NewRows([]string{"id", "estimated_pages", "average_page_size", "st_dev_page_size", "indexing_node"}).
			AddRow(siteID, estimatedPages, averagePageSize, stDevPageSize, nil))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT indexing_node_type
		FROM wp_thinkpixel_sites
		WHERE id = ?
	`)).
		WithArgs(siteID).
		WillReturnRows(sqlmock.NewRows([]string{"indexing_node_type"}).AddRow(indexingNodeType))

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, assigned_memory_bytes, node_name
		FROM wp_thinkpixel_index_nodes
		WHERE status = 'active'
		  AND node_type = 'qdrant'
		  AND max_capacity_bytes - GREATEST(used_memory_bytes, assigned_memory_bytes) >= ?
		ORDER BY (max_capacity_bytes - GREATEST(used_memory_bytes, assigned_memory_bytes)) ASC
		LIMIT 1
		FOR UPDATE
	`)).
		WithArgs(int64(requestedBytes)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "assigned_memory_bytes", "node_name"}).
			AddRow(assignedNodeID, 512, assignedNodeName))

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE wp_thinkpixel_index_nodes
		SET assigned_memory_bytes = assigned_memory_bytes + ?
		WHERE id = ?
	`)).
		WithArgs(int64(requestedBytes), assignedNodeID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO wp_thinkpixel_index_requests (
		    site_id,
		    requested_memory_bytes,
		    status,
		    assigned_node_id,
		    assigned_at
		) VALUES (?, ?, 'assigned', ?, NOW())
	`)).
		WithArgs(siteID, int64(requestedBytes), assignedNodeID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE wp_thinkpixel_sites
		SET indexing_node = ?
		WHERE id = ?
	`)).
		WithArgs(assignedNodeName+":6334", siteID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	if err := ActivateAPIKey(domain, path, apiKey); err != nil {
		t.Fatalf("ActivateAPIKey() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
