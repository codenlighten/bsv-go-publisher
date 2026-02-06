package bsv

import (
	"encoding/hex"
	"fmt"
	"os"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/script"
)

// KeyPair represents a BSV private/public keypair
type KeyPair struct {
	PrivateKey *ec.PrivateKey
	PublicKey  *ec.PublicKey
	Address    string
	WIF        string
}

// GenerateKeyPair creates a new random BSV keypair
func GenerateKeyPair() (*KeyPair, error) {
	// Generate new private key
	privKey, err := ec.NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Get public key
	pubKey := privKey.PubKey()

	// Generate P2PKH address
	pubKeyHash, err := script.NewAddressFromPublicKey(pubKey, true) // mainnet
	if err != nil {
		return nil, fmt.Errorf("failed to create address: %w", err)
	}

	// Get WIF
	wifStr := privKey.Wif()

	return &KeyPair{
		PrivateKey: privKey,
		PublicKey:  pubKey,
		Address:    pubKeyHash.AddressString,
		WIF:        wifStr,
	}, nil
}

// LoadOrGenerateKeyPair loads a keypair from env var or generates a new one
func LoadOrGenerateKeyPair(envVarName string) (*KeyPair, error) {
	privKeyWIF := os.Getenv(envVarName)

	if privKeyWIF == "" {
		// Generate new keypair
		kp, err := GenerateKeyPair()
		if err != nil {
			return nil, err
		}

		// Print warning to set it in .env
		fmt.Printf("\n⚠️  WARNING: No %s found in environment!\n", envVarName)
		fmt.Printf("Generated new keypair:\n")
		fmt.Printf("  Address: %s\n", kp.Address)
		fmt.Printf("  Private Key (WIF): %s\n", kp.WIF)
		fmt.Printf("\nAdd this to your .env file:\n")
		fmt.Printf("%s=%s\n\n", envVarName, kp.WIF)

		return kp, nil
	}

	// Load existing private key from WIF
	privKey, err := ec.PrivateKeyFromWif(privKeyWIF)
	if err != nil {
		// Try hex format as fallback
		privKeyBytes, err2 := hex.DecodeString(privKeyWIF)
		if err2 != nil {
			return nil, fmt.Errorf("failed to decode private key (tried WIF and hex): %w", err)
		}
		privKey, _ = ec.PrivateKeyFromBytes(privKeyBytes)
	}

	pubKey := privKey.PubKey()
	address, err := script.NewAddressFromPublicKey(pubKey, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create address: %w", err)
	}

	wifStr := privKey.Wif()

	fmt.Printf("✓ Loaded %s: %s\n", envVarName, address.AddressString)

	return &KeyPair{
		PrivateKey: privKey,
		PublicKey:  pubKey,
		Address:    address.AddressString,
		WIF:        wifStr,
	}, nil
}

// Sign signs a transaction hash with the private key
func (kp *KeyPair) Sign(hash []byte) ([]byte, error) {
	sig, err := kp.PrivateKey.Sign(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}
	return sig.Serialize(), nil
}
