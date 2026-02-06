package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Client represents an API client with authentication credentials
type Client struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name          string             `bson:"name" json:"name"`
	APIKeyHash    string             `bson:"api_key_hash" json:"-"` // Never expose in JSON
	PublicKey     string             `bson:"public_key" json:"publicKey,omitempty"`
	IsActive      bool               `bson:"is_active" json:"isActive"`
	SiteOrigin    string             `bson:"site_origin,omitempty" json:"siteOrigin,omitempty"`
	MaxDailyTx    int                `bson:"max_daily_tx" json:"maxDailyTx"`
	TxCount       int                `bson:"tx_count" json:"txCount"`              // Daily counter
	LastResetDate string             `bson:"last_reset_date" json:"lastResetDate"` // YYYY-MM-DD
	CreatedAt     time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updatedAt"`
}
