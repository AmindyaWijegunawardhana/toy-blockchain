package block

import (
	"testing"
	"toy-blockchain/ledger"
)

func TestDeterministicHashing(t *testing.T) {
	txs := []ledger.Transaction{
		ledger.NewTransaction("Alice", "Bob", 50.0),
	}

	b1 := NewBlock(1, txs, "mock-prevhash")

	b2 := &Block{
		Index:        b1.Index,
		Timestamp:    b1.Timestamp,
		Transactions: b1.Transactions,
		PrevHash:     b1.PrevHash,
		Nonce:        b1.Nonce,
	}

	b2.Hash = b2.CalculateHash()

	if b1.Hash != b2.Hash {
		t.Errorf("Hashhing is non-deterministic! Got %s and %s", b1.Hash, b2.Hash)
	}
}
