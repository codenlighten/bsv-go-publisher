package bsv

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/akua/bsv-broadcaster/internal/database"
	"github.com/akua/bsv-broadcaster/internal/models"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/p2pkh"
)

// SimpleFeeModel implements transaction.FeeModel for sat-per-byte fee calculation
type SimpleFeeModel struct {
	FeeRate float64 // sats per byte
}

// ComputeFee calculates the transaction fee based on size and feeRate
func (fm *SimpleFeeModel) ComputeFee(tx *transaction.Transaction) (uint64, error) {
	size := uint64(tx.Size())
	fee := uint64(float64(size) * fm.FeeRate)
	return fee, nil
}

// Splitter handles splitting funding UTXOs into publishing UTXOs
type Splitter struct {
	db             *database.Database
	fundingKey     *KeyPair
	publishingAddr string
	feeRate        float64 // sats per byte
}

// NewSplitter creates a new UTXO splitter
func NewSplitter(db *database.Database, fundingKey *KeyPair, publishingAddr string, feeRate float64) *Splitter {
	return &Splitter{
		db:             db,
		fundingKey:     fundingKey,
		publishingAddr: publishingAddr,
		feeRate:        feeRate,
	}
}

// SplitResult contains the results of a split operation
type SplitResult struct {
	BranchTxIDs []string
	LeafTxIDs   []string
	TotalUTXOs  int
}

// CreatePublishingUTXOs implements the tree-based split strategy
// Phase 1: Split funding UTXO into N branch UTXOs
// Phase 2: Split each branch into 1000 leaf UTXOs (publishing)
func (s *Splitter) CreatePublishingUTXOs(ctx context.Context, targetCount int) (*SplitResult, error) {
	fmt.Printf("🌳 Starting UTXO split to create %d publishing UTXOs...\n", targetCount)

	// Calculate how many branches we need
	leavesPerBranch := 1000
	branchCount := (targetCount + leavesPerBranch - 1) / leavesPerBranch // Round up

	fmt.Printf("   Strategy: %d branches × %d leaves = %d UTXOs\n", branchCount, leavesPerBranch, branchCount*leavesPerBranch)

	// Phase 1: Create branches
	branchUTXOs, branchTxIDs, err := s.createBranches(ctx, branchCount)
	if err != nil {
		return nil, fmt.Errorf("failed to create branches: %w", err)
	}

	fmt.Printf("✓ Phase 1 complete: Created %d branch UTXOs\n", len(branchUTXOs))

	// Phase 2: Create leaves from each branch (can be parallelized)
	leafTxIDs, err := s.createLeaves(ctx, branchUTXOs, leavesPerBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to create leaves: %w", err)
	}

	totalUTXOs := len(leafTxIDs) * leavesPerBranch
	fmt.Printf("✓ Phase 2 complete: Created %d leaf transactions (%d UTXOs)\n", len(leafTxIDs), totalUTXOs)

	return &SplitResult{
		BranchTxIDs: branchTxIDs,
		LeafTxIDs:   leafTxIDs,
		TotalUTXOs:  totalUTXOs,
	}, nil
}

