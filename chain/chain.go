package chain

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"toy-blockchain/block"
	"toy-blockchain/ledger"
)

const (
	BlockIntervalWindow = 4 // Recalculate difficulty every N blocks
	TargetBlockTime     = 5 // Target time in seconds per block
	TargetDuration      = BlockIntervalWindow * TargetBlockTime
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
func (bc *Blockchain) GetBalances() map[string]int64 {
	balances := make(map[string]int64)

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

	// 1. Verify cryptographic authorization signature
	if !tx.Verify() {
		return errors.New("invalid transaction signature: unauthorized transfer execution")
	}

	// 2. Validate basic amount constraint
	if tx.Amount <= 0 {
		return errors.New("transaction amount must be positive")
	}

	// 3. Prevent double-spending against the unconfirmed pool queue
	if tx.Sender != "faucet" && tx.Sender != "system" {
		balances := bc.GetBalances()

		availableBalance := balances[tx.Sender]
		for _, pendingTx := range bc.PendingPool {
			if pendingTx.Sender == tx.Sender {
				availableBalance -= pendingTx.Amount
			}
		}

		if availableBalance < tx.Amount {
			return fmt.Errorf("insufficient funds (including pending pool): %s has %d, trying to spend %d",
				tx.Sender, availableBalance, tx.Amount)
		}
	}

	bc.PendingPool = append(bc.PendingPool, tx)
	return nil
}

// MinePendingBlock takes pending transactions, calculates next target difficulty, and mines a new block.
func (bc *Blockchain) MinePendingBlock() (*block.Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if len(bc.PendingPool) == 0 {
		return nil, errors.New("no pending transactions to mine")
	}

	latestBlock := bc.Blocks[len(bc.Blocks)-1]

	// Dynamic difficulty retargeting evaluation
	nextDifficulty := bc.CalculateNextDifficulty(latestBlock.Index + 1)
	bc.Difficulty = nextDifficulty

	newBlock := block.NewBlock(latestBlock.Index+1, bc.PendingPool, latestBlock.Hash)
	newBlock.Mine(bc.Difficulty)

	bc.Blocks = append(bc.Blocks, newBlock)
	bc.PendingPool = make([]ledger.Transaction, 0)

	return newBlock, nil
}

// CalculateNextDifficulty returns the algorithmic dynamic difficulty factor for a given block height.
func (bc *Blockchain) CalculateNextDifficulty(nextIndex int) int {
	if nextIndex == 0 || nextIndex%BlockIntervalWindow != 0 {
		return bc.Difficulty
	}

	lastAdjustmentBlock := bc.Blocks[nextIndex-BlockIntervalWindow]
	latestBlock := bc.Blocks[nextIndex-1]

	actualDuration := latestBlock.Timestamp - lastAdjustmentBlock.Timestamp
	currentDiff := bc.Difficulty

	if actualDuration < int64(TargetDuration/2) {
		return currentDiff + 1
	} else if actualDuration > int64(TargetDuration*2) {
		if currentDiff > 1 {
			return currentDiff - 1
		}
	}
	return currentDiff
}

// ResolveFork replaces local chain with a competing chain if the competing chain is valid and strictly longer.
func (bc *Blockchain) ResolveFork(competingBlocks []*block.Block) (bool, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// 1. Must be strictly longer to trigger reorg
	if len(competingBlocks) <= len(bc.Blocks) {
		return false, errors.New("competing chain is not longer than local chain")
	}

	// 2. Perform full structural validation on competing chain snapshot
	tempChain := &Blockchain{
		Blocks:     competingBlocks,
		Difficulty: bc.Difficulty,
	}

	valid, brokenIdx, err := tempChain.ValidateChain()
	if !valid {
		return false, fmt.Errorf("competing chain failed validation at index %d: %v", brokenIdx, err)
	}

	// 3. Reorganization: Adopt longer chain & update pending pool
	bc.Blocks = competingBlocks

	confirmedTxs := make(map[string]bool)
	for _, b := range bc.Blocks {
		for _, tx := range b.Transactions {
			if tx.Signature != "" {
				confirmedTxs[tx.Signature] = true
			}
		}
	}

	newPending := make([]ledger.Transaction, 0)
	for _, tx := range bc.PendingPool {
		if !confirmedTxs[tx.Signature] {
			newPending = append(newPending, tx)
		}
	}
	bc.PendingPool = newPending

	return true, nil
}

// ValidateChain checks the integrity of the entire block history including dynamic difficulty & Merkle rules.
func (bc *Blockchain) ValidateChain() (bool, int, error) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	if len(bc.Blocks) == 0 {
		return false, 0, errors.New("blockchain is completely empty")
	}

	// 1. Validate Genesis Block Merkle integrity explicitly
	genesisMerkle := bc.Blocks[0].CalculateMerkleRoot()
	if bc.Blocks[0].MerkleRoot != genesisMerkle {
		return false, 0, errors.New("genesis block transaction data tampered")
	}

	historicalBalances := make(map[string]int64)
	expectedDifficulty := bc.Difficulty

	// Process transactions for Genesis Block
	for _, tx := range bc.Blocks[0].Transactions {
		if !tx.Verify() {
			return false, 0, fmt.Errorf("genesis block contains invalid cryptographic signatures")
		}
		if tx.Amount <= 0 {
			return false, 0, fmt.Errorf("genesis block contains non-positive transaction amount: %d", tx.Amount)
		}
		if tx.Sender != "faucet" && tx.Sender != "system" {
			historicalBalances[tx.Sender] -= tx.Amount
		}
		historicalBalances[tx.Recipient] += tx.Amount
	}

	// 2. Continuous history verification loop
	for i := 1; i < len(bc.Blocks); i++ {
		current := bc.Blocks[i]
		previous := bc.Blocks[i-1]

		if current.Index != previous.Index+1 {
			return false, current.Index, fmt.Errorf("inconsistent block height at index %d", current.Index)
		}

		if current.Timestamp <= previous.Timestamp {
			return false, current.Index, fmt.Errorf("timestamp sequence error at index %d: %d is not after %d", current.Index, current.Timestamp, previous.Timestamp)
		}

		if current.PrevHash != previous.Hash {
			return false, current.Index, fmt.Errorf("broken hash link at index %d: expected %s, got %s",
				current.Index, previous.Hash, current.PrevHash)
		}

		// Retargeting tracker rule validation
		if current.Index%BlockIntervalWindow == 0 {
			lastAdjustmentBlock := bc.Blocks[current.Index-BlockIntervalWindow]
			actualDuration := previous.Timestamp - lastAdjustmentBlock.Timestamp

			if actualDuration < int64(TargetDuration/2) {
				expectedDifficulty++
			} else if actualDuration > int64(TargetDuration*2) {
				if expectedDifficulty > 1 {
					expectedDifficulty--
				}
			}
		}

		// Verify block hash matches recorded header configuration
		recalculatedHash := current.CalculateHash()
		if current.Hash != recalculatedHash {
			return false, current.Index, fmt.Errorf("hash mismatch at index %d: block header data was modified", current.Index)
		}

		// Verify Proof-of-Work prefix matches expected difficulty
		targetPrefix := strings.Repeat("0", expectedDifficulty)
		if !strings.HasPrefix(current.Hash, targetPrefix) {
			return false, current.Index, fmt.Errorf("proof of work target unmet at index %d: expected diff %d", current.Index, expectedDifficulty)
		}

		// Verify Merkle Root summary integrity
		actualMerkleRoot := current.CalculateMerkleRoot()
		if current.MerkleRoot != actualMerkleRoot {
			return false, current.Index, fmt.Errorf("merkle root mismatch at index %d: transaction data tampered", current.Index)
		}

		// Verify transactions and history replays
		for _, tx := range current.Transactions {
			if !tx.Verify() {
				return false, current.Index, fmt.Errorf("cryptographic signature mismatch at index %d: unauthorized transfer", current.Index)
			}
			if tx.Amount <= 0 {
				return false, current.Index, fmt.Errorf("block %d contains non-positive transaction amount: %d", current.Index, tx.Amount)
			}
			if tx.Sender != "faucet" && tx.Sender != "system" {
				historicalBalances[tx.Sender] -= tx.Amount
				if historicalBalances[tx.Sender] < 0 {
					return false, current.Index, fmt.Errorf("block %d replay caused illegal negative balance for %s", current.Index, tx.Sender)
				}
			}
			historicalBalances[tx.Recipient] += tx.Amount
		}
	}

	return true, 0, nil
}

// SaveToFile serializes the blockchain and writes it out to disk.
func (bc *Blockchain) SaveToFile(filename string) error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	data, err := json.MarshalIndent(bc.Blocks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal blockchain: %v", err)
	}

	return ioutil.WriteFile(filename, data, 0644)
}

// LoadFromFile updates the blockchain by deserializing state from a disk file.
func (bc *Blockchain) LoadFromFile(filename string) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", filename)
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read blockchain file: %v", err)
	}

	var importedBlocks []*block.Block
	if err := json.Unmarshal(data, &importedBlocks); err != nil {
		return fmt.Errorf("failed to unmarshal blockchain file data: %v", err)
	}

	bc.Blocks = importedBlocks
	return nil
}
