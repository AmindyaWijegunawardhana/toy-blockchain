package block

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
	"toy-blockchain/ledger"
)

// Block represents a single block in the append-only chain.
type Block struct {
	Index        int                  `json:"index"`
	Timestamp    int64                `json:"timestamp"`
	Transactions []ledger.Transaction `json:"transactions"`
	PrevHash     string               `json:"prev_hash"`
	Nonce        int64                `json:"nonce"`
	Hash         string               `json:"hash"`
}

type HashInput struct {
	Index        int                  `json:"index"`
	Timestamp    int64                `json:"timestamp"`
	Transactions []ledger.Transaction `json:"transactions"`
	PrevHash     string               `json:"prev_hash"`
	Nonce        int64                `json:"nonce"`
}

// NewBlock initializes a new block with the given parameters.
func NewBlock(index int, transactions []ledger.Transaction, prevHash string) *Block {
	b := &Block{
		Index:        index,
		Timestamp:    time.Now().Unix(),
		Transactions: transactions,
		PrevHash:     prevHash,
		Nonce:        0,
	}
	b.Hash = b.CalculateHash()
	return b
}

// CalculateHash computes the SHA-256 hash over a stable JSON serialization
// of the block's core fields (excluding the hash field itself).
func (b *Block) CalculateHash() string {
	input := HashInput{
		Index:        b.Index,
		Timestamp:    b.Timestamp,
		Transactions: b.Transactions,
		PrevHash:     b.PrevHash,
		Nonce:        b.Nonce,
	}

	// encoding/json provides a stable, deterministic serialization for structs
	data, err := json.Marshal(input)
	if err != nil {
		// Panic is acceptable here as failure to marshal standard types indicates a severe system runtime issue
		panic(fmt.Sprintf("failed to marshal block data: %v", err))
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

func NewGenesisBlock() *Block {
	b := &Block{
		Index:        0,
		Timestamp:    1719878400,
		Transactions: []ledger.Transaction{},
		PrevHash:     "00000000000000000000000000000000000000000000000000000000000000",
		Nonce:        0,
	}
	b.Hash = b.CalculateHash()
	return b
}
