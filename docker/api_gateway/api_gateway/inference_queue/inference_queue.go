package inference_queue

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"api_gateway/logger"
)

type QueueItem[T any] struct {
	Item T     // The item being stored in the queue
	Size int64 // Size in bytes of the marshaled item
}

// InferenceQueue implements a queue for items with in-memory and disk-spill capabilities.
type InferenceQueue[T any] struct {
	mu                 sync.Mutex     // Mutex for protecting access to queue fields
	inMemoryQueue      []QueueItem[T] // In-memory buffer for items
	currentMemoryBytes atomic.Int64   // Current byte size of items in inMemoryQueue
	maxMemoryBytes     int64          // Maximum allowed byte size in inMemoryQueue before spilling to disk
	maxBatchSize       int            // Maximum items to return in a single batch

	// Fields for managing the current temporary file being WRITTEN to
	tempFile     *os.File      // The current temporary file for writing spilled items
	tempFilePath string        // Path to the current temporary file
	writer       *bufio.Writer // Writer for the temporary file
	hasSpilled   atomic.Bool   // Flag to indicate if items have spilled to this temp file

	// Fields for managing the current temporary file being READ from
	reader          *bufio.Reader // Reader for consuming a temporary file
	readingFile     *os.File      // The *os.File handle for the current file being read by 'reader'
	readingFilePath string        // Path to the current temporary file being read

	totalQueuedItems atomic.Int64 // Total count of items across memory and disk

	cond *sync.Cond // Conditional variable for signaling consumer when items are available

	isClosed atomic.Bool // Flag to indicate if the queue is being closed
}

