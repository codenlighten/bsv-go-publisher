package bsv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/akua/bsv-broadcaster/internal/database"
	"github.com/akua/bsv-broadcaster/internal/models"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction/template/p2pkh"
)

// BitailsUTXO represents a UTXO from Bitails API
type BitailsUTXO struct {
	TxID          string `json:"txid"`
	Vout          int    `json:"vout"`
	Satoshis      uint64 `json:"satoshis"`
	Time          int64  `json:"time"`
	BlockHeight   int    `json:"blockheight"`
	Confirmations int    `json:"confirmations"`
}

// BitailsResponse represents the response from Bitails /unspent endpoint
type BitailsResponse struct {
	Address    string        `json:"address"`
	ScriptHash string        `json:"scripthash"`
	Unspent    []BitailsUTXO `json:"unspent"`
}

// SyncService handles syncing local UTXO database with blockchain state
type SyncService struct {
	db             *database.Database
	fundingAddr    string
	publishingAddr string
}

// NewSyncService creates a new blockchain sync service
func NewSyncService(db *database.Database, fundingAddr, publishingAddr string) *SyncService {
	return &SyncService{
		db:             db,
		fundingAddr:    fundingAddr,
		publishingAddr: publishingAddr,
	}
}

// SyncUTXOs queries the blockchain and syncs the local database
// This is called on startup to ensure consistency
func (s *SyncService) SyncUTXOs(ctx context.Context) error {
	fmt.Println("🔄 Starting UTXO sync with blockchain...")

	// Clear all existing UTXOs to start fresh from blockchain state
	fmt.Println("   Clearing existing UTXO database...")
	if err := s.db.ClearAllUTXOs(ctx); err != nil {
		fmt.Printf("⚠️  Warning: failed to clear UTXOs: %v\n", err)
	}

	// Sync both addresses
	if err := s.syncAddress(ctx, s.fundingAddr); err != nil {
		return fmt.Errorf("failed to sync funding address: %w", err)
	}

	if err := s.syncAddress(ctx, s.publishingAddr); err != nil {
		return fmt.Errorf("failed to sync publishing address: %w", err)
	}

	// Count results
	stats, err := s.db.GetUTXOStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Printf("✓ Sync complete: %d funding, %d publishing, %d change UTXOs\n",
		stats["funding_available"],
		stats["publishing_available"],
		stats["change_available"],
	)

	return nil
}

// syncAddress fetches UTXOs for a specific address from Bitails
func (s *SyncService) syncAddress(ctx context.Context, address string) error {
	// Build URL with high limit to get all UTXOs in one request
	url := fmt.Sprintf("https://api.bitails.io/address/%s/unspent?limit=100000", address)

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Make request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch UTXOs from Bitails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Bitails returned status %d", resp.StatusCode)
	}

	// Parse response - Bitails returns unspent array directly
	var response BitailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	utxos := response.Unspent
	fmt.Printf("   Found %d UTXOs for %s\n", len(utxos), address)

	// Insert/update each UTXO in database
	for _, u := range utxos {
		outpoint := fmt.Sprintf("%s:%d", u.TxID, u.Vout)

		// Determine category
		utxoType := CategorizeUTXO(u.Satoshis)

		// Get script pubkey
		scriptPubKey := createP2PKHScriptFromAddress(address)

		// Create UTXO model
		utxo := &models.UTXO{
			Outpoint:     outpoint,
			TxID:         u.TxID,
			Vout:         uint32(u.Vout),
			Satoshis:     u.Satoshis,
			ScriptPubKey: scriptPubKey,
			Status:       models.UTXOStatusAvailable,
			Type:         utxoType,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Upsert to database (won't duplicate if already exists)
		if err := s.db.UpsertUTXO(ctx, utxo); err != nil {
			fmt.Printf("⚠️  Warning: failed to upsert UTXO %s: %v\n", outpoint, err)
			continue
		}
	}

	fmt.Printf("✓ Synced %d UTXOs for %s\n", len(utxos), address)
	return nil
}

// CategorizeUTXO determines the type of UTXO based on satoshi value
func CategorizeUTXO(satoshis uint64) models.UTXOType {
	switch {
	case satoshis > 100:
		return models.UTXOTypeFunding
	case satoshis == 100:
		return models.UTXOTypePublishing
	default:
		return models.UTXOTypeChange
	}
}

// createP2PKHScriptFromAddress creates a P2PKH locking script from an address
func createP2PKHScriptFromAddress(address string) string {
	// Decode address to get the public key hash
	addr, err := script.NewAddressFromString(address)
	if err != nil {
		return ""
	}

	// Create P2PKH locking script: OP_DUP OP_HASH160 <pubKeyHash> OP_EQUALVERIFY OP_CHECKSIG
	lockingScript, err := p2pkh.Lock(addr)
	if err != nil {
		return ""
	}

	return lockingScript.String()
}
