package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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

// BitailsResponse represents the response from Bitails
type BitailsResponse struct {
	Address    string        `json:"address"`
	ScriptHash string        `json:"scripthash"`
	Unspent    []BitailsUTXO `json:"unspent"`
}

func main() {
	publishingAddr := "12w4BoPtqCt7EFLmUPi9GLmpbZ1CHdPvzj"

	// Fetch UTXOs
	url := fmt.Sprintf("https://api.bitails.io/address/%s/unspent?limit=100000", publishingAddr)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Failed to fetch UTXOs: %v", err)
	}
	defer resp.Body.Close()

	var bitailsResp BitailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&bitailsResp); err != nil {
		log.Fatalf("Failed to decode response: %v", err)
	}

	utxos := bitailsResp.Unspent
	fmt.Printf("Total UTXOs: %d\n", len(utxos))

	totalSats := uint64(0)
	for _, u := range utxos {
		totalSats += u.Satoshis
	}
	fmt.Printf("Total satoshis: %d (%.8f BSV)\n", totalSats, float64(totalSats)/1e8)

	// Calculate optimal split for 50k target
	fmt.Printf("\nTo reach 50k UTXOs with consolidation:\n")
	fmt.Printf("  Current: 25,000 × 100 sats = 2,500,000 sats total\n")
	fmt.Printf("  Need additional: 2,500,000 sats to reach 50,000 × 100 sats\n")
	fmt.Printf("  Send 2,500,000 sats from publishing address back to funding\n")
	fmt.Printf("  Then run Phase 1 on funding address to split into 2 branches\n")
	fmt.Printf("  Then run Phase 2 on those 2 branches to get 1000 more publishing UTXOs\n")

	os.Exit(0)
}
