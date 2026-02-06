package recovery

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/akua/bsv-broadcaster/internal/database"
)

// Janitor handles periodic cleanup of stuck UTXOs
type Janitor struct {
	db         *database.Database
	interval   time.Duration
	maxLockAge time.Duration
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewJanitor creates a new janitor service
func NewJanitor(db *database.Database, interval, maxLockAge time.Duration) *Janitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &Janitor{
		db:         db,
		interval:   interval,
		maxLockAge: maxLockAge,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start begins the janitor background routine
func (j *Janitor) Start() {
	j.wg.Add(1)
	go j.run()
	log.Printf("🧹 Janitor started: checking every %v for UTXOs locked > %v", j.interval, j.maxLockAge)
}

// Stop gracefully stops the janitor
func (j *Janitor) Stop() {
	log.Println("🧹 Janitor stopping...")
	j.cancel()
	j.wg.Wait()
	log.Println("✓ Janitor stopped")
}

// run is the main janitor loop
func (j *Janitor) run() {
	defer j.wg.Done()

	// Run immediately on startup
	j.cleanup()

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			j.cleanup()
		case <-j.ctx.Done():
			return
		}
	}
}

// cleanup recovers stuck UTXOs
func (j *Janitor) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	recovered, err := j.db.RecoverStuckUTXOs(ctx, j.maxLockAge)
	if err != nil {
		log.Printf("❌ Janitor cleanup failed: %v", err)
		return
	}

	if recovered > 0 {
		log.Printf("🧹 Janitor recovered %d stuck UTXOs", recovered)
	}
}

// RunStartupRecovery performs initial recovery on server startup
func RunStartupRecovery(db *database.Database, maxAge time.Duration) error {
	log.Println("🔧 Running startup recovery for stuck UTXOs...")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	recovered, err := db.RecoverStuckUTXOs(ctx, maxAge)
	if err != nil {
		return err
	}

	if recovered > 0 {
		log.Printf("✓ Startup recovery: unlocked %d UTXOs", recovered)
	} else {
		log.Println("✓ Startup recovery: no stuck UTXOs found")
	}

	return nil
}