// createBranches splits a funding UTXO into multiple branch UTXOs
func (s *Splitter) createBranches(ctx context.Context, branchCount int) ([]*models.UTXO, []string, error) {
	// Find a large funding UTXO
	fundingUTXO, err := s.db.FindAndLockUTXO(ctx, models.UTXOTypeFunding)
	if err != nil {
		return nil, nil, fmt.Errorf("no funding UTXOs available: %w", err)
	}

	// Each branch needs enough sats for 1000 leaves + fees
	// 1000 * 100 = 100,000 sats per branch + ~500 sats for fees
	satsPerBranch := uint64(100500)
	totalNeeded := satsPerBranch * uint64(branchCount)

	if fundingUTXO.Satoshis < totalNeeded {
		return nil, nil, fmt.Errorf("funding UTXO has %d sats, need %d", fundingUTXO.Satoshis, totalNeeded)
	}

	// Create the branch transaction
	tx := transaction.NewTransaction()

	// Add input using AddInputFrom
	utxoScript, err := hex.DecodeString(fundingUTXO.ScriptPubKey)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid script: %w", err)
	}

	// Create unlocker for P2PKH
	unlocker, err := p2pkh.Unlock(s.fundingKey.PrivateKey, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create unlocker: %w", err)
	}

	err = tx.AddInputFrom(
		fundingUTXO.TxID,
		fundingUTXO.Vout,
		hex.EncodeToString(utxoScript),
		fundingUTXO.Satoshis,
		unlocker,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to add input: %w", err)
	}

	// Add outputs (branches)
	for i := 0; i < branchCount; i++ {
		err = tx.PayToAddress(s.publishingAddr, satsPerBranch)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to add output %d: %w", i, err)
		}
	}

	// Add change output manually
	totalOutputValue := uint64(branchCount) * satsPerBranch
	txSize := tx.Size() + 34
	fee := uint64(float64(txSize) * s.feeRate)
	if fee < 1 {
		fee = 1
	}
	changeAmount := fundingUTXO.Satoshis - totalOutputValue - fee
	if changeAmount > 546 {
		if err := tx.PayToAddress(s.fundingKey.Address, changeAmount); err != nil {
			return nil, nil, fmt.Errorf("failed to add change output: %w", err)
		}
	}

	// Sign the transaction
	if err := tx.Sign(); err != nil {
		return nil, nil, fmt.Errorf("failed to sign: %w", err)
	}

	// In production, broadcast this transaction via ARC
	// For now, we'll just create the UTXO records
	txid := tx.TxID().String()

	// Create branch UTXO records
	branchUTXOs := make([]*models.UTXO, 0, branchCount)
	for i := 0; i < branchCount; i++ {
		scriptHex := hex.EncodeToString(*tx.Outputs[i].LockingScript)
		utxo := &models.UTXO{
			Outpoint:     fmt.Sprintf("%s:%d", txid, i),
			TxID:         txid,
			Vout:         uint32(i),
			Satoshis:     satsPerBranch,
			ScriptPubKey: scriptHex,
			Status:       models.UTXOStatusAvailable,
			Type:         models.UTXOTypeFunding, // These are still "funding" tier
		}

		if err := s.db.InsertUTXO(ctx, utxo); err != nil {
			return nil, nil, err
		}

		branchUTXOs = append(branchUTXOs, utxo)
	}

	// Mark original funding UTXO as spent
	s.db.MarkUTXOSpent(ctx, fundingUTXO.Outpoint, txid)

	return branchUTXOs, []string{txid}, nil
}

// createLeaves splits branch UTXOs into 100-sat publishing UTXOs
func (s *Splitter) createLeaves(ctx context.Context, branchUTXOs []*models.UTXO, leavesPerBranch int) ([]string, error) {
	leafTxIDs := make([]string, 0, len(branchUTXOs))

	for _, branch := range branchUTXOs {
		tx := transaction.NewTransaction()

		// Add input
		utxoScript, err := hex.DecodeString(branch.ScriptPubKey)
		if err != nil {
			return nil, err
		}

		unlocker, err := p2pkh.Unlock(s.fundingKey.PrivateKey, nil)
		if err != nil {
			return nil, err
		}

		err = tx.AddInputFrom(
			branch.TxID,
			branch.Vout,
			hex.EncodeToString(utxoScript),
			branch.Satoshis,
			unlocker,
		)
		if err != nil {
			return nil, err
		}

		// Add 1000 outputs of 100 sats each
		for i := 0; i < leavesPerBranch; i++ {
			if err := tx.PayToAddress(s.publishingAddr, 100); err != nil {
				return nil, err
			}
		}

		// Sign
		if err := tx.Sign(); err != nil {
			return nil, fmt.Errorf("failed to sign leaf tx: %w", err)
		}

		txid := tx.TxID().String()

		// Create 1000 publishing UTXO records
		for i := 0; i < leavesPerBranch; i++ {
			out := tx.Outputs[i]
			scriptHex := hex.EncodeToString(*out.LockingScript)
			utxo := &models.UTXO{
				Outpoint:     fmt.Sprintf("%s:%d", txid, i),
				TxID:         txid,
				Vout:         uint32(i),
				Satoshis:     100,
				ScriptPubKey: scriptHex,
				Status:       models.UTXOStatusAvailable,
				Type:         models.UTXOTypePublishing,
			}

			if err := s.db.InsertUTXO(ctx, utxo); err != nil {
				return nil, err
			}
		}

		leafTxIDs = append(leafTxIDs, txid)

		// Mark branch as spent
		s.db.MarkUTXOSpent(ctx, branch.Outpoint, txid)
	}

	return leafTxIDs, nil
}

