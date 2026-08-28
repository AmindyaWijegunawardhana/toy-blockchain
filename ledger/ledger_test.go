package ledger

import (
	"testing"
)

func TestWalletCreationAndSignatures(t *testing.T) {
	alice, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create Alice wallet: %v", err)
	}
	bob, err := NewWallet()
	if err != nil {
		t.Fatalf("Failed to create Bob wallet: %v", err)
	}

	tx, err := SignTransaction(alice, bob.Address(), 250)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	if err := VerifyTransaction(tx); err != nil {
		t.Fatalf("Expected valid transaction signature, got error: %v", err)
	}

	// Tamper with transaction amount
	tx.Amount = 999
	if err := VerifyTransaction(tx); err == nil {
		t.Errorf("Expected verification to fail after payload tampering")
	}
}