// NewInferenceQueue creates and initializes a new InferenceQueue.
// maxMemoryBytes: The maximum memory (in bytes) to use before spilling to a temporary file.
// maxBatchSize: The maximum number of items to return in a single GetBatch call.
func NewInferenceQueue[T any](maxMemoryBytes int64, maxBatchSize int) (*InferenceQueue[T], error) {
	if maxMemoryBytes <= 0 || maxBatchSize <= 0 {
		return nil, fmt.Errorf("maxMemoryBytes and maxBatchSize must be positive")
	}

	q := &InferenceQueue[T]{
		inMemoryQueue:  make([]QueueItem[T], 0),
		maxMemoryBytes: maxMemoryBytes,
		maxBatchSize:   maxBatchSize,
	}
	q.cond = sync.NewCond(&q.mu)

	// Create the initial temporary file for potential spills
	file, err := os.CreateTemp("", "inference-queue-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	q.tempFile = file
	q.tempFilePath = file.Name()
	q.writer = bufio.NewWriter(file)

	logger.Infof("Queue initialized. Initial temp file for writing: %s\n", q.tempFilePath)
	return q, nil
}

// waitForItemsOrShutdown blocks until there are items available in the queue or the queue is closed.
// It checks the total queued items and the closed state of the queue.
// Returns io.EOF if the queue is closed and empty, indicating no more items will arrive.
func (q *InferenceQueue[T]) waitForItemsOrShutdown() error {
	// Wait if the queue is empty AND not closed.
	// This makes the consumer block until items are available or the queue is explicitly closed.
	for q.totalQueuedItems.Load() == 0 && !q.isClosed.Load() {
		q.cond.Wait()
	}

	// If the queue is closed and all items have been processed, return io.EOF.
	if q.isClosed.Load() && q.totalQueuedItems.Load() == 0 {
		return io.EOF
	}

	return nil // No error, items are available or queue is closed
}

// consumeFromMemory attempts to fill a batch from the in-memory queue.
// It consumes items until the batch is full or there are no more items in memory.
// This function is called by GetBatch to prioritize in-memory items before reading from disk.
// It modifies the provided batch slice directly.
// It also updates the current memory usage and total queued items accordingly.
func (q *InferenceQueue[T]) consumeFromMemory(batch *[]T) {
	// Consume items from in-memory queue until batch is full or memory is exhausted
	for i := 0; i < q.maxBatchSize && len(q.inMemoryQueue) > 0; i++ {
		item := q.inMemoryQueue[0]
		q.inMemoryQueue = q.inMemoryQueue[1:] // Remove the item from the front of the queue

		q.currentMemoryBytes.Add(-item.Size) // Decrement memory usage
		q.totalQueuedItems.Add(-1)           // Decrement total queued items

		*batch = append(*batch, item.Item)
	}
}

func (q *InferenceQueue[T]) consumeFromDisk(batch *[]T) error {
	// Now, if q.reader is not nil (either was already active, or just became active above)
	if q.reader == nil {
		return nil
	}

	// Read items from the active temporary file until batch is full or file is exhausted
	for len(*batch) < q.maxBatchSize || q.currentMemoryBytes.Load() < q.maxMemoryBytes {
		line, err := q.reader.ReadBytes('\n') // Read until newline delimiter
		if err == io.EOF {
			// End of file reached for the current temporary file.
			// Close and delete this file.
			if q.readingFile != nil {
				if closeErr := q.readingFile.Close(); closeErr != nil {
					logger.Warningf("Failed to close consumed temp file %s: %v\n", q.readingFilePath, closeErr)
				}
				if removeErr := os.Remove(q.readingFilePath); removeErr != nil {
					logger.Warningf("Failed to delete consumed temp file %s: %v\n", q.readingFilePath, removeErr)
				}
			}
			q.reader = nil         // Clear reader
			q.readingFile = nil    // Clear file handle
			q.readingFilePath = "" // Clear path
			break                  // Stop reading from disk, move to next logic if needed
		}
		if err != nil {
			return fmt.Errorf("failed to read from temporary file %s: %w", q.readingFilePath, err)
		}

		item := QueueItem[T]{}
		if err := json.Unmarshal(bytes.TrimSuffix(line, []byte{'\n'}), &item); err != nil {
			return fmt.Errorf("failed to unmarshal item from temporary file %s: %w", q.readingFilePath, err)
		}

		// Decide where to store: batch first, then pre-fill memory
		if len(*batch) < q.maxBatchSize {
			*batch = append(*batch, item.Item)
			q.totalQueuedItems.Add(-1)
		} else if q.currentMemoryBytes.Load()+item.Size <= q.maxMemoryBytes {
			q.inMemoryQueue = append(q.inMemoryQueue, item)
			q.currentMemoryBytes.Add(item.Size)
		} else {
			// No more space to pre-fill memory
			break
		}
	}
	return nil
}

func (q *InferenceQueue[T]) prepareReader() error {
	// If we are not currently reading from a disk file, but items have spilled to the current writing file.
	// This means we need to switch the current writing file to a reading file.
	if q.reader != nil || !q.hasSpilled.Load() {
		return nil // No need to prepare reader if we are already reading from a file
	}

	// Ensure all pending writes to the current file are flushed and the file is closed.
	if q.writer != nil {
		if err := q.writer.Flush(); err != nil {
			return fmt.Errorf("failed to flush writer before switching to read from disk: %w", err)
		}
		if err := q.tempFile.Close(); err != nil {
			return fmt.Errorf("failed to close temp file before switching to read: %w", err)
		}
	}

	// Store the path of the file we just closed (which now contains all spilled items)
	q.readingFilePath = q.tempFilePath
	logger.Infof("Switching from writing to reading temp file: %s (Remaining total queued: %d)\n", q.readingFilePath, q.totalQueuedItems.Load())

	// 2. Create a NEW temporary file immediately for any subsequent pushes.
	// This is crucial to ensure that new pushes go to a clean file while the old one is being consumed.
	newFile, err := os.CreateTemp("", "inference-queue-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create new temporary file for subsequent writes: %w", err)
	}
	q.tempFile = newFile
	q.tempFilePath = newFile.Name()
	q.writer = bufio.NewWriter(newFile)
	q.hasSpilled.Store(false) // Reset spilled flag for the new file's lifecycle
	logger.Infof("Created new temp file for future writes: %s\n", q.tempFilePath)

	// 3. Open the OLD file (the one that was just filled with spilled items) for reading.
	readFd, err := os.Open(q.readingFilePath)
	if err != nil {
		return fmt.Errorf("failed to open old temp file for reading (%s): %w", q.readingFilePath, err)
	}
	q.readingFile = readFd
	q.reader = bufio.NewReader(readFd)
	return nil
}

// FinalizeWriter ensures the writer is flushed and its file closed, preserving the file for later reading.
func (q *InferenceQueue[T]) finalizeWriter() error {
	if q.writer != nil {
		if err := q.writer.Flush(); err != nil {
			return fmt.Errorf("failed to flush writer in FinalizeWriter: %w", err)
		}
	}
	if q.tempFile != nil {
		if err := q.tempFile.Close(); err != nil {
			return fmt.Errorf("failed to close temp file in FinalizeWriter: %w", err)
		}
		// Do NOT remove the tempFilePath - let GetBatch() rotate to it
	}
	// Clear writer handles so we know it's finalized
	q.writer = nil
	q.tempFile = nil
	return nil
}

