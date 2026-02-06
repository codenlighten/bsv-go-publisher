package main

import (
	"fmt"
	"log"
	"os"

	"github.com/akua/bsv-broadcaster/internal/bsv"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/p2pkh"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Load publishing key
	publishingKey, err := bsv.LoadOrGenerateKeyPair("PUBLISHING_PRIVKEY")
	if err != nil {
		log.Fatalf("Failed to load publishing key: %v", err)
	}

	fundingAddr := os.Getenv("FUNDING_ADDRESS")
	if fundingAddr == "" {
		log.Fatal("Missing FUNDING_ADDRESS")
	}

	fmt.Printf("Sending from: %s\n", publishingKey.Address)
	fmt.Printf("Sending to: %s\n", fundingAddr)

	// UTXO to spend: ee6ca3bfee73f54af85d04a5aa314f6fd2d9ed5f8ae552d52edb975015e6ceec:0
	txid := "ee6ca3bfee73f54af85d04a5aa314f6fd2d9ed5f8ae552d52edb975015e6ceec"
	vout := uint32(0)
	satoshis := uint64(100000000) // 1 BSV

	// Get locking script for publishing address
	pubAddr, err := script.NewAddressFromPublicKey(publishingKey.PublicKey, true)
	if err != nil {
		log.Fatalf("Failed to get address: %v", err)
	}
	lockingScript, err := p2pkh.Lock(pubAddr)
	if err != nil {
		log.Fatalf("Failed to create locking script: %v", err)
	}

	// Create transaction
	tx := transaction.NewTransaction()

	// Add input from publishing address
	unlocker, err := p2pkh.Unlock(publishingKey.PrivateKey, nil)
	if err != nil {
		log.Fatalf("Failed to create unlocker: %v", err)
	}

	if err := tx.AddInputFrom(txid, vout, lockingScript.String(), satoshis, unlocker); err != nil {
		log.Fatalf("Failed to add input: %v", err)
	}

	// Send to funding address (minus fee)
	fee := uint64(200) // ~200 sats for simple transaction
	amount := satoshis - fee

	if err := tx.PayToAddress(fundingAddr, amount); err != nil {
		log.Fatalf("Failed to add output: %v", err)
	}

	// Sign
	if err := tx.Sign(); err != nil {
		log.Fatalf("Failed to sign: %v", err)
	}

	fmt.Println("\n✅ Transaction created successfully!")
	fmt.Printf("TXID: %s\n", tx.TxID())
	fmt.Printf("Size: %d bytes\n", len(tx.String())/2)
	fmt.Printf("Amount: %.8f BSV\n", float64(amount)/100000000)
	fmt.Printf("\nRaw hex:\n%s\n", tx.String())
	fmt.Println("\n📡 To broadcast, run:")
	fmt.Printf("curl -X POST https://arc.gorillapool.io/v1/tx -H \"Content-Type: text/plain\" -d '%s'\n", tx.String())
}
