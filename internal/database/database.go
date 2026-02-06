package database

import (
	"context"
	"fmt"
	"time"

	"github.com/akua/bsv-broadcaster/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	DatabaseName                = "bsv_broadcaster"
	CollectionUTXOs             = "utxos"
	CollectionBroadcastRequests = "broadcast_requests"
	CollectionClients           = "clients"
)

type Database struct {
	client *mongo.Client
	db     *mongo.Database
}

// Connect establishes a connection to MongoDB
func Connect(ctx context.Context, uri string) (*Database, error) {
	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := &Database{
		client: client,
		db:     client.Database(DatabaseName),
	}

	// Create indexes
	if err := db.createIndexes(ctx); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return db, nil
}

// createIndexes sets up critical indexes for performance
func (d *Database) createIndexes(ctx context.Context) error {
	utxosCollection := d.db.Collection(CollectionUTXOs)

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "outpoint", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "type", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "locked_at", Value: 1},
			},
		},
	}

	_, err := utxosCollection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create UTXO indexes: %w", err)
	}

	// Index for broadcast requests
	requestsCollection := d.db.Collection(CollectionBroadcastRequests)
	_, err = requestsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "uuid", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create request indexes: %w", err)
	}

	// Index for clients
	clientsCollection := d.db.Collection(CollectionClients)
	_, err = clientsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "api_key_hash", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create client indexes: %w", err)
	}

	return nil
}

// FindAndLockUTXO atomically finds an available UTXO and locks it
// This is the critical thread-safe operation for high concurrency
func (d *Database) FindAndLockUTXO(ctx context.Context, utxoType models.UTXOType) (*models.UTXO, error) {
	collection := d.db.Collection(CollectionUTXOs)

	filter := bson.M{
		"status": models.UTXOStatusAvailable,
		"type":   utxoType,
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":     models.UTXOStatusLocked,
			"locked_at":  now,
			"updated_at": now,
		},
	}

	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After).
		SetSort(bson.D{{Key: "created_at", Value: 1}}) // FIFO

	var utxo models.UTXO
	err := collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&utxo)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no available %s UTXOs", utxoType)
		}
		return nil, fmt.Errorf("failed to lock UTXO: %w", err)
	}

	return &utxo, nil
}

// FindUTXOsByType finds all UTXOs of a specific type and status without locking
func (d *Database) FindUTXOsByType(ctx context.Context, utxoType models.UTXOType, status models.UTXOStatus) ([]*models.UTXO, error) {
	collection := d.db.Collection(CollectionUTXOs)

	filter := bson.M{
		"type":   utxoType,
		"status": status,
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find UTXOs: %w", err)
	}
	defer cursor.Close(ctx)

	var utxos []*models.UTXO
	if err := cursor.All(ctx, &utxos); err != nil {
		return nil, fmt.Errorf("failed to decode UTXOs: %w", err)
	}

	return utxos, nil
}

// FindAndLockBatch atomically locks multiple UTXOs for batch operations
func (d *Database) FindAndLockBatch(ctx context.Context, utxoType models.UTXOType, count int) ([]*models.UTXO, error) {
	utxos := make([]*models.UTXO, 0, count)

	for i := 0; i < count; i++ {
		utxo, err := d.FindAndLockUTXO(ctx, utxoType)
		if err != nil {
			// If we can't get all requested, return what we have
			if len(utxos) > 0 {
				return utxos, nil
			}
			return nil, err
		}
		utxos = append(utxos, utxo)
	}

	return utxos, nil
}

// MarkUTXOSpent marks a UTXO as spent
func (d *Database) MarkUTXOSpent(ctx context.Context, outpoint string, txid string) error {
	collection := d.db.Collection(CollectionUTXOs)

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":     models.UTXOStatusSpent,
			"spent_at":   now,
			"updated_at": now,
		},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"outpoint": outpoint}, update)
	return err
}

// LockUTXO locks a specific UTXO by outpoint
func (d *Database) LockUTXO(ctx context.Context, outpoint string) error {
	collection := d.db.Collection(CollectionUTXOs)

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":     models.UTXOStatusLocked,
			"locked_at":  now,
			"updated_at": now,
		},
	}

	result, err := collection.UpdateOne(ctx, bson.M{"outpoint": outpoint}, update)
	if err != nil {
		return fmt.Errorf("failed to lock UTXO: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("UTXO not found: %s", outpoint)
	}
	return nil
}

