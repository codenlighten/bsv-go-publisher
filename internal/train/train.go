package train

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/akua/bsv-broadcaster/internal/arc"
	"github.com/akua/bsv-broadcaster/internal/database"
	"github.com/akua/bsv-broadcaster/internal/models"
)

// TxWork represents a transaction ready to be broadcast
type TxWork struct {
	UUID     string
	RawTxHex string
	UTXOUsed string
}

// Train implements the "train station" batching logic
// Every N seconds, it collects up to M transactions and broadcasts them
type Train struct {
	db           *database.Database
	arcClient    *arc.Client
	txQueue      chan TxWork
	interval     time.Duration
	maxBatchSize int
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// NewTrain creates a new train worker
func NewTrain(db *database.Database, arcClient *arc.Client, interval time.Duration, maxBatchSize int) *Train {
	ctx, cancel := context.WithCancel(context.Background())

	return &Train{
		db:           db,
		arcClient:    arcClient,
		txQueue:      make(chan TxWork, maxBatchSize*10), // Buffer for 10 trains
		interval:     interval,
		maxBatchSize: maxBatchSize,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start begins the train worker
func (t *Train) Start() {
	t.wg.Add(1)
	go t.run()
	log.Printf("🚂 Train started: %v interval, max %d tx per batch", t.interval, t.maxBatchSize)
}

// Stop gracefully stops the train, finishing the current batch
func (t *Train) Stop() {
	log.Println("🛑 Train stopping... finishing current batch")
	t.cancel()
	t.wg.Wait()
	log.Println("✓ Train stopped cleanly")
}

// Enqueue adds a transaction to the queue
func (t *Train) Enqueue(work TxWork) error {
	select {
	case t.txQueue <- work:
		return nil
	case <-t.ctx.Done():
		return fmt.Errorf("train is shutting down")
	default:
		return fmt.Errorf("queue is full")
	}
}

// run is the main train loop
func (t *Train) run() {
	defer t.wg.Done()

	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	var batch []TxWork

	for {
		select {
		case work := <-t.txQueue:
			// Add to current batch
			batch = append(batch, work)

			// If batch is full, broadcast immediately
			if len(batch) >= t.maxBatchSize {
				log.Printf("🚂 Train departing early (batch full: %d tx)", len(batch))
				t.broadcastBatch(batch)
				batch = nil
			}

		case <-ticker.C:
			// Time's up! Broadcast whatever we have
			if len(batch) > 0 {
				log.Printf("🚂 Train departing on schedule (%d tx)", len(batch))
				t.broadcastBatch(batch)
				batch = nil
			}

		case <-t.ctx.Done():
			// Shutdown signal received
			// Make final attempt to broadcast pending batch
			if len(batch) > 0 {
				log.Printf("🚂 Final departure: broadcasting %d pending tx", len(batch))
				t.broadcastBatch(batch)
			}

			// Drain any remaining items in queue
			remaining := len(t.txQueue)
			if remaining > 0 {
				log.Printf("⚠️  Warning: %d transactions left in queue at shutdown", remaining)
				finalBatch := make([]TxWork, 0, remaining)
				for i := 0; i < remaining && i < t.maxBatchSize; i++ {
					finalBatch = append(finalBatch, <-t.txQueue)
				}
				if len(finalBatch) > 0 {
					t.broadcastBatch(finalBatch)
				}
			}

			return
		}
	}
}

// broadcastBatch sends a batch of transactions to ARC
func (t *Train) broadcastBatch(batch []TxWork) {
	if len(batch) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Extract hex strings
	hexes := make([]string, len(batch))
	for i, work := range batch {
		hexes[i] = work.RawTxHex
	}

	// Update all requests to "processing" status
	for _, work := range batch {
		t.db.UpdateRequestStatus(ctx, work.UUID, models.RequestStatusProcessing, "", "", "")
	}

	// Broadcast to ARC
	responses, err := t.arcClient.BroadcastBatch(ctx, hexes)
	if err != nil {
		log.Printf("❌ Batch broadcast failed: %v", err)

		// Mark all as failed
		for _, work := range batch {
			t.db.UpdateRequestStatus(ctx, work.UUID, models.RequestStatusFailed, "", "", err.Error())
			t.db.UnlockUTXO(ctx, work.UTXOUsed)
		}
		return
	}

	// Process responses
	successCount := 0
	failCount := 0

	for i, resp := range responses {
		work := batch[i]

		switch resp.TxStatus {
		case arc.TxStatusAccepted, arc.TxStatusSeenOnNetwork:
			// Success! Mark UTXO as spent and request as successful
			t.db.MarkUTXOSpent(ctx, work.UTXOUsed, resp.TxID)
			t.db.UpdateRequestStatus(ctx, work.UUID, models.RequestStatusSuccess, resp.TxID, string(resp.TxStatus), "")
			successCount++

		case arc.TxStatusMined:
			// Even better - already mined
			t.db.MarkUTXOSpent(ctx, work.UTXOUsed, resp.TxID)
			t.db.UpdateRequestStatus(ctx, work.UUID, models.RequestStatusMined, resp.TxID, string(resp.TxStatus), "")
			successCount++

		case arc.TxStatusDoubleSpend:
			// Someone else spent this UTXO - mark as spent anyway
			t.db.MarkUTXOSpent(ctx, work.UTXOUsed, resp.TxID)
			t.db.UpdateRequestStatus(ctx, work.UUID, models.RequestStatusFailed, "", string(resp.TxStatus), "double spend detected")
			failCount++

		case arc.TxStatusRejected:
			// Transaction rejected - unlock UTXO for reuse
			t.db.UnlockUTXO(ctx, work.UTXOUsed)
			t.db.UpdateRequestStatus(ctx, work.UUID, models.RequestStatusFailed, "", string(resp.TxStatus), resp.ExtraInfo)
			failCount++

		default:
			// Unknown status - mark as failed and unlock UTXO
			t.db.UnlockUTXO(ctx, work.UTXOUsed)
			t.db.UpdateRequestStatus(ctx, work.UUID, models.RequestStatusFailed, "", string(resp.TxStatus), resp.ExtraInfo)
			failCount++
		}
	}

	log.Printf("✓ Batch complete: %d success, %d failed", successCount, failCount)
}

// QueueSize returns the current number of transactions waiting
func (t *Train) QueueSize() int {
	return len(t.txQueue)
}

// IsRunning returns true if the train is still processing
func (t *Train) IsRunning() bool {
	select {
	case <-t.ctx.Done():
		return false
	default:
		return true
	}
}
