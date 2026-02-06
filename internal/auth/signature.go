package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
)

// VerifySignature ensures the 'data' was signed by the 'pubKey'
// Returns true if the signature is valid for the given data and public key
func VerifySignature(pubKeyHex, dataHex, sigHex string) (bool, error) {
	// Decode public key
	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return false, fmt.Errorf("invalid public key hex: %w", err)
	}

	pubKey, err := ec.ParsePubKey(pubKeyBytes)
	if err != nil {
		return false, fmt.Errorf("failed to parse public key: %w", err)
	}

	// Decode signature
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return false, fmt.Errorf("invalid signature hex: %w", err)
	}

	sig, err := ec.ParseDERSignature(sigBytes)
	if err != nil {
		return false, fmt.Errorf("failed to parse signature: %w", err)
	}

	// Decode and hash the data (Bitcoin uses double SHA-256)
	dataBytes, err := hex.DecodeString(dataHex)
	if err != nil {
		return false, fmt.Errorf("invalid data hex: %w", err)
	}

	// Double SHA-256 hash
	hash1 := sha256.Sum256(dataBytes)
	hash2 := sha256.Sum256(hash1[:])

	// Verify signature
	return sig.Verify(hash2[:], pubKey), nil
}
