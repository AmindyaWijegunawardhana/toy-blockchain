package chain

import (
	"strings"
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

// TestMiningMeetsTarget verifies that mined blocks meet the difficulty target (FR-5)
func TestMiningMeetsTarget(t *testing.T) {
	difficulty := 3
	bc := NewBlockchain(difficulty)
	targetPrefix := strings.Repeat("0", difficulty)

	// Add a valid transaction and mine
	err := bc.AddTransaction(ledger.NewTransaction("faucet", "Amindya", 100.0))
	if err != nil {
		t.Fatalf("Failed to add transaction: %v", err)
	}

	minedBlock, err := bc.MinePendingBlock()
	if err != nil {
		t.Fatalf("Mining failed: %v", err)
	}

	// Verify the hash prefix satisfies the PoW rules
	if !strings.HasPrefix(minedBlock.Hash, targetPrefix) {
		t.Errorf("Mined block hash %s does not match difficulty target prefix %s", minedBlock.Hash, targetPrefix)
	}

	// Verify full chain validation passes on the honest chain
	valid, brokenIdx, valErr := bc.ValidateChain()
	if !valid {
		t.Errorf("Honest chain validation failed at index %d: %v", brokenIdx, valErr)
	}
}
