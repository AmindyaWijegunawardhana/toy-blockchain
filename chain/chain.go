package chain

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"toy-blockchain/block"
	"toy-blockchain/ledger"
)

// Blockchain manages the sequential chain of blocks and ledger state with full thread safety.
type Blockchain struct {
	Blocks              []*block.Block
	PendingTransactions []ledger.Transaction
	Difficulty          int
	mu                  sync.RWMutex
}

// NewBlockchain creates a new chain instance with a Genesis block.
func NewBlockchain(difficulty int) *Blockchain {
	genesis := &block.Block{
		Index:        0,
		Timestamp:    0,
		Transactions: []ledger.Transaction{},
		PrevHash:     "0",
		Nonce:        0,
	}
	genesis.Mine(difficulty)

	return &Blockchain{
		Blocks:              []*block.Block{genesis},
		PendingTransactions: make([]ledger.Transaction, 0),
		Difficulty:          difficulty,
	}
}

// LatestBlock returns a thread-safe copy of the most recent block on the chain.
func (bc *Blockchain) LatestBlock() *block.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	tip := bc.Blocks[len(bc.Blocks)-1]
	copied := *tip
	return &copied
}

// GetBlocks returns a thread-safe copy slice of the current block pointers.
func (bc *Blockchain) GetBlocks() []*block.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	blocksCopy := make([]*block.Block, len(bc.Blocks))
	copy(blocksCopy, bc.Blocks)
	return blocksCopy
}

// BlockCount returns the current length of the chain.
func (bc *Blockchain) BlockCount() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return len(bc.Blocks)
}

// AddBlock appends a validated block directly to the chain.
func (bc *Blockchain) AddBlock(b *block.Block) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.Blocks = append(bc.Blocks, b)
}

// AddTransaction stages a new transaction into the pending transaction pool.
func (bc *Blockchain) AddTransaction(tx ledger.Transaction) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.PendingTransactions = append(bc.PendingTransactions, tx)
	return nil
}

// GetPendingTransactions returns a thread-safe copy of the staged transactions.
func (bc *Blockchain) GetPendingTransactions() []ledger.Transaction {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	txsCopy := make([]ledger.Transaction, len(bc.PendingTransactions))
	copy(txsCopy, bc.PendingTransactions)
	return txsCopy
}

// MinePendingBlock packages pending transactions into a new block and mines it.
func (bc *Blockchain) MinePendingBlock() (*block.Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	tip := bc.Blocks[len(bc.Blocks)-1]
	txsCopy := make([]ledger.Transaction, len(bc.PendingTransactions))
	copy(txsCopy, bc.PendingTransactions)

	newBlock := &block.Block{
		Index:        tip.Index + 1,
		Timestamp:    time.Now().UnixNano(),
		Transactions: txsCopy,
		PrevHash:     tip.Hash,
		Nonce:        0,
	}

	newBlock.Mine(bc.Difficulty)
	bc.Blocks = append(bc.Blocks, newBlock)
	bc.PendingTransactions = make([]ledger.Transaction, 0)
	return newBlock, nil
}

// ValidateChain validates the current chain or a candidate chain from Genesis to tip.
func (bc *Blockchain) ValidateChain(chains ...[]*block.Block) (bool, error) {
	bc.mu.RLock()
	targetChain := bc.Blocks
	if len(chains) > 0 && chains[0] != nil {
		targetChain = chains[0]
	}
	diff := bc.Difficulty
	bc.mu.RUnlock()

	return validateChainList(targetChain, diff)
}

func validateChainList(targetChain []*block.Block, diff int) (bool, error) {
	if len(targetChain) == 0 {
		return false, errors.New("chain is empty")
	}

	if targetChain[0].PrevHash != "0" {
		return false, errors.New("invalid genesis previous hash")
	}

	targetPrefix := strings.Repeat("0", diff)

	for i := 1; i < len(targetChain); i++ {
		prev := targetChain[i-1]
		curr := targetChain[i]

		if curr.Index != prev.Index+1 {
			return false, fmt.Errorf("block %d has invalid index sequence", curr.Index)
		}
		if curr.PrevHash != prev.Hash {
			return false, fmt.Errorf("block %d prev hash mismatch", curr.Index)
		}
		if curr.CalculateHash() != curr.Hash {
			return false, fmt.Errorf("block %d hash mismatch", curr.Index)
		}
		if !strings.HasPrefix(curr.Hash, targetPrefix) {
			return false, fmt.Errorf("block %d does not satisfy difficulty %d", curr.Index, diff)
		}
	}
	return true, nil
}

// GetBalances calculates and returns all ledger account balances.
func (bc *Blockchain) GetBalances() map[string]int64 {
	return bc.RebuildLedgerBalances()
}

// RebuildLedgerBalances calculates full account balances based on the active chain transactions.
func (bc *Blockchain) RebuildLedgerBalances() map[string]int64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	balances := make(map[string]int64)
	for _, b := range bc.Blocks {
		for _, tx := range b.Transactions {
			if tx.Sender != "" {
				balances[tx.Sender] -= tx.Amount
			}
			if tx.Recipient != "" {
				balances[tx.Recipient] += tx.Amount
			}
		}
	}
	return balances
}

// SaveToFile serializes the blockchain state to a JSON file.
func (bc *Blockchain) SaveToFile(filename string) error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	data, err := json.MarshalIndent(bc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// LoadFromFile loads the blockchain state from a JSON file.
func (bc *Blockchain) LoadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	bc.mu.Lock()
	defer bc.mu.Unlock()

	var loaded Blockchain
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	bc.Blocks = loaded.Blocks
	bc.PendingTransactions = loaded.PendingTransactions
	bc.Difficulty = loaded.Difficulty
	return nil
}

// ResolveFork compares candidate chain against current chain.
func (bc *Blockchain) ResolveFork(candidateChain []*block.Block) (bool, []ledger.Transaction, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if len(candidateChain) <= len(bc.Blocks) {
		return false, nil, nil
	}

	valid, err := validateChainList(candidateChain, bc.Difficulty)
	if !valid || err != nil {
		return false, nil, fmt.Errorf("candidate chain invalid: %w", err)
	}

	candidateBlockHashes := make(map[string]bool)
	for _, b := range candidateChain {
		candidateBlockHashes[b.Hash] = true
	}

	candidateTxs := make(map[string]bool)
	for _, b := range candidateChain {
		for _, tx := range b.Transactions {
			key := tx.Signature
			if key == "" {
				key = fmt.Sprintf("%s:%s:%d", tx.Sender, tx.Recipient, tx.Amount)
			}
			candidateTxs[key] = true
		}
	}

	orphanedTxs := make([]ledger.Transaction, 0)
	for _, b := range bc.Blocks {
		if !candidateBlockHashes[b.Hash] {
			for _, tx := range b.Transactions {
				if tx.Sender != "" {
					key := tx.Signature
					if key == "" {
						key = fmt.Sprintf("%s:%s:%d", tx.Sender, tx.Recipient, tx.Amount)
					}
					if !candidateTxs[key] {
						orphanedTxs = append(orphanedTxs, tx)
					}
				}
			}
		}
	}

	bc.Blocks = candidateChain
	return true, orphanedTxs, nil
}