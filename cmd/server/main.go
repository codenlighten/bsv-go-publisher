package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/akua/bsv-broadcaster/internal/api"
	"github.com/akua/bsv-broadcaster/internal/arc"
	"github.com/akua/bsv-broadcaster/internal/bsv"
	"github.com/akua/bsv-broadcaster/internal/database"
	"github.com/akua/bsv-broadcaster/internal/recovery"
	"github.com/akua/bsv-broadcaster/internal/train"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, using environment variables")
	}

	log.Println("🚀 BSV AKUA Broadcast Server starting...")

	// Parse configuration
	config := loadConfig()

	// Setup context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Connect to database
	db, err := database.Connect(ctx, config.MongoURI)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close(context.Background())
	log.Println("✓ Connected to MongoDB")

	// Load or generate keypairs
	fundingKey, err := bsv.LoadOrGenerateKeyPair("FUNDING_PRIVKEY")
	if err != nil {
		log.Fatalf("❌ Failed to load funding key: %v", err)
	}

	publishingKey, err := bsv.LoadOrGenerateKeyPair("PUBLISHING_PRIVKEY")
	if err != nil {
		log.Fatalf("❌ Failed to load publishing key: %v", err)
	}

	log.Printf("✓ Funding Address: %s", fundingKey.Address)
	log.Printf("✓ Publishing Address: %s", publishingKey.Address)

	// Run startup recovery
	if err := recovery.RunStartupRecovery(db, 5*time.Minute); err != nil {
		log.Fatalf("❌ Startup recovery failed: %v", err)
	}

	// Sync blockchain state (placeholder for now)
	syncService := bsv.NewSyncService(db, fundingKey.Address, publishingKey.Address)
	if err := syncService.SyncUTXOs(ctx); err != nil {
		log.Printf("⚠️  Blockchain sync failed: %v", err)
	}

	// Check UTXO pool and refill if needed
	stats, _ := db.GetUTXOStats(ctx)
	publishingAvailable := stats["publishing_available"]
	log.Printf("📊 Current UTXO stats: %d publishing UTXOs available", publishingAvailable)

	if publishingAvailable < int64(config.TargetPublishingUTXOs/10) {
		log.Println("⚠️  Low on publishing UTXOs, consider running splitter")
		log.Println("   (Splitter integration with ARC broadcasting needed for production)")
	}

	// Initialize ARC client
	arcClient := arc.NewClient(config.ARCURL, config.ARCToken)
	log.Printf("✓ ARC client configured: %s", config.ARCURL)

	// Test ARC connectivity
	if err := arcClient.Health(ctx); err != nil {
		log.Printf("⚠️  ARC health check failed: %v", err)
	} else {
		log.Println("✓ ARC is healthy")
	}

	// Initialize splitter
	splitter := bsv.NewSplitter(db, fundingKey, publishingKey.Address, 1.0) // 1 sat/byte fee rate
	log.Println("✓ Splitter initialized")

	// Start the train
	trainWorker := train.NewTrain(db, arcClient, config.TrainInterval, config.TrainMaxBatch)
	trainWorker.Start()

	// Start the janitor
	janitor := recovery.NewJanitor(db, 10*time.Minute, 5*time.Minute)
	janitor.Start()

	// Start API server
	apiServer := api.NewServer(db, trainWorker, publishingKey, splitter, arcClient)
	go func() {
		if err := apiServer.Start(":8080"); err != nil {
			log.Printf("❌ API server error: %v", err)
		}
	}()

	log.Println("✓ Server ready!")
	log.Println()
	log.Println("📡 Endpoints:")
	log.Println("   POST /publish         - Submit OP_RETURN data for broadcasting")
	log.Println("   GET  /status/:uuid    - Check broadcast status")
	log.Println("   GET  /health          - Health check with UTXO stats")
	log.Println("   GET  /admin/stats     - Detailed statistics")
	log.Println()

	// Wait for interrupt signal
	<-ctx.Done()

	log.Println()
	log.Println("🛑 Shutdown signal received, cleaning up...")

	// Graceful shutdown sequence
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Stop accepting new requests
	if err := apiServer.Shutdown(); err != nil {
		log.Printf("⚠️  API shutdown error: %v", err)
	}

	// 2. Stop the train (finishes current batch)
	trainWorker.Stop()

	// 3. Stop the janitor
	janitor.Stop()

	// 4. Close database
	if err := db.Close(shutdownCtx); err != nil {
		log.Printf("⚠️  Database close error: %v", err)
	}

	log.Println("✓ Server stopped cleanly")
}

// Config holds application configuration
type Config struct {
	MongoURI              string
	ARCURL                string
	ARCToken              string
	TrainInterval         time.Duration
	TrainMaxBatch         int
	TargetPublishingUTXOs int
}

// loadConfig loads configuration from environment
func loadConfig() Config {
	trainInterval, _ := time.ParseDuration(getEnv("TRAIN_INTERVAL", "3s"))
	trainMaxBatch, _ := strconv.Atoi(getEnv("TRAIN_MAX_BATCH", "1000"))
	targetUTXOs, _ := strconv.Atoi(getEnv("TARGET_PUBLISHING_UTXOS", "50000"))

	return Config{
		MongoURI:              getEnv("MONGO_URI", "mongodb://root:password@localhost:27017"),
		ARCURL:                getEnv("ARC_URL", "https://arc.gorillapool.io"),
		ARCToken:              getEnv("ARC_TOKEN", ""),
		TrainInterval:         trainInterval,
		TrainMaxBatch:         trainMaxBatch,
		TargetPublishingUTXOs: targetUTXOs,
	}
}

// getEnv retrieves an environment variable with a fallback default
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
