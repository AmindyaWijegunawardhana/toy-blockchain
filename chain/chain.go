package chain

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"toy-blockchain/block"
	"toy-blockchain/ledger"
)

// Blockchain manages the append-only ledger state.
type Blockchain struct {
	mu          sync.RWMutex
	Blocks      []*block.Block
	PendingPool []ledger.Transaction
	Difficulty  int
}

// NewBlockchain initializes a new chain with a deterministic Genesis block.
func NewBlockchain(difficulty int) *Blockchain {
	bc := &Blockchain{
		Blocks:      make([]*block.Block, 0),
		PendingPool: make([]ledger.Transaction, 0),
		Difficulty:  difficulty,
	}
	bc.Blocks = append(bc.Blocks, block.NewGenesisBlock())
	return bc
}

// GetBalances computes current account balances by traversing the entire chain history.
func (bc *Blockchain) GetBalances() map[string]float64 {
	balances := make(map[string]float64)

	for _, b := range bc.Blocks {
		for _, tx := range b.Transactions {
			if tx.Sender != "faucet" && tx.Sender != "system" {
				balances[tx.Sender] -= tx.Amount
			}
			balances[tx.Recipient] += tx.Amount
		}
	}
	return balances
}

// AddTransaction validates and adds a transaction to the pending pool.
func (bc *Blockchain) AddTransaction(tx ledger.Transaction) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if tx.Amount <= 0 {
		return errors.New("transaction amount must be positive")
	}

	if tx.Sender != "faucet" && tx.Sender != "system" {
		balances := bc.GetBalances()
		if balances[tx.Sender] < tx.Amount {
			return fmt.Errorf("insufficient funds: %s has balance of %.2f, attempted to send %.2f",
				tx.Sender, balances[tx.Sender], tx.Amount)
		}
	}

	bc.PendingPool = append(bc.PendingPool, tx)
	return nil
}

// MinePendingBlock takes pending transactions, mines a new block, and updates the chain.
func (bc *Blockchain) MinePendingBlock() (*block.Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if len(bc.PendingPool) == 0 {
		return nil, errors.New("no pending transactions to mine")
	}

	latestBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := block.NewBlock(latestBlock.Index+1, bc.PendingPool, latestBlock.Hash)

	newBlock.Mine(bc.Difficulty)

	bc.Blocks = append(bc.Blocks, newBlock)
	bc.PendingPool = make([]ledger.Transaction, 0)

	return newBlock, nil
}

// ValidateChain checks the integrity of the entire block history.
func (bc *Blockchain) ValidateChain() (bool, int, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	targetPrefix := strings.Repeat("0", bc.Difficulty)

	for i := 1; i < len(bc.Blocks); i++ {
		current := bc.Blocks[i]
		previous := bc.Blocks[i-1]

		if current.Index != previous.Index+1 {
			return false, current.Index, fmt.Errorf("inconsistent block height at index %d", current.Index)
		}

		if current.PrevHash != previous.Hash {
			return false, current.Index, fmt.Errorf("broken hash link at index %d: expected %s, got %s",
				current.Index, previous.Hash, current.PrevHash)
		}

		recalculatedHash := current.CalculateHash()
		if current.Hash != recalculatedHash {
			return false, current.Index, fmt.Errorf("hash mismatch at index %d: block data was modified", current.Index)
		}

		if !strings.HasPrefix(current.Hash, targetPrefix) {
			return false, current.Index, fmt.Errorf("proof of work target unmet at index %d", current.Index)
		}
	}

	return true, 0, nil
}