// UnlockUTXO releases a locked UTXO back to available status
func (d *Database) UnlockUTXO(ctx context.Context, outpoint string) error {
	collection := d.db.Collection(CollectionUTXOs)

	update := bson.M{
		"$set": bson.M{
			"status":     models.UTXOStatusAvailable,
			"locked_at":  nil,
			"updated_at": time.Now(),
		},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"outpoint": outpoint}, update)
	return err
}

// ClearAllUTXOs removes all UTXOs from the database
// Used during blockchain sync to start fresh
func (d *Database) ClearAllUTXOs(ctx context.Context) error {
	collection := d.db.Collection(CollectionUTXOs)
	result, err := collection.DeleteMany(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to clear UTXOs: %w", err)
	}
	if result.DeletedCount > 0 {
		fmt.Printf("   Cleared %d existing UTXOs\n", result.DeletedCount)
	}
	return nil
}

// InsertUTXO adds a new UTXO to the database
func (d *Database) InsertUTXO(ctx context.Context, utxo *models.UTXO) error {
	collection := d.db.Collection(CollectionUTXOs)

	utxo.CreatedAt = time.Now()
	utxo.UpdatedAt = time.Now()

	_, err := collection.InsertOne(ctx, utxo)
	if err != nil {
		// Ignore duplicate key errors (UTXO already exists)
		if mongo.IsDuplicateKeyError(err) {
			return nil
		}
		return fmt.Errorf("failed to insert UTXO: %w", err)
	}

	return nil
}

// UpsertUTXO inserts or updates a UTXO in the database
// Used during blockchain sync to ensure UTXOs are up-to-date without duplicates
func (d *Database) UpsertUTXO(ctx context.Context, utxo *models.UTXO) error {
	collection := d.db.Collection(CollectionUTXOs)

	now := time.Now()
	utxo.UpdatedAt = now

	filter := bson.M{"outpoint": utxo.Outpoint}

	update := bson.M{
		"$set": bson.M{
			"txid":           utxo.TxID,
			"vout":           utxo.Vout,
			"satoshis":       utxo.Satoshis,
			"script_pub_key": utxo.ScriptPubKey,
			"type":           utxo.Type,
			"updated_at":     now,
		},
		"$setOnInsert": bson.M{
			"outpoint":   utxo.Outpoint,
			"status":     models.UTXOStatusAvailable,
			"locked_at":  nil,
			"spent_at":   nil,
			"created_at": now,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert UTXO: %w", err)
	}

	return nil
}

// InsertBroadcastRequest creates a new broadcast request record
func (d *Database) InsertBroadcastRequest(ctx context.Context, req *models.BroadcastRequest) error {
	collection := d.db.Collection(CollectionBroadcastRequests)

	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	_, err := collection.InsertOne(ctx, req)
	return err
}

// UpdateRequestStatus updates the status of a broadcast request
func (d *Database) UpdateRequestStatus(ctx context.Context, uuid string, status models.RequestStatus, txid, arcStatus, errorMsg string) error {
	collection := d.db.Collection(CollectionBroadcastRequests)

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	if txid != "" {
		update["$set"].(bson.M)["txid"] = txid
	}
	if arcStatus != "" {
		update["$set"].(bson.M)["arc_status"] = arcStatus
	}
	if errorMsg != "" {
		update["$set"].(bson.M)["error"] = errorMsg
	}

	_, err := collection.UpdateOne(ctx, bson.M{"uuid": uuid}, update)
	return err
}

// GetRequestByUUID retrieves a broadcast request by UUID
func (d *Database) GetRequestByUUID(ctx context.Context, uuid string) (*models.BroadcastRequest, error) {
	collection := d.db.Collection(CollectionBroadcastRequests)

	var req models.BroadcastRequest
	err := collection.FindOne(ctx, bson.M{"uuid": uuid}).Decode(&req)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("request not found")
		}
		return nil, err
	}

	return &req, nil
}

