package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UTXOStatus represents the current state of a UTXO
type UTXOStatus string

const (
	UTXOStatusAvailable UTXOStatus = "available"
	UTXOStatusLocked    UTXOStatus = "locked"
	UTXOStatusSpent     UTXOStatus = "spent"
)

// UTXOType categorizes UTXOs by purpose
type UTXOType string

const (
	UTXOTypeFunding    UTXOType = "funding"    // >100 sats
	UTXOTypePublishing UTXOType = "publishing" // =100 sats
	UTXOTypeChange     UTXOType = "change"     // <100 sats
)

// UTXO represents a Bitcoin SV unspent transaction output
type UTXO struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Outpoint     string             `bson:"outpoint" json:"outpoint"`           // txid:vout
	TxID         string             `bson:"txid" json:"txid"`                   // Transaction ID
	Vout         uint32             `bson:"vout" json:"vout"`                   // Output index
	Satoshis     uint64             `bson:"satoshis" json:"satoshis"`           // Value in satoshis
	ScriptPubKey string             `bson:"script_pub_key" json:"scriptPubKey"` // Locking script (hex)
	Status       UTXOStatus         `bson:"status" json:"status"`               // available, locked, spent
	Type         UTXOType           `bson:"type" json:"type"`                   // funding, publishing, change
	LockedAt     *time.Time         `bson:"locked_at,omitempty" json:"lockedAt,omitempty"`
	SpentAt      *time.Time         `bson:"spent_at,omitempty" json:"spentAt,omitempty"`
	CreatedAt    time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updatedAt"`
}

// RequestStatus represents the state of a broadcast request
type RequestStatus string

const (
	RequestStatusPending    RequestStatus = "pending"    // Queued, waiting for train
	RequestStatusProcessing RequestStatus = "processing" // In current batch
	RequestStatusSuccess    RequestStatus = "success"    // Broadcasted successfully
	RequestStatusMined      RequestStatus = "mined"      // Confirmed in block
	RequestStatusFailed     RequestStatus = "failed"     // Broadcast failed
)

// BroadcastResult contains the result of a broadcast operation
type BroadcastResult struct {
	TXID      string
	ARCStatus string
	Error     error
}

// BroadcastRequest tracks a user's OP_RETURN publish request
type BroadcastRequest struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UUID      string             `bson:"uuid" json:"uuid"` // User-facing identifier
	RawTxHex  string             `bson:"raw_tx_hex" json:"rawTxHex"`
	TxID      string             `bson:"txid,omitempty" json:"txid,omitempty"`
	UTXOUsed  string             `bson:"utxo_used" json:"utxoUsed"` // Outpoint of publishing UTXO
	Status    RequestStatus      `bson:"status" json:"status"`
	ARCStatus string             `bson:"arc_status,omitempty" json:"arcStatus,omitempty"`
	Error     string             `bson:"error,omitempty" json:"error,omitempty"`
	CreatedAt time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updatedAt"`

	// ResponseChan is used for synchronous wait mode (?wait=true)
	// Not persisted to database
	ResponseChan chan BroadcastResult `bson:"-" json:"-"`
}
