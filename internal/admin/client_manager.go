package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/akua/bsv-broadcaster/internal/auth"
	"github.com/akua/bsv-broadcaster/internal/database"
	"github.com/akua/bsv-broadcaster/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ClientManager handles client registration and management
type ClientManager struct {
	db *database.Database
}

// NewClientManager creates a new client manager
func NewClientManager(db *database.Database) *ClientManager {
	return &ClientManager{db: db}
}

// RegisterClient creates a new API client with authentication credentials
func (cm *ClientManager) RegisterClient(ctx context.Context, name, publicKey, siteOrigin string, maxDailyTx int) (string, *models.Client, error) {
	// Generate API key
	rawKey, hashedKey, err := auth.GenerateAPIKey()
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Create client record
	now := time.Now()
	client := &models.Client{
		ID:            primitive.NewObjectID(),
		Name:          name,
		APIKeyHash:    hashedKey,
		PublicKey:     publicKey,
		IsActive:      true,
		SiteOrigin:    siteOrigin,
		MaxDailyTx:    maxDailyTx,
		TxCount:       0,
		LastResetDate: now.Format("2006-01-02"),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Store in database
	if err := cm.db.CreateClient(ctx, client); err != nil {
		return "", nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Return the raw key (this is the ONLY time it will be shown)
	return rawKey, client, nil
}

// GetClientByAPIKey retrieves a client by their API key (hashes the key first)
func (cm *ClientManager) GetClientByAPIKey(ctx context.Context, apiKey string) (*models.Client, error) {
	hashedKey := auth.HashAPIKey(apiKey)
	return cm.db.GetClientByAPIKeyHash(ctx, hashedKey)
}

// GetClientByID retrieves a client by their ObjectID
func (cm *ClientManager) GetClientByID(ctx context.Context, clientID primitive.ObjectID) (*models.Client, error) {
	return cm.db.GetClientByID(ctx, clientID)
}

// IncrementClientTxCount increments the transaction count for a client
// Resets the counter if it's a new day
func (cm *ClientManager) IncrementClientTxCount(ctx context.Context, clientID primitive.ObjectID) error {
	today := time.Now().Format("2006-01-02")
	return cm.db.IncrementClientTxCount(ctx, clientID, today)
}

// DeactivateClient disables a client's API access
func (cm *ClientManager) DeactivateClient(ctx context.Context, clientID primitive.ObjectID) error {
	return cm.db.UpdateClientStatus(ctx, clientID, false)
}

// ActivateClient enables a client's API access
func (cm *ClientManager) ActivateClient(ctx context.Context, clientID primitive.ObjectID) error {
	return cm.db.UpdateClientStatus(ctx, clientID, true)
}

// ListClients returns all registered clients
func (cm *ClientManager) ListClients(ctx context.Context) ([]*models.Client, error) {
	return cm.db.ListClients(ctx)
}
