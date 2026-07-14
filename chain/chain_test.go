package chain

import (
	"testing"
	"toy-blockchain/block"
	"toy-blockchain/ledger"
)

// TestPendingPoolDoubleSpend catches the pending queue double-spend weakness
func TestPendingPoolDoubleSpend(t *testing.T) {
	bc := NewBlockchain(2)

	// Mint 100 coins
	_ = bc.AddTransaction(ledger.NewTransaction("faucet", "Alice", 100))
	_, _ = bc.MinePendingBlock()

	// Attempt double spending within the pool queue
	err1 := bc.AddTransaction(ledger.NewTransaction("Alice", "Bob", 60))
	if err1 != nil {
		t.Fatalf("First legitimate spend was unexpectedly rejected: %v", err1)
	}

	// This must fail because 60 coins are already committed in pending pool
	err2 := bc.AddTransaction(ledger.NewTransaction("Alice", "Charlie", 60))
	if err2 == nil {
		t.Error("CRITICAL EXPLOIT: System accepted a double-spend transaction into the pending pool queue.")
	}
}

// TestGenesisTamperDetection verifies that mutating block 0 triggers explicit validation errors
func TestGenesisTamperDetection(t *testing.T) {
	bc := NewBlockchain(2)

	// Inject fraudulent balance transaction data explicitly into the genesis block index
	bc.Blocks[0].Transactions = append(bc.Blocks[0].Transactions, ledger.NewTransaction("system", "Eve", 1000000))

	valid, _, err := bc.ValidateChain()
	if valid || err == nil {
		t.Error("CRITICAL SECURITY EXPLOIT: Genesis block tampering was accepted silently without triggering errors.")
	}
}

// TestNegativeLedgerReplay confirms ledger state engine refuses negative drops
func TestNegativeLedgerReplay(t *testing.T) {
	bc := NewBlockchain(1)

	_ = bc.AddTransaction(ledger.NewTransaction("faucet", "Bob", 50))
	_, _ = bc.MinePendingBlock()

	// Bypass pool validation via structural backend injection to simulate disk manipulation
	maliciousBlock := block.NewBlock(2, []ledger.Transaction{
		ledger.NewTransaction("Bob", "Eve", 9999),
	}, bc.Blocks[1].Hash)
	maliciousBlock.Mine(1)

	bc.Blocks = append(bc.Blocks, maliciousBlock)

	valid, _, err := bc.ValidateChain()
	if valid || err == nil {
		t.Error("SECURITY BREAK: Ledger replay did not intercept negative asset balances during validation.")
	}
}

// TestTimestampValidation enforces chronological structural continuity rules
func TestTimestampValidation(t *testing.T) {
	bc := NewBlockchain(1)

	_ = bc.AddTransaction(ledger.NewTransaction("faucet", "Bob", 10))
	_, _ = bc.MinePendingBlock()

	// Inject a broken historical block where timestamp sequence steps backward
	badBlock := block.NewBlock(2, []ledger.Transaction{
		ledger.NewTransaction("Bob", "Alice", 5),
	}, bc.Blocks[1].Hash)
	badBlock.Timestamp = bc.Blocks[1].Timestamp - 500 // Backward jump
	badBlock.Mine(1)

	bc.Blocks = append(bc.Blocks, badBlock)

	valid, _, err := bc.ValidateChain()
	if valid || err == nil {
		t.Error("STRUCTURAL RULE GAP: Chain validation did not reject out-of-order block timestamps.")
	}
}
