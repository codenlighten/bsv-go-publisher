package admin

import (
	"context"
	"fmt"
	"log"

	"github.com/akua/bsv-broadcaster/internal/arc"
	"github.com/akua/bsv-broadcaster/internal/bsv"
	"github.com/akua/bsv-broadcaster/internal/database"
	"github.com/akua/bsv-broadcaster/internal/models"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/p2pkh"
)

// Sweeper handles UTXO consolidation operations
type Sweeper struct {
	db            *database.Database
	publishingKey *bsv.KeyPair
	arcClient     *arc.Client
	feeRate       float64
}

// NewSweeper creates a new UTXO sweeper
func NewSweeper(db *database.Database, publishingKey *bsv.KeyPair, arcClient *arc.Client, feeRate float64) *Sweeper {
	return &Sweeper{
		db:            db,
		publishingKey: publishingKey,
		arcClient:     arcClient,
		feeRate:       feeRate,
	}
}

// SweepUTXOs consolidates multiple small UTXOs into one large UTXO at destination address
func (s *Sweeper) SweepUTXOs(ctx context.Context, destAddress string, maxInputs int, utxoType models.UTXOType) (string, uint64, error) {
	// 1. Fetch available UTXOs to consolidate
	utxos, err := s.db.GetAvailableUTXOs(ctx, utxoType, maxInputs)
	if err != nil {
		return "", 0, fmt.Errorf("failed to fetch UTXOs: %w", err)
	}

	if len(utxos) == 0 {
		return "", 0, fmt.Errorf("no available UTXOs found")
	}

	log.Printf("🧹 Sweeping %d UTXOs to %s", len(utxos), destAddress)

	// 2. Build consolidation transaction
	tx := transaction.NewTransaction()
	var totalInputSats uint64

	// Get unlocker
	unlocker, err := p2pkh.Unlock(s.publishingKey.PrivateKey, nil)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create unlocker: %w", err)
	}

	// Add all inputs
	for _, utxo := range utxos {
		if err := tx.AddInputFrom(utxo.TxID, utxo.Vout, utxo.ScriptPubKey, utxo.Satoshis, unlocker); err != nil {
			return "", 0, fmt.Errorf("failed to add input: %w", err)
		}
		totalInputSats += utxo.Satoshis
	}

	// 3. Calculate fee based on transaction size
	// Estimate size: inputs already added, plus one output
	estimatedSize := tx.Size() + 34 // +34 for output
	fee := uint64(float64(estimatedSize) * s.feeRate)
	if fee < 1 {
		fee = 1
	}

	// Check if we have enough for fee
	if totalInputSats <= fee {
		return "", 0, fmt.Errorf("total input (%d sats) is less than fee (%d sats)", totalInputSats, fee)
	}

	outputAmount := totalInputSats - fee

	// 4. Add single output to destination
	if err := tx.PayToAddress(destAddress, outputAmount); err != nil {
		return "", 0, fmt.Errorf("failed to add output: %w", err)
	}

	// 5. Sign transaction
	if err := tx.Sign(); err != nil {
		return "", 0, fmt.Errorf("failed to sign transaction: %w", err)
	}

	txHex := tx.String()
	txid := tx.TxID().String()

	log.Printf("Sweep TX: %s", txid)
	log.Printf("Size: %d bytes, Fee: %d sats, Output: %d sats", len(txHex)/2, fee, outputAmount)

	// 6. Broadcast transaction
	response, err := s.arcClient.BroadcastSingle(ctx, txHex)
	if err != nil {
		return "", 0, fmt.Errorf("failed to broadcast: %w", err)
	}

	log.Printf("Broadcast status: %s", response.TxStatus)

	// 7. Mark all input UTXOs as spent in database
	for _, utxo := range utxos {
		outpoint := fmt.Sprintf("%s:%d", utxo.TxID, utxo.Vout)
		if err := s.db.MarkUTXOSpent(ctx, outpoint, txid); err != nil {
			log.Printf("Warning: failed to mark UTXO %s as spent: %v", outpoint, err)
		}
	}

	return txid, outputAmount, nil
}

// ConsolidateDust sweeps all small change UTXOs back to funding address
func (s *Sweeper) ConsolidateDust(ctx context.Context, fundingAddress string, maxInputs int) (string, uint64, error) {
	return s.SweepUTXOs(ctx, fundingAddress, maxInputs, models.UTXOTypeChange)
}

// EstimateSweepValue calculates how much would be consolidated (minus fees)
func (s *Sweeper) EstimateSweepValue(ctx context.Context, utxoType models.UTXOType, maxInputs int) (uint64, int, error) {
	utxos, err := s.db.GetAvailableUTXOs(ctx, utxoType, maxInputs)
	if err != nil {
		return 0, 0, err
	}

	if len(utxos) == 0 {
		return 0, 0, nil
	}

	var total uint64
	for _, utxo := range utxos {
		total += utxo.Satoshis
	}

	// Estimate fee (roughly 150 bytes per input + 34 for output + 10 overhead)
	estimatedSize := uint64(len(utxos)*150 + 34 + 10)
	estimatedFee := uint64(float64(estimatedSize) * s.feeRate)

	if total <= estimatedFee {
		return 0, len(utxos), nil
	}

	return total - estimatedFee, len(utxos), nil
}