// Push adds a item to the queue.
// It tries to store the item in memory first. If memory is full, or if the queue is
// currently in a "spilled" state (meaning the previous temp file is active for writing),
// it writes the item to the temporary file.
// Returns an error if the queue is closed or if marshaling/writing fails.
func (q *InferenceQueue[T]) Push(item T) error {
	// Check if the queue is closed before proceeding
	if q.isClosed.Load() {
		return fmt.Errorf("queue is closed, cannot push items")
	}

	// Marshal the item to JSON to estimate its size for memory tracking
	itemJSON, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}
	itemSize := int64(len(itemJSON))

	q.mu.Lock()
	defer q.mu.Unlock()

	// Decide whether to store in memory or spill to disk
	// Condition for in-memory storage:
	// 1. Current memory usage + new item size does not exceed maxMemoryBytes
	// 2. The queue is NOT currently in a "spilled" state (i.e., we are not actively writing to the temp file
	//    that has been designated for new spills). This ensures memory is prioritized unless explicit spill is happening.
	if q.currentMemoryBytes.Load()+itemSize <= q.maxMemoryBytes && !q.hasSpilled.Load() {
		q.inMemoryQueue = append(q.inMemoryQueue, QueueItem[T]{Item: item, Size: itemSize})
		q.currentMemoryBytes.Add(itemSize)
		logger.Debugf("Pushed to memory. Current memory bytes: %d/%d\n", q.currentMemoryBytes.Load(), q.maxMemoryBytes)
	} else {
		// Memory is full, or we've previously spilled and are still using the temp file for writing.
		// Write the item to the temporary file.
		q.hasSpilled.Store(true) // Mark that items are now spilling to disk
		if _, err := q.writer.Write(itemJSON); err != nil {
			return fmt.Errorf("failed to write to temporary file: %w", err)
		}
		if _, err := q.writer.WriteRune('\n'); err != nil { // Add newline delimiter for easier reading
			return fmt.Errorf("failed to write newline to temporary file: %w", err)
		}
		// Flush to ensure data is written to disk promptly. This is important before a potential read operation.
		if err := q.writer.Flush(); err != nil {
			return fmt.Errorf("failed to flush temporary file writer: %w", err)
		}
		logger.Infof("Pushed to disk (%s). Item size: %d bytes. Current memory bytes: %d/%d\n", q.tempFilePath, itemSize, q.currentMemoryBytes.Load(), q.maxMemoryBytes) // For debugging
	}

	q.totalQueuedItems.Add(1) // Increment total count of items in the queue
	q.cond.Signal()           // Signal one waiting consumer that an item is available
	return nil
}

