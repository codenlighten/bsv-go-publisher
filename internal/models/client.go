package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Client represents an API client with authentication credentials
type Client struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name       string             `bson:"name" json:"name"`
	APIKeyHash string             `bson:"api_key_hash" json:"-"` // Never expose in JSON
	PublicKey  string             `bson:"public_key" json:"publicKey,omitempty"`

	// KEY ROTATION SUPPORT
	OldPublicKey     string     `bson:"old_public_key,omitempty" json:"oldPublicKey,omitempty"`
	KeyRotatedAt     *time.Time `bson:"key_rotated_at,omitempty" json:"keyRotatedAt,omitempty"`
	GracePeriodHours int        `bson:"grace_period_hours" json:"gracePeriodHours"` // Default 24

	// ADAPTIVE SECURITY TIERS
	Tier             string   `bson:"tier" json:"tier"`                                  // "pilot", "enterprise", "government"
	RequireSignature bool     `bson:"require_signature" json:"requireSignature"`         // Toggle enforcement
	AllowedIPs       []string `bson:"allowed_ips,omitempty" json:"allowedIPs,omitempty"` // IP whitelist for legacy mode

	// QUOTAS & ACTIVITY
	IsActive      bool      `bson:"is_active" json:"isActive"`
	SiteOrigin    string    `bson:"site_origin,omitempty" json:"siteOrigin,omitempty"`
	MaxDailyTx    int       `bson:"max_daily_tx" json:"maxDailyTx"`
	TxCount       int       `bson:"tx_count" json:"txCount"`              // Daily counter
	LastResetDate string    `bson:"last_reset_date" json:"lastResetDate"` // YYYY-MM-DD
	CreatedAt     time.Time `bson:"created_at" json:"createdAt"`
	UpdatedAt     time.Time `bson:"updated_at" json:"updatedAt"`
}
