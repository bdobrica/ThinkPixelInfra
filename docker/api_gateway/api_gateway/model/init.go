package model

import (
	"sync"

	"api_gateway/config"
	"api_gateway/model_queue"
)

var (
	queuePool    sync.Pool
	batchMaxSize int
)

// initModelQueue initializes a new inference queue for TextItem.
func initModelQueue() (*model_queue.ModelQueue[TextItem], error) {
	queueMaxMemoryBytes := int64(config.GetEnvByteSize("API_GATEWAY_MODEL_QUEUE_MAX_MEMORY", 52_428_800)) // 50 MB default
	batchMaxSize = config.GetEnvInt("API_GATEWAY_MODEL_BATCH_MAX_SIZE", 10)

	return model_queue.NewModelQueue[TextItem](queueMaxMemoryBytes, batchMaxSize)
}

// init initializes the queuePool for inference queues.
func init() {
	queuePool = sync.Pool{
		New: func() any {
			queue, err := initModelQueue()
			if err != nil {
				panic("Failed to create inference queue: " + err.Error())
			}
			return queue
		},
	}
}