// PushBatch adds a slice of items to the queue efficiently.
// It tries to store items in memory first. If memory is full, or if the queue is
// currently in a "spilled" state, it writes items to the temporary file.
// Returns an error if the queue is closed or if marshaling/writing fails for any item.
func (q *InferenceQueue[T]) PushBatch(items []T) error {
	if q.isClosed.Load() {
		return fmt.Errorf("queue is closed, cannot push items")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	var flushed bool // Track if writer was flushed in this batch

	for _, item := range items {
		itemJSON, err := json.Marshal(item)
		if err != nil {
			// If marshaling fails for one item, we might choose to return an error
			// immediately or log it and continue. For robustness, returning is safer.
			return fmt.Errorf("failed to marshal item in batch: %w", err)
		}
		itemSize := int64(len(itemJSON))

		// Check if the item fits in memory without exceeding the limit AND if we haven't already spilled
		// to the current writing file. This ensures memory is prioritized first.
		if q.currentMemoryBytes.Load()+itemSize <= q.maxMemoryBytes && !q.hasSpilled.Load() {
			q.inMemoryQueue = append(q.inMemoryQueue, QueueItem[T]{Item: item, Size: itemSize})
			q.currentMemoryBytes.Add(itemSize)
			logger.Debugf("Pushed to memory in batch. Current memory bytes: %d/%d\n", q.currentMemoryBytes.Load(), q.maxMemoryBytes)
		} else {
			// Memory is full, or we've previously spilled and are still using the temp file for writing.
			// Write the item to the temporary file.
			q.hasSpilled.Store(true) // Mark that items are now spilling to disk
			if _, err := q.writer.Write(itemJSON); err != nil {
				return fmt.Errorf("failed to write to temporary file in batch: %w", err)
			}
			if _, err := q.writer.WriteRune('\n'); err != nil { // Add newline delimiter
				return fmt.Errorf("failed to write newline to temporary file in batch: %w", err)
			}
			flushed = true // Indicate that a flush is needed at the end of the batch
			logger.Debugf("Pushed to disk in batch (%s). Item size: %d bytes. Current memory bytes: %d/%d\n", q.tempFilePath, itemSize, q.currentMemoryBytes.Load(), q.maxMemoryBytes)
		}
		q.totalQueuedItems.Add(1) // Increment total count for each item pushed
	}

	// Flush the writer once at the end of the batch if any items were written to disk.
	if flushed {
		if err := q.writer.Flush(); err != nil {
			return fmt.Errorf("failed to flush temporary file writer after batch: %w", err)
		}
	}

	q.cond.Signal() // Signal consumers once for the entire batch push operation
	return nil
}

// GetBatch retrieves a batch of items from the queue.
// It prioritizes items in memory. If memory is empty, it attempts to read from
// the temporary file currently being read from. If that file is exhausted,
// or if no file is currently being read, it will transition the current writing
// file to a reading file and create a new writing file.
// Returns io.EOF if the queue is closed and empty, indicating no more items will arrive.
func (q *InferenceQueue[T]) GetBatch() ([]T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Wait for items to be available or for the queue to be closed
	if err := q.waitForItemsOrShutdown(); err != nil {
		if err == io.EOF {
			// If the queue is closed and empty, return EOF to signal no more items will arrive
			return nil, io.EOF
		}
		return nil, fmt.Errorf("error waiting for items or shutdown: %w", err)
	}

	// Prepare a batch to return
	batch := make([]T, 0, q.maxBatchSize)

	// First, try to consume items from the in-memory queue
	q.consumeFromMemory(&batch)

	// If the batch is not yet full AND there are still items queued (which implies they are on disk)
	if len(batch) < q.maxBatchSize && q.totalQueuedItems.Load() > 0 {
		// Prepare the reader if not already done
		if err := q.prepareReader(); err != nil {
			return nil, fmt.Errorf("failed to prepare reader for disk items: %w", err)
		}

		// Now, try to consume items from the disk file
		if err := q.consumeFromDisk(&batch); err != nil {
			return nil, fmt.Errorf("failed to consume items from disk: %w", err)
		}
	}

	return batch, nil
}

// Close cleans up the temporary files and signals consumers to stop.
// It should be called when the application is shutting down to ensure proper resource release.
func (q *InferenceQueue[T]) Close() error {
	// Idempotent check: if already closed, do nothing
	if !q.isClosed.CompareAndSwap(false, true) {
		return nil
	}
	q.cond.Broadcast() // Wake up any waiting consumers so they can check the closed flag

	q.mu.Lock()
	defer q.mu.Unlock()

	var err error
	// If we have a writer, finalize it to ensure all data is flushed and the file is closed.
	if err := q.finalizeWriter(); err != nil {
		return fmt.Errorf("error finalizing writer during close: %w", err)
	}

	// Close and remove the temporary file that was potentially being read from (q.readingFile)
	if q.readingFile != nil {
		if closeErr := q.readingFile.Close(); closeErr != nil {
			if err == nil {
				err = fmt.Errorf("failed to close reading temp file on close: %w", closeErr)
			} else {
				err = fmt.Errorf("%s; failed to close reading temp file on close: %w", err.Error(), closeErr)
			}
		}
		if removeErr := os.Remove(q.readingFilePath); removeErr != nil && !os.IsNotExist(removeErr) {
			if err == nil {
				err = fmt.Errorf("failed to remove reading temp file on close: %w", removeErr)
			} else {
				err = fmt.Errorf("%s; failed to remove reading temp file on close: %w", err.Error(), removeErr)
			}
		}
	}
	return err
}

// Reset clears the in-memory queue and resets counters.
// Any unconsumed disk-spilled data will remain — make sure the queue is empty before calling this.
func (q *InferenceQueue[T]) Reset() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.totalQueuedItems.Load() > 0 {
		return fmt.Errorf("cannot reset queue: %d items still in queue", q.totalQueuedItems.Load())
	}

	q.inMemoryQueue = q.inMemoryQueue[:0]
	q.currentMemoryBytes.Store(0)
	q.hasSpilled.Store(false)

	// Clean up any leftover reading file
	if q.reader != nil {
		q.reader.Reset(nil)
		q.reader = nil
	}
	if q.readingFile != nil {
		if err := q.readingFile.Close(); err != nil {
			logger.Warningf("Failed to close reading file during reset: %v", err)
			return fmt.Errorf("failed to close reading file during reset: %w", err)
		}
		q.readingFile = nil
	}
	if q.readingFilePath != "" {
		if err := os.Remove(q.readingFilePath); err != nil && !os.IsNotExist(err) {
			logger.Warningf("Failed to remove reading file during reset: %v", err)
			return fmt.Errorf("failed to remove reading file during reset: %w", err)
		}
		q.readingFilePath = ""
	}

	return nil
}
