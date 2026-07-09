package chain

import (
	"testing"
	"toy-blockchain/ledger"
)

// TestNewBlockchain verifies that the chain initializes with exactly one block (Genesis)
func TestNewBlockchain(t *testing.T) {
	bc := NewBlockchain(3)

	if len(bc.Blocks) != 1 {
		t.Fatalf("Expected initial chain length to be 1, got %d", len(bc.Blocks))
	}

	if bc.Blocks[0].Index != 0 {
		t.Errorf("Expected Genesis block index to be 0, got %d", bc.Blocks[0].Index)
	}
}

// TestTransactionRejection verifies that overspending or malformed amounts are rejected (FR-4)
func TestTransactionRejection(t *testing.T) {
	bc := NewBlockchain(2)

	// 1. Add funds via faucet
	_ = bc.AddTransaction(ledger.NewTransaction("faucet", "Bob", 100.0))
	_, _ = bc.MinePendingBlock()

	// 2. Test overspending (Bob has 100, attempts to send 150)
	err := bc.AddTransaction(ledger.NewTransaction("Bob", "Bob", 150.0))
	if err == nil {
		t.Error("Expected error when overspending, but transaction was accepted")
	}

	// 3. Test malformed amount (negative value)
	err = bc.AddTransaction(ledger.NewTransaction("Bob", "Bob", -50.0))
	if err == nil {
		t.Error("Expected error for negative transaction amount, but it was accepted")
	}

	// Verify balance remains unchanged after failed transactions
	balances := bc.GetBalances()
	if balances["Bob"] != 100.0 {
		t.Errorf("Expected Bob's balance to remain 100.0, got %.2f", balances["Bob"])
	}
}

// TestTamperDetection verifies that altering historical transactions breaks chain validation (FR-6)
func TestTamperDetection(t *testing.T) {
	bc := NewBlockchain(2)

	// Build a small history
	_ = bc.AddTransaction(ledger.NewTransaction("faucet", "Bob", 200.0))
	_, _ = bc.MinePendingBlock()

	_ = bc.AddTransaction(ledger.NewTransaction("Bob", "Bob", 50.0))
	_, _ = bc.MinePendingBlock()

	// Verify it validates initially
	valid, _, _ := bc.ValidateChain()
	if !valid {
		t.Fatal("Honest chain failed validation unexpectedly")
	}

	// Deliberately tamper with Block 1 transaction data (FR-6, Research 7.1)
	bc.Blocks[1].Transactions[0].Amount = 999.0

	// Re-run validation
	valid, brokenIdx, err := bc.ValidateChain()
	if valid {
		t.Error("Validation passed despite history being tampered with")
	}
	if brokenIdx != 1 {
		t.Errorf("Expected validation to fail at block 1, failed at block %d", brokenIdx)
	}
	if err == nil {
		t.Errorf("Expected an explicit error detailing the mismatch")
	}
}
