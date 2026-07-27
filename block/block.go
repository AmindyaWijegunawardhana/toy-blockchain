package block

import (
	"crypto/sha256"
	"encoding/hex"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"
	"toy-blockchain/ledger"
)

// Block represents a single verified node in the blockchain.
type Block struct {
	Index        int                  `json:"index"`
	Timestamp    int64                `json:"timestamp"`
	PrevHash     string               `json:"prev_hash"`
	Hash         string               `json:"hash"`
	Nonce        int                  `json:"nonce"`
	Transactions []ledger.Transaction `json:"transactions"`
	MerkleRoot   string               `json:"merkle_root"`
}

// NewBlock constructs a fully populated block with a derived Merkle Root.
func NewBlock(index int, transactions []ledger.Transaction, prevHash string) *Block {
	b := &Block{
		Index:        index,
		Timestamp:    time.Now().Unix(),
		PrevHash:     prevHash,
		Transactions: transactions,
	}
	b.MerkleRoot = b.CalculateMerkleRoot()
	return b
}

// NewGenesisBlock builds a static, deterministic starting block.
func NewGenesisBlock() *Block {
	genesisTx := ledger.NewTransaction("system", "faucet", 1000000)

	b := &Block{
		Index:        0,
		Timestamp:    1700000000,
		PrevHash:     "0000000000000000000000000000000000000000000000000000000000000000",
		Transactions: []ledger.Transaction{genesisTx},
	}
	b.MerkleRoot = b.CalculateMerkleRoot()
	b.Hash = b.CalculateHash()
	return b
}

// CalculateHashWithNonce computes the hash over header fields given a specific nonce value.
func (b *Block) CalculateHashWithNonce(nonce int) string {
	record := strconv.Itoa(b.Index) +
		strconv.FormatInt(b.Timestamp, 10) +
		b.PrevHash +
		b.MerkleRoot +
		strconv.Itoa(nonce)

	hash := sha256.Sum256([]byte(record))
	return hex.EncodeToString(hash[:])
}

// CalculateHash computes the block hash over header fields using the current internal Nonce state.
func (b *Block) CalculateHash() string {
	return b.CalculateHashWithNonce(b.Nonce)
}

// Mine searches the nonce space concurrently across multiple goroutines, stopping cleanly when one succeeds.
func (b *Block) Mine(difficulty int) {
	target := ""
	for i := 0; i < difficulty; i++ {
		target += "0"
	}

	// Use all available logical CPU cores on the local machine
	numWorkers := runtime.NumCPU()

	// Create channels to handle success results safely
	type MiningResult struct {
		Nonce int
		Hash  string
	}
	resultChan := make(chan MiningResult, 1)

	// Thread-safe atomic flag to cleanly tell all other workers to stop spinning instantly
	var found uint32 = 0

	// Launch parallel workers across divided segments of the nonce range
	for w := 0; w < numWorkers; w++ {
		// Each worker starts at its worker index ID and jumps by the worker step factor
		go func(workerID int, step int) {
			currentNonce := workerID

			for {
				// Periodically check if another worker already hit the jackpot
				if atomic.LoadUint32(&found) == 1 {
					return
				}

				hash := b.CalculateHashWithNonce(currentNonce)
				if hash[:difficulty] == target {
					// Attempt to flip the atomic bit to claim the discovery victory
					if atomic.CompareAndSwapUint32(&found, 0, 1) {
						resultChan <- MiningResult{Nonce: currentNonce, Hash: hash}
					}
					return
				}

				// Increment by the total worker stride to prevent workers from cross-scanning duplicated nonces
				currentNonce += step
			}
		}(w, numWorkers)
	}

	// Block until the winning result arrives from the fastest active thread
	winningResult := <-resultChan

	// Lock the winning metrics right into the primary block context configuration
	b.Nonce = winningResult.Nonce
	b.Hash = winningResult.Hash
}

// CalculateMerkleRoot recursively computes the Merkle Root of the block's transactions.
func (b *Block) CalculateMerkleRoot() string {
	if len(b.Transactions) == 0 {
		emptyHash := sha256.Sum256([]byte(""))
		return hex.EncodeToString(emptyHash[:])
	}

	var level [][]byte
	for _, tx := range b.Transactions {
		level = append(level, tx.GetHash())
	}

	for len(level) > 1 {
		var nextLevel [][]byte

		if len(level)%2 != 0 {
			level = append(level, level[len(level)-1])
		}

		for i := 0; i < len(level); i += 2 {
			concat := append(level[i], level[i+1]...)
			parentHash := sha256.Sum256(concat)
			nextLevel = append(nextLevel, parentHash[:])
		}
		level = nextLevel
	}

	return hex.EncodeToString(level[0])
}
