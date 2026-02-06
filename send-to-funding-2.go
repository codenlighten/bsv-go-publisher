package main

import (
	"context"
	"fmt"
	"log"
	"os"

	goarc "github.com/bsv-blockchain/go-arc/arc"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/p2pkh"
)

func main() {
	// Config
	publishingPrivKey := "L1tvUUBsdYsRt1hbMCLtj1XEHL3XAfrcJKt2x7VxoKrQ8SdfFpxg"
	publishingAddr := "12w4BoPtqCt7EFLmUPi9GLmpbZ1CHdPvzj"
	fundingAddr := "1XJ82FS3QLrXRT6zfrB4W9BanSSgFgw1m"
	amountSats := uint64(2000000) // 2M sats = 0.02 BSV

	fmt.Printf("Sending %d sats from %s → %s\n", amountSats, publishingAddr, fundingAddr)

	// Create transaction
	tx := transaction.NewTransaction()

	// Add input - we'll use UTXOs from publishing address
	// For now, just create a P2PKH input from the first UTXO
	// In reality we'd need to query blockchain for actual UTXO

	privKey, err := transaction.PrivateKeyFromString(publishingPrivKey)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	// For this demo, we'll construct a simple transaction
	// The go-sdk will handle change automatically
	unlocker, err := p2pkh.Unlock(privKey, nil)
	if err != nil {
		log.Fatalf("Failed to create unlocker: %v", err)
	}

	// We need a UTXO from the publishing address
	// Let's use a dummy one for now - in practice this should be queried
	txID := "dd61aed9114a2120933a15b28a51fbf0ab399b44020fb686151b68b91cea6fe2" // Phase 2 tx
	vout := uint32(0)                                                          // First output
	satoshis := uint64(100)

	// Create a simple P2PKH locking script for the publishing address
	addr, err := transaction.AddressFromString(publishingAddr)
	if err != nil {
		log.Fatalf("Failed to parse address: %v", err)
	}

	lockingScript, err := p2pkh.Lock(addr)
	if err != nil {
		log.Fatalf("Failed to create locking script: %v", err)
	}

	if err := tx.AddInputFrom(txID, vout, lockingScript.String(), satoshis, unlocker); err != nil {
		log.Fatalf("Failed to add input: %v", err)
	}

	// Add output to funding address
	if err := tx.PayToAddress(fundingAddr, amountSats); err != nil {
		log.Fatalf("Failed to add output: %v", err)
	}

	// Sign
	if err := tx.Sign(); err != nil {
		log.Fatalf("Failed to sign: %v", err)
	}

	// Broadcast
	txHex := tx.String()
	fmt.Printf("Transaction hex: %s\n", txHex[:100]+"...")
	fmt.Printf("Transaction size: %d bytes\n", len(txHex)/2)
	fmt.Printf("Transaction ID: %s\n", tx.TxID().String())

	// Broadcast to ARC
	client := goarc.NewARCClient(goarc.Production)
	ctx := context.Background()

	responses, err := client.BroadcastBatch(ctx, []string{txHex})
	if err != nil {
		log.Fatalf("Failed to broadcast: %v", err)
	}

	fmt.Printf("ARC Response:\n")
	fmt.Printf("  Status: %s\n", responses[0].TxStatus)
	if responses[0].TxID != "" {
		fmt.Printf("  TX ID: %s\n", responses[0].TxID)
	}
	if responses[0].ExtraInfo != "" {
		fmt.Printf("  Info: %s\n", responses[0].ExtraInfo)
	}

	os.Exit(0)
}
