package chain

import (
	"testing"
	"time"

	"toy-blockchain/block"
	"toy-blockchain/ledger"
)

func TestBlockchainInitialization(t *testing.T) {
	bc := NewBlockchain(1)
	if bc.BlockCount() != 1 {
		t.Fatalf("Expected initial chain length 1, got %d", bc.BlockCount())
	}
	if bc.Blocks[0].Index != 0 {
		t.Errorf("Expected genesis index 0, got %d", bc.Blocks[0].Index)
	}
}

func TestForkResolutionAndReorganization(t *testing.T) {
	bcMain := NewBlockchain(1)
	bcFork := NewBlockchain(1)

	sender, err := ledger.NewWallet()
	if err != nil {
		t.Fatalf("Failed to create sender wallet: %v", err)
	}
	recipient, err := ledger.NewWallet()
	if err != nil {
		t.Fatalf("Failed to create recipient wallet: %v", err)
	}

	// Create shared genesis block with initial credit
	rewardTx := ledger.NewTransaction("", sender.Address(), 500)
	sharedGenesis := &block.Block{
		Index:        0,
		Timestamp:    0,
		Transactions: []ledger.Transaction{rewardTx},
		PrevHash:     "0",
		Nonce:        0,
	}
	sharedGenesis.Mine(1)

	bcMain.Blocks = []*block.Block{sharedGenesis}
	bcFork.Blocks = []*block.Block{sharedGenesis}

	// Main chain stages and mines Tx A
	txA, err := ledger.SignTransaction(sender, recipient.Address(), 100)
	if err != nil {
		t.Fatalf("Failed to sign txA: %v", err)
	}
	_ = bcMain.AddTransaction(*txA)
	bMain1, err := bcMain.MinePendingBlock()
	if err != nil {
		t.Fatalf("Failed to mine on main: %v", err)
	}
	if len(bMain1.Transactions) == 0 {
		t.Fatalf("bMain1 has no transactions")
	}

	time.Sleep(10 * time.Millisecond)

	// Fork chain stages and mines Tx B, then mines a second block to make it strictly longer
	txB, err := ledger.SignTransaction(sender, recipient.Address(), 50)
	if err != nil {
		t.Fatalf("Failed to sign txB: %v", err)
	}
	_ = bcFork.AddTransaction(*txB)
	_, err = bcFork.MinePendingBlock()
	if err != nil {
		t.Fatalf("Failed to mine fork block 1: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	_, err = bcFork.MinePendingBlock()
	if err != nil {
		t.Fatalf("Failed to mine fork block 2: %v", err)
	}

	reorged, orphanedTxs, err := bcMain.ResolveFork(bcFork.Blocks)
	if err != nil {
		t.Fatalf("Fork resolution failed: %v", err)
	}

	if !reorged {
		t.Fatalf("Expected chain to reorganize to longer fork")
	}

	if bcMain.BlockCount() != 3 {
		t.Errorf("Expected chain length to be 3 after reorg, got %d", bcMain.BlockCount())
	}

	if len(orphanedTxs) != 1 || orphanedTxs[0].Signature != txA.Signature {
		t.Errorf("Expected Tx A to be resurrected as orphaned transaction, got %d txs", len(orphanedTxs))
	}

	balances := bcMain.RebuildLedgerBalances()
	if balances[recipient.Address()] != 50 {
		t.Errorf("Expected recipient balance 50, got %d", balances[recipient.Address()])
	}
}
