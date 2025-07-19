package ingestion

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/your-username/click-lite-log-analytics/backend/internal/database"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
)

// BatchProcessor handles batching of logs for efficient writes
type BatchProcessor struct {
	db           *database.DB
	batchSize    int
	flushInterval time.Duration
	buffer       []models.Log
	bufferMu     sync.Mutex
	flushChan    chan struct{}
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(db *database.DB, batchSize int, flushInterval time.Duration) *BatchProcessor {
	bp := &BatchProcessor{
		db:            db,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		buffer:        make([]models.Log, 0, batchSize),
		flushChan:     make(chan struct{}, 1),
		stopChan:      make(chan struct{}),
	}
	
	bp.wg.Add(1)
	go bp.run()
	
	return bp
}

// Add adds a log to the batch
func (bp *BatchProcessor) Add(log models.Log) {
	bp.bufferMu.Lock()
	bp.buffer = append(bp.buffer, log)
	shouldFlush := len(bp.buffer) >= bp.batchSize
	bp.bufferMu.Unlock()
	
	if shouldFlush {
		select {
		case bp.flushChan <- struct{}{}:
		default:
		}
	}
}

// AddBatch adds multiple logs to the batch
func (bp *BatchProcessor) AddBatch(logs []models.Log) {
	bp.bufferMu.Lock()
	bp.buffer = append(bp.buffer, logs...)
	shouldFlush := len(bp.buffer) >= bp.batchSize
	bp.bufferMu.Unlock()
	
	if shouldFlush {
		select {
		case bp.flushChan <- struct{}{}:
		default:
		}
	}
}

// run is the main processing loop
func (bp *BatchProcessor) run() {
	defer bp.wg.Done()
	
	ticker := time.NewTicker(bp.flushInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-bp.stopChan:
			bp.flush()
			return
		case <-ticker.C:
			bp.flush()
		case <-bp.flushChan:
			bp.flush()
		}
	}
}

// flush writes the current batch to the database
func (bp *BatchProcessor) flush() {
	bp.bufferMu.Lock()
	if len(bp.buffer) == 0 {
		bp.bufferMu.Unlock()
		return
	}
	
	// Copy buffer and reset
	batch := make([]models.Log, len(bp.buffer))
	copy(batch, bp.buffer)
	bp.buffer = bp.buffer[:0]
	bp.bufferMu.Unlock()
	
	// Write batch with retries
	ctx := context.Background()
	maxRetries := 3
	backoff := time.Second
	
	for i := 0; i < maxRetries; i++ {
		if err := bp.writeBatch(ctx, batch); err != nil {
			log.Error().Err(err).Int("attempt", i+1).Int("batch_size", len(batch)).Msg("Failed to write batch")
			if i < maxRetries-1 {
				time.Sleep(backoff)
				backoff *= 2
			}
			continue
		}
		log.Info().Int("batch_size", len(batch)).Msg("Successfully wrote batch")
		return
	}
	
	log.Error().Int("batch_size", len(batch)).Msg("Failed to write batch after all retries")
}

// writeBatch writes a batch of logs to the database
func (bp *BatchProcessor) writeBatch(ctx context.Context, batch []models.Log) error {
	for _, logEntry := range batch {
		if err := bp.db.InsertLog(ctx, &logEntry); err != nil {
			return err
		}
	}
	return nil
}

// Stop gracefully shuts down the batch processor
func (bp *BatchProcessor) Stop() {
	close(bp.stopChan)
	bp.wg.Wait()
}