package qdrant

import (
	"api_gateway/config"
	"api_gateway/logger"
	"net/url"
	"strconv"

	"github.com/qdrant/go-client/qdrant"
)

// QdrantClient now wraps the official qdrant.Client.
type QdrantClient struct {
	client *qdrant.Client
}

// QdrantClientPool holds a buffered channel of QdrantClient pointers.
type QdrantClientPool struct {
	size int
	pool chan *QdrantClient
}

// NewQdrantClient creates a new QdrantClient using the official client.
func NewQdrantClient(baseURL, apiKey string) (*QdrantClient, error) {
	// Parse the baseURL to extract the Host, Port, and Protocol
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	host := parsedURL.Hostname()
	portRaw := parsedURL.Port()
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		port = 6333
	}
	protocol := parsedURL.Scheme
	if protocol == "" {
		protocol = "https"
	}

	// Log the extracted values
	logger.Infof("Parsed Qdrant Client URL - Host: %s, Port: %s, Protocol: %s", host, port, protocol)

	cfg := qdrant.Config{
		Host:   host,
		Port:   port,
		UseTLS: protocol == "https",
		APIKey: apiKey,
	}
	c, err := qdrant.NewClient(&cfg)
	if err != nil {
		return nil, err
	}
	return &QdrantClient{client: c}, nil
}

// NewQdrantClientPool creates a new pool with the given size, baseURL, and apiKey.
func NewQdrantClientPool(size int, baseURL, apiKey string) *QdrantClientPool {
	p := &QdrantClientPool{
		size: 0,
		pool: make(chan *QdrantClient, size),
	}
	// Prepopulate the pool.
	for i := 0; i < size; i++ {
		client, err := NewQdrantClient(baseURL, apiKey)
		if err != nil {
			// You may choose to log the error and/or panic here.
			continue
		}
		p.pool <- client
		p.size++
	}
	return p
}

// Get retrieves a QdrantClient from the pool.
func (p *QdrantClientPool) Get() *QdrantClient {
	return <-p.pool
}

// Put returns a QdrantClient back to the pool.
func (p *QdrantClientPool) Put(client *QdrantClient) {
	// Optionally, add timeout or error handling here.
	p.pool <- client
}

// defaultPool holds the client pool used for all operations.
var defaultPool *QdrantClientPool

// init initializes the default client pool.
func init() {
	var apiKey = config.GetEnv("API_GATEWAY_QDRANT_API_KEY", "")
	var baseURL = config.GetEnv("API_GATEWAY_QDRANT_BASE_URL", "")
	var poolSizeRaw = config.GetEnv("API_GATEWAY_QDRANT_POOL_SIZE", "3")

	poolSize, err := strconv.Atoi(poolSizeRaw)
	if err != nil {
		logger.Warningf("Invalid API_GATEWAY_QDRANT_POOL_SIZE: %s. Using default pool size.", poolSizeRaw)
		poolSize = 3
	}

	defaultPool = NewQdrantClientPool(poolSize, baseURL, apiKey)
	if defaultPool.size == 0 {
		logger.Warningf("Failed to create Qdrant client pool")
	}
}