// SplitBranchesIntoLeaves takes all available branch UTXOs and splits them into publishing UTXOs
// Each branch (~2M sats) is split into 1,000 publishing UTXOs of 100 sats each
// This is Phase 2 of the tree split
func (s *Splitter) SplitBranchesIntoLeaves(ctx context.Context, arcBroadcastFunc func(context.Context, string) (string, error)) (*SplitResult, error) {
	fmt.Println("🍃 Phase 2: Splitting branch UTXOs into publishing leaves...")

	// Find all available funding UTXOs (branches from Phase 1)
	fundingUTXOs, err := s.db.FindUTXOsByType(ctx, models.UTXOTypeFunding, models.UTXOStatusAvailable)
	if err != nil {
		return nil, fmt.Errorf("failed to find funding UTXOs: %w", err)
	}

	if len(fundingUTXOs) == 0 {
		return nil, fmt.Errorf("no funding UTXOs available for splitting")
	}

	fmt.Printf("   Found %d branch UTXOs to split\n", len(fundingUTXOs))

	leafTxIDs := make([]string, 0, len(fundingUTXOs))
	totalPublishingUTXOs := 0

	// Process each branch
	for idx, branch := range fundingUTXOs {
		fmt.Printf("   Processing branch %d/%d: %s (%d sats)\n", idx+1, len(fundingUTXOs), branch.Outpoint, branch.Satoshis)

		// Lock the UTXO
		if err := s.db.LockUTXO(ctx, branch.Outpoint); err != nil {
			fmt.Printf("   ⚠️  Warning: failed to lock UTXO %s: %v\n", branch.Outpoint, err)
			continue
		}

		// Calculate how many 100-sat outputs we can create
		const publishingSats = uint64(100)
		const maxOutputsPerTx = 500                     // Keep transactions reasonable size
		estimatedTxSize := 192 + (maxOutputsPerTx * 34) // ~1 input + outputs
		estimatedFee := uint64(float64(estimatedTxSize) * s.feeRate)
		availableForLeaves := branch.Satoshis - estimatedFee

		if availableForLeaves < publishingSats {
			fmt.Printf("   ⚠️  Skipping: not enough sats for even one leaf\n")
			s.db.UnlockUTXO(ctx, branch.Outpoint)
			continue
		}

		totalLeavesCount := int(availableForLeaves / publishingSats)
		if totalLeavesCount > 500 {
			totalLeavesCount = 500 // Cap at 500 to keep tx size reasonable (~17KB)
		}

		fmt.Printf("   Creating %d publishing UTXOs (100 sats each)\n", totalLeavesCount)

		// Create transaction
		tx := transaction.NewTransaction()

		// Add input from branch UTXO
		unlocker, err := p2pkh.Unlock(s.fundingKey.PrivateKey, nil)
		if err != nil {
			s.db.UnlockUTXO(ctx, branch.Outpoint)
			return nil, fmt.Errorf("failed to create unlocker: %w", err)
		}

		if err := tx.AddInputFrom(branch.TxID, branch.Vout, branch.ScriptPubKey, branch.Satoshis, unlocker); err != nil {
			s.db.UnlockUTXO(ctx, branch.Outpoint)
			return nil, fmt.Errorf("failed to add input: %w", err)
		}

		// Add publishing outputs
		for i := 0; i < totalLeavesCount; i++ {
			if err := tx.PayToAddress(s.publishingAddr, publishingSats); err != nil {
				s.db.UnlockUTXO(ctx, branch.Outpoint)
				return nil, fmt.Errorf("failed to add output %d: %w", i, err)
			}
		}

		// Calculate change manually and add it
		totalOutputValue := uint64(totalLeavesCount) * publishingSats
		txSize := tx.Size() + 34 // Estimate for change output
		fee := uint64(float64(txSize) * s.feeRate)
		if fee < 1 {
			fee = 1 // Minimum fee
		}

		changeAmount := branch.Satoshis - totalOutputValue - fee
		if changeAmount > 546 { // Dust limit
			if err := tx.PayToAddress(s.fundingKey.Address, changeAmount); err != nil {
				s.db.UnlockUTXO(ctx, branch.Outpoint)
				return nil, fmt.Errorf("failed to add change output: %w", err)
			}
		}

		// Sign
		if err := tx.Sign(); err != nil {
			s.db.UnlockUTXO(ctx, branch.Outpoint)
			return nil, fmt.Errorf("failed to sign: %w", err)
		}

		// Get transaction hex (use EF for ARC)
		var rawHex string
		ef, err2 := tx.EF()
		if err2 == nil {
			rawHex = hex.EncodeToString(ef)
		} else {
			rawHex = tx.String()
		}

		txid := tx.TxID().String()
		fmt.Printf("   Transaction: %s (%d bytes)\n", txid, len(rawHex)/2)

		// Broadcast via ARC
		fmt.Println("   📡 Broadcasting...")
		broadcastedTxID, err := arcBroadcastFunc(ctx, rawHex)
		if err != nil {
			s.db.UnlockUTXO(ctx, branch.Outpoint)
			return nil, fmt.Errorf("ARC broadcast failed for branch %s: %w", branch.Outpoint, err)
		}

		fmt.Printf("   ✅ Broadcast successful: %s\n", broadcastedTxID)

		// Mark branch as spent
		if err := s.db.MarkUTXOSpent(ctx, branch.Outpoint, txid); err != nil {
			return nil, fmt.Errorf("failed to mark UTXO spent: %w", err)
		}

		// Create publishing UTXO records
		for i := 0; i < totalLeavesCount; i++ {
			out := tx.Outputs[i]
			scriptHex := hex.EncodeToString(*out.LockingScript)

			utxo := &models.UTXO{
				Outpoint:     fmt.Sprintf("%s:%d", txid, i),
				TxID:         txid,
				Vout:         uint32(i),
				Satoshis:     publishingSats,
				ScriptPubKey: scriptHex,
				Status:       models.UTXOStatusAvailable,
				Type:         models.UTXOTypePublishing,
			}

			if err := s.db.InsertUTXO(ctx, utxo); err != nil {
				return nil, fmt.Errorf("failed to insert publishing UTXO: %w", err)
			}
		}

		leafTxIDs = append(leafTxIDs, txid)
		totalPublishingUTXOs += totalLeavesCount
	}

	fmt.Printf("✓ Phase 2 complete: Created %d publishing UTXOs across %d transactions\n", totalPublishingUTXOs, len(leafTxIDs))

	return &SplitResult{
		LeafTxIDs:  leafTxIDs,
		TotalUTXOs: totalPublishingUTXOs,
	}, nil
}

