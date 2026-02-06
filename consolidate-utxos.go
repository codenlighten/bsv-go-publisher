package main

import (
	"context"
	"fmt"
	"log"

	goarc "github.com/bsv-blockchain/go-arc/arc"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/p2pkh"
)

func main() {
	ctx := context.Background()

	publishingAddr := "12w4BoPtqCt7EFLmUPi9GLmpbZ1CHdPvzj"
	fundingAddr := "1XJ82FS3QLrXRT6zfrB4W9BanSSgFgw1m"
	publishingPrivKey := "L1tvUUBsdYsRt1hbMCLtj1XEHL3XAfrcJKt2x7VxoKrQ8SdfFpxg"

	// Need to consolidate publishing UTXOs
	// For now, just use one of the known Phase 2 output UTXOs
	// dd61aed9114a2120933a15b28a51fbf0ab399b44020fb686151b68b91cea6fe2:0-499 are available

	// Create transaction using first few publishing UTXOs to consolidate
	privKey, err := transaction.PrivateKeyFromString(publishingPrivKey)
	if err != nil {
		log.Fatalf("Failed to parse key: %v", err)
	}

	tx := transaction.NewTransaction()

	// Get unlocker
	unlocker, err := p2pkh.Unlock(privKey, nil)
	if err != nil {
		log.Fatalf("Failed to create unlocker: %v", err)
	}

	// Add inputs from published UTXOs (use a few to consolidate)
	inputs := []struct {
		txid string
		vout uint32
		sats uint64
	}{
		{"dd61aed9114a2120933a15b28a51fbf0ab399b44020fb686151b68b91cea6fe2", 0, 100},
		{"dd61aed9114a2120933a15b28a51fbf0ab399b44020fb686151b68b91cea6fe2", 1, 100},
		{"dd61aed9114a2120933a15b28a51fbf0ab399b44020fb686151b68b91cea6fe2", 2, 100},
	}

	// Create P2PKH script for publishing address
	publishingAddrObj, err := transaction.AddressFromString(publishingAddr)
	if err != nil {
		log.Fatalf("Failed to parse address: %v", err)
	}

	scriptPubKey, err := p2pkh.Lock(publishingAddrObj)
	if err != nil {
		log.Fatalf("Failed to create script: %v", err)
	}

	totalInputs := uint64(0)
	for _, inp := range inputs {
		if err := tx.AddInputFrom(inp.txid, inp.vout, scriptPubKey.String(), inp.sats, unlocker); err != nil {
			log.Fatalf("Failed to add input: %v", err)
		}
		totalInputs += inp.sats
	}

	// Send to funding address
	if err := tx.PayToAddress(fundingAddr, totalInputs-1000); err != nil {
		log.Fatalf("Failed to add output: %v", err)
	}

	// Sign
	if err := tx.Sign(); err != nil {
		log.Fatalf("Failed to sign: %v", err)
	}

	txHex := tx.String()
	fmt.Printf("Consolidation TX: %s\n", tx.TxID().String())
	fmt.Printf("Size: %d bytes\n", len(txHex)/2)

	// Broadcast
	client := goarc.NewARCClient(goarc.Production)
	responses, err := client.BroadcastBatch(ctx, []string{txHex})
	if err != nil {
		log.Fatalf("Broadcast failed: %v", err)
	}

	fmt.Printf("Status: %s\n", responses[0].TxStatus)
	if responses[0].ExtraInfo != "" {
		fmt.Printf("Info: %s\n", responses[0].ExtraInfo)
	}
}
