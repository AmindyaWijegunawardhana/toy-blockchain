package block

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
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

// HashInput defines the structure used strictly for computing the block's hash.
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

// NewGenesisBlock generates the initial, deterministic block 0 of the chain.
func NewGenesisBlock() *Block {
	b := &Block{
		Index:        0,
		Timestamp:    1719878400, // Fixed Unix timestamp
		Transactions: []ledger.Transaction{},
		PrevHash:     "0000000000000000000000000000000000000000000000000000000000000000",
		Nonce:        0,
	}
	b.Hash = b.CalculateHash()
	return b
}

// CalculateHash computes the SHA-256 hash over a stable JSON serialization.
func (b *Block) CalculateHash() string {
	input := HashInput{
		Index:        b.Index,
		Timestamp:    b.Timestamp,
		Transactions: b.Transactions,
		PrevHash:     b.PrevHash,
		Nonce:        b.Nonce,
	}

	data, err := json.Marshal(input)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal block data: %v", err))
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// Mine increments the block's nonce until its SHA-256 hash satisfies
// the proof-of-work difficulty target (N leading zero hex characters).
func (b *Block) Mine(difficulty int) {
	target := strings.Repeat("0", difficulty)
	startTime := time.Now()

	fmt.Printf("Mining block %d (Difficulty: %d)...\n", b.Index, difficulty)

	for {
		b.Hash = b.CalculateHash()
		if strings.HasPrefix(b.Hash, target) {
			break
		}
		b.Nonce++
	}

	elapsed := time.Since(startTime)
	fmt.Printf("Block %d successfully mined!\n", b.Index)
	fmt.Printf("  Nonce found:  %d\n", b.Nonce)
	fmt.Printf("  Time elapsed: %s\n\n", elapsed)
}