// RecoverStuckUTXOs finds UTXOs locked for more than the specified duration and unlocks them
func (d *Database) RecoverStuckUTXOs(ctx context.Context, maxAge time.Duration) (int64, error) {
	collection := d.db.Collection(CollectionUTXOs)

	threshold := time.Now().Add(-maxAge)

	filter := bson.M{
		"status":    models.UTXOStatusLocked,
		"locked_at": bson.M{"$lt": threshold},
	}

	update := bson.M{
		"$set": bson.M{
			"status":     models.UTXOStatusAvailable,
			"locked_at":  nil,
			"updated_at": time.Now(),
		},
	}

	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("failed to recover stuck UTXOs: %w", err)
	}

	return result.ModifiedCount, nil
}

// GetUTXOStats returns counts of UTXOs by type and status
func (d *Database) GetUTXOStats(ctx context.Context) (map[string]int64, error) {
	collection := d.db.Collection(CollectionUTXOs)

	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "type", Value: "$type"},
				{Key: "status", Value: "$status"},
			}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	stats := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Type   models.UTXOType   `bson:"type"`
				Status models.UTXOStatus `bson:"status"`
			} `bson:"_id"`
			Count int64 `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		key := fmt.Sprintf("%s_%s", result.ID.Type, result.ID.Status)
		stats[key] = result.Count
	}

	return stats, nil
}

// GetAvailableUTXOs fetches up to limit available UTXOs of a specific type
func (d *Database) GetAvailableUTXOs(ctx context.Context, utxoType models.UTXOType, limit int) ([]*models.UTXO, error) {
	collection := d.db.Collection(CollectionUTXOs)

	filter := bson.M{
		"status": models.UTXOStatusAvailable,
		"type":   utxoType,
	}

	opts := options.Find().SetLimit(int64(limit))
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var utxos []*models.UTXO
	if err := cursor.All(ctx, &utxos); err != nil {
		return nil, err
	}

	return utxos, nil
}

// Client Management Methods

// CreateClient creates a new API client
func (d *Database) CreateClient(ctx context.Context, client *models.Client) error {
	collection := d.db.Collection(CollectionClients)
	_, err := collection.InsertOne(ctx, client)
	return err
}

// GetClientByAPIKeyHash retrieves a client by their hashed API key
func (d *Database) GetClientByAPIKeyHash(ctx context.Context, apiKeyHash string) (*models.Client, error) {
	collection := d.db.Collection(CollectionClients)

	var client models.Client
	err := collection.FindOne(ctx, bson.M{"api_key_hash": apiKeyHash}).Decode(&client)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("invalid API key")
		}
		return nil, err
	}

	return &client, nil
}

// IncrementClientTxCount increments the transaction count for a client
// Resets the counter if it's a new day
func (d *Database) IncrementClientTxCount(ctx context.Context, clientID interface{}, today string) error {
	collection := d.db.Collection(CollectionClients)

	// First, check if we need to reset the counter
	var client models.Client
	err := collection.FindOne(ctx, bson.M{"_id": clientID}).Decode(&client)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	// If it's a new day, reset the counter
	if client.LastResetDate != today {
		update["$set"].(bson.M)["tx_count"] = 1
		update["$set"].(bson.M)["last_reset_date"] = today
	} else {
		update["$inc"] = bson.M{"tx_count": 1}
	}

	_, err = collection.UpdateOne(ctx, bson.M{"_id": clientID}, update)
	return err
}

// UpdateClientStatus activates or deactivates a client
func (d *Database) UpdateClientStatus(ctx context.Context, clientID interface{}, isActive bool) error {
	collection := d.db.Collection(CollectionClients)

	update := bson.M{
		"$set": bson.M{
			"is_active":  isActive,
			"updated_at": time.Now(),
		},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"_id": clientID}, update)
	return err
}

// ListClients returns all registered clients
func (d *Database) ListClients(ctx context.Context) ([]*models.Client, error) {
	collection := d.db.Collection(CollectionClients)

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var clients []*models.Client
	if err := cursor.All(ctx, &clients); err != nil {
		return nil, err
	}

	return clients, nil
}

// Close closes the database connection
func (d *Database) Close(ctx context.Context) error {
	return d.client.Disconnect(ctx)
}
