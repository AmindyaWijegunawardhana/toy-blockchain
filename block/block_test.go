package block

import (
	"strings"
	"testing"
	"toy-blockchain/ledger"
)

func TestDeterministicHashing(t *testing.T) {
	b1 := NewGenesisBlock()
	b2 := NewGenesisBlock()

	if b1.MerkleRoot != b2.MerkleRoot {
		t.Errorf("Mismatch in Merkle Root determinism: got %s and %s", b1.MerkleRoot, b2.MerkleRoot)
	}
}

func TestMerkleRootTampering(t *testing.T) {
	txs := []ledger.Transaction{
		ledger.NewTransaction("Alice", "Bob", 100),
		ledger.NewTransaction("Bob", "Charlie", 50),
	}

	b := NewBlock(1, txs, "prev-hash-placeholder")
	originalRoot := b.CalculateMerkleRoot()

	b.Transactions[1].Amount = 99999
	tamperedRoot := b.CalculateMerkleRoot()

	if originalRoot == tamperedRoot {
		t.Error("CRITICAL SECURITY ERROR: Merkle Root failed to detect transaction tampering!")
	}
}

// TestConcurrentMining asserts that parallel workers reliably secure proof of work targets
func TestConcurrentMining(t *testing.T) {
	txs := []ledger.Transaction{
		ledger.NewTransaction("faucet", "Alice", 250),
	}

	difficulty := 3
	b := NewBlock(1, txs, "00000000000000000000000000000000")

	// Fire off concurrent multi-threaded mining
	b.Mine(difficulty)

	expectedPrefix := strings.Repeat("0", difficulty)
	if !strings.HasPrefix(b.Hash, expectedPrefix) {
		t.Errorf("Concurrent mining output hash failed to meet difficulty rules. Got: %s", b.Hash)
	}

	// Verify that the saved final nonce actually evaluates to the saved hash
	recalculatedHash := b.CalculateHash()
	if b.Hash != recalculatedHash {
		t.Error("Structural validation failure: Nonce discovered by winning worker does not align with block hash results.")
	}
}