// SplitIntoFiftyBranches takes a single funding UTXO and splits it into 50 branch UTXOs
// Each branch will be ~2,000,000 sats (adjustable based on input amount and fees)
// This is Phase 1 of the tree split - Phase 2 will split each branch into 1,000 leaves
func (s *Splitter) SplitIntoFiftyBranches(ctx context.Context, arcBroadcastFunc func(context.Context, string) (string, error)) (*SplitResult, error) {
	fmt.Println("🌳 Phase 1: Splitting funding UTXO into 50 branches...")

	// Find a funding UTXO
	fundingUTXO, err := s.db.FindAndLockUTXO(ctx, models.UTXOTypeFunding)
	if err != nil {
		return nil, fmt.Errorf("no funding UTXO available: %w", err)
	}

	fmt.Printf("   Using funding UTXO: %s (%d sats)\n", fundingUTXO.Outpoint, fundingUTXO.Satoshis)

	// Calculate branch size
	const branchCount = 50
	totalInput := fundingUTXO.Satoshis

	// Estimate tx size: 1 input (~150 bytes) + 50 outputs (~34 bytes each) + overhead (~10 bytes)
	// = 150 + (50 * 34) + 10 = 1860 bytes
	estimatedSize := uint64(1860)
	estimatedFee := uint64(float64(estimatedSize) * s.feeRate)

	// Amount available for branches after fee
	availableForBranches := totalInput - estimatedFee

	// Each branch gets equal amount
	branchAmount := availableForBranches / branchCount

	fmt.Printf("   Total input: %d sats\n", totalInput)
	fmt.Printf("   Estimated fee: %d sats\n", estimatedFee)
	fmt.Printf("   Each branch: %d sats\n", branchAmount)

	// Create transaction
	tx := transaction.NewTransaction()

	// Add input from funding UTXO
	unlocker, err := p2pkh.Unlock(s.fundingKey.PrivateKey, nil)
	if err != nil {
		s.db.UnlockUTXO(ctx, fundingUTXO.Outpoint)
		return nil, fmt.Errorf("failed to create unlocker: %w", err)
	}

	if err := tx.AddInputFrom(
		fundingUTXO.TxID,
		fundingUTXO.Vout,
		fundingUTXO.ScriptPubKey,
		fundingUTXO.Satoshis,
		unlocker,
	); err != nil {
		// Unlock on failure
		s.db.UnlockUTXO(ctx, fundingUTXO.Outpoint)
		return nil, fmt.Errorf("failed to add input: %w", err)
	}

	// Add 50 branch outputs to funding address
	for i := 0; i < branchCount; i++ {
		if err := tx.PayToAddress(s.fundingKey.Address, branchAmount); err != nil {
			s.db.UnlockUTXO(ctx, fundingUTXO.Outpoint)
			return nil, fmt.Errorf("failed to add output %d: %w", i, err)
		}
	}

	// Sign transaction
	if err := tx.Sign(); err != nil {
		s.db.UnlockUTXO(ctx, fundingUTXO.Outpoint)
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Get raw hex (use Extended Format for ARC)
	var rawHex string
	ef, err := tx.EF()
	if err == nil {
		rawHex = hex.EncodeToString(ef)
		fmt.Println("   Using Extended Format (EF)")
	} else {
		rawHex = tx.String()
		fmt.Println("   Using standard format")
	}

	txid := tx.TxID().String()

	fmt.Printf("   Transaction created: %s\n", txid)
	fmt.Printf("   Size: %d bytes\n", len(rawHex)/2)

	// Broadcast via ARC
	fmt.Println("   📡 Broadcasting to ARC...")
	broadcastedTxID, err := arcBroadcastFunc(ctx, rawHex)
	if err != nil {
		s.db.UnlockUTXO(ctx, fundingUTXO.Outpoint)
		return nil, fmt.Errorf("ARC broadcast failed: %w", err)
	}

	fmt.Printf("   ✓ Broadcast successful: %s\n", broadcastedTxID)

	// Mark original UTXO as spent
	if err := s.db.MarkUTXOSpent(ctx, fundingUTXO.Outpoint, txid); err != nil {
		return nil, fmt.Errorf("failed to mark UTXO spent: %w", err)
	}

	// Create 50 new branch UTXO records
	for i := 0; i < branchCount; i++ {
		out := tx.Outputs[i]
		scriptHex := hex.EncodeToString(*out.LockingScript)

		branchUTXO := &models.UTXO{
			Outpoint:     fmt.Sprintf("%s:%d", txid, i),
			TxID:         txid,
			Vout:         uint32(i),
			Satoshis:     branchAmount,
			ScriptPubKey: scriptHex,
			Status:       models.UTXOStatusAvailable,
			Type:         models.UTXOTypeFunding, // Still funding type, will split further
		}

		if err := s.db.InsertUTXO(ctx, branchUTXO); err != nil {
			return nil, fmt.Errorf("failed to insert branch UTXO %d: %w", i, err)
		}
	}

	fmt.Printf("✓ Phase 1 complete: Created %d branch UTXOs\n", branchCount)

	return &SplitResult{
		BranchTxIDs: []string{txid},
		LeafTxIDs:   []string{},
		TotalUTXOs:  branchCount,
	}, nil
}

// CheckAndRefill checks if publishing UTXO count is below threshold and triggers split
func (s *Splitter) CheckAndRefill(ctx context.Context, minCount int) error {
	stats, err := s.db.GetUTXOStats(ctx)
	if err != nil {
		return err
	}

	availablePublishing := stats["publishing_available"]

	if availablePublishing < int64(minCount) {
		fmt.Printf("⚠️  Low on publishing UTXOs (%d available, need %d)\n", availablePublishing, minCount)
		fmt.Println("   Triggering refill...")

		needed := minCount - int(availablePublishing)
		_, err := s.CreatePublishingUTXOs(ctx, needed)
		return err
	}

	fmt.Printf("✓ Publishing UTXO pool healthy: %d available\n", availablePublishing)
	return nil
}
