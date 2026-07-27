package chain

import (
	"testing"
	"toy-blockchain/block"
	"toy-blockchain/ledger"
)

// TestPendingPoolDoubleSpend catches the pending queue double-spend weakness with cryptographic keys
func TestPendingPoolDoubleSpend(t *testing.T) {
	bc := NewBlockchain(2)

	_, alicePub, err := ledger.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keys: %v", err)
	}

	_ = bc.AddTransaction(ledger.NewTransaction("faucet", alicePub, 100))
	_, _ = bc.MinePendingBlock()

	alicePriv, alicePubReal, _ := ledger.GenerateKeyPair()
	_ = bc.AddTransaction(ledger.NewTransaction("faucet", alicePubReal, 100))

	latestBlock := bc.Blocks[len(bc.Blocks)-1]
	b := block.NewBlock(latestBlock.Index+1, bc.PendingPool, latestBlock.Hash)
	b.Timestamp = latestBlock.Timestamp + 1
	b.Mine(bc.Difficulty)

	bc.mu.Lock()
	bc.Blocks = append(bc.Blocks, b)
	bc.PendingPool = make([]ledger.Transaction, 0)
	bc.mu.Unlock()

	tx1 := ledger.NewTransaction(alicePubReal, "Bob", 60)
	if err := tx1.Sign(alicePriv); err != nil {
		t.Fatalf("Signing failed: %v", err)
	}

	err1 := bc.AddTransaction(tx1)
	if err1 != nil {
		t.Fatalf("First legitimate spend was unexpectedly rejected: %v", err1)
	}

	tx2 := ledger.NewTransaction(alicePubReal, "Charlie", 60)
	if err := tx2.Sign(alicePriv); err != nil {
		t.Fatalf("Signing failed: %v", err)
	}

	err2 := bc.AddTransaction(tx2)
	if err2 == nil {
		t.Error("CRITICAL EXPLOIT: System accepted a double-spend transaction into the pending pool queue.")
	}
}

// TestGenesisTamperDetection verifies that mutating block 0 triggers explicit validation errors
func TestGenesisTamperDetection(t *testing.T) {
	bc := NewBlockchain(2)

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

	maliciousBlock := block.NewBlock(2, []ledger.Transaction{
		ledger.NewTransaction("Bob", "Eve", 9999),
	}, bc.Blocks[len(bc.Blocks)-1].Hash)
	maliciousBlock.Timestamp = bc.Blocks[len(bc.Blocks)-1].Timestamp + 1
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

	badBlock := block.NewBlock(2, []ledger.Transaction{
		ledger.NewTransaction("Bob", "Alice", 5),
	}, bc.Blocks[len(bc.Blocks)-1].Hash)
	badBlock.Timestamp = bc.Blocks[0].Timestamp - 500
	badBlock.Mine(1)

	bc.Blocks = append(bc.Blocks, badBlock)

	valid, _, err := bc.ValidateChain()
	if valid || err == nil {
		t.Error("STRUCTURAL RULE GAP: Chain validation did not reject out-of-order block timestamps.")
	}
}

// TestSignatureVerificationAndIdentityForgery validates cryptographic authentication fields
func TestSignatureVerificationAndIdentityForgery(t *testing.T) {
	bc := NewBlockchain(1)

	alicePriv, alicePub, err := ledger.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keys: %v", err)
	}

	tx1 := ledger.NewTransaction("faucet", alicePub, 500)
	err = bc.AddTransaction(tx1)
	if err != nil {
		t.Fatalf("Failed to add funding tx: %v", err)
	}

	latestBlock := bc.Blocks[len(bc.Blocks)-1]
	b1 := block.NewBlock(latestBlock.Index+1, bc.PendingPool, latestBlock.Hash)
	b1.Timestamp = bc.Blocks[0].Timestamp + 1
	b1.Mine(bc.Difficulty)

	bc.mu.Lock()
	bc.Blocks = append(bc.Blocks, b1)
	bc.PendingPool = make([]ledger.Transaction, 0)
	bc.mu.Unlock()

	_, bobPub, _ := ledger.GenerateKeyPair()
	tx2 := ledger.NewTransaction(alicePub, bobPub, 200)

	err = tx2.Sign(alicePriv)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	err = bc.AddTransaction(tx2)
	if err != nil {
		t.Fatalf("Legitimate signed transaction was rejected at gateway: %v", err)
	}

	latestBlock = bc.Blocks[len(bc.Blocks)-1]
	b2 := block.NewBlock(latestBlock.Index+1, bc.PendingPool, latestBlock.Hash)
	b2.Timestamp = b1.Timestamp + 1
	b2.Mine(bc.Difficulty)

	bc.mu.Lock()
	bc.Blocks = append(bc.Blocks, b2)
	bc.PendingPool = make([]ledger.Transaction, 0)
	bc.mu.Unlock()

	valid, brokenIdx, valErr := bc.ValidateChain()
	if !valid {
		t.Fatalf("Blockchain rejected valid cryptographic signatures at block index %d: %v", brokenIdx, valErr)
	}

	maliciousTx := ledger.NewTransaction(alicePub, bobPub, 50)
	attackerPriv, _, _ := ledger.GenerateKeyPair()

	_ = maliciousTx.Sign(attackerPriv)

	latestBlock = bc.Blocks[len(bc.Blocks)-1]
	forgedBlock := block.NewBlock(latestBlock.Index+1, []ledger.Transaction{maliciousTx}, latestBlock.Hash)
	forgedBlock.Timestamp = latestBlock.Timestamp + 1
	forgedBlock.Mine(bc.Difficulty)

	bc.Blocks = append(bc.Blocks, forgedBlock)

	validPostAttack, _, _ := bc.ValidateChain()
	if validPostAttack {
		t.Error("CRITICAL HIGH EXPLOIT: Validation engine accepted an illicit signature forgery.")
	}
}

// TestDifficultyRetargeting confirms that mining complexity shifts dynamically to match block production speed
// TestDifficultyRetargeting confirms that mining complexity shifts dynamically to match block production speed
// TestDifficultyRetargeting confirms that mining complexity shifts dynamically to match block production speed
// TestDifficultyRetargeting confirms that mining complexity shifts dynamically to match block production speed
// TestDifficultyRetargeting confirms that mining complexity shifts dynamically to match block production speed
func TestDifficultyRetargeting(t *testing.T) {
	bc := NewBlockchain(1) // Initial chain difficulty baseline is 1

	_, alicePub, _ := ledger.GenerateKeyPair()

	// Mine blocks 1 through 4 manually with fast 1-second intervals
	for i := 1; i <= 4; i++ {
		_ = bc.AddTransaction(ledger.NewTransaction("faucet", alicePub, int64(i*10)))

		latestBlock := bc.Blocks[len(bc.Blocks)-1]
		b := block.NewBlock(latestBlock.Index+1, bc.PendingPool, latestBlock.Hash)
		b.Timestamp = latestBlock.Timestamp + 1 // Fast 1-second interval

		targetDiff := bc.CalculateNextDifficulty(b.Index)
		b.Mine(targetDiff)

		bc.mu.Lock()
		bc.Blocks = append(bc.Blocks, b)
		bc.PendingPool = make([]ledger.Transaction, 0)
		bc.mu.Unlock()
	}

	// At block index 4, retargeting calculates that the NEXT block (5) should be difficulty 2
	nextDiff := bc.CalculateNextDifficulty(4)
	if nextDiff <= 1 {
		t.Errorf("RETARGETING FAILURE: Expected difficulty to scale up due to hyper-fast block generation velocity, but stayed at %d", nextDiff)
	}

	// Keep bc.Difficulty aligned with starting state for replay verification
	bc.Difficulty = 1

	valid, brokenIdx, err := bc.ValidateChain()
	if !valid {
		t.Fatalf("Retargeting verification error at block index %d: %v", brokenIdx, err)
	}
}

// TestForkResolution verifies that the node adopts the longest valid competing chain
func TestForkResolution(t *testing.T) {
	localBC := NewBlockchain(1)
	_, alicePub, _ := ledger.GenerateKeyPair()

	_ = localBC.AddTransaction(ledger.NewTransaction("faucet", alicePub, 100))
	b1Local, err := localBC.MinePendingBlock()
	if err != nil || b1Local == nil {
		t.Fatalf("Failed to mine initial local block: %v", err)
	}

	competingBC := NewBlockchain(1)
	_, bobPub, _ := ledger.GenerateKeyPair()

	_ = competingBC.AddTransaction(ledger.NewTransaction("faucet", bobPub, 200))
	b1Comp, err := competingBC.MinePendingBlock()
	if err != nil || b1Comp == nil {
		t.Fatalf("Failed to mine competing block 1: %v", err)
	}

	_ = competingBC.AddTransaction(ledger.NewTransaction("faucet", "Charlie", 50))
	latestBlock := competingBC.Blocks[len(competingBC.Blocks)-1]
	b2Comp := block.NewBlock(latestBlock.Index+1, competingBC.PendingPool, latestBlock.Hash)
	b2Comp.Timestamp = latestBlock.Timestamp + 1
	b2Comp.Mine(competingBC.Difficulty)

	competingBC.mu.Lock()
	competingBC.Blocks = append(competingBC.Blocks, b2Comp)
	competingBC.PendingPool = make([]ledger.Transaction, 0)
	competingBC.mu.Unlock()

	replaced, err := localBC.ResolveFork(competingBC.Blocks)
	if !replaced || err != nil {
		t.Fatalf("FAILED: Local chain failed to adopt valid longer competing chain: %v", err)
	}

	if len(localBC.Blocks) != 3 {
		t.Errorf("Expected local chain length to be 3 after reorg, got %d", len(localBC.Blocks))
	}

	invalidBlocks := make([]*block.Block, len(competingBC.Blocks))
	copy(invalidBlocks, competingBC.Blocks)

	lastBlock := invalidBlocks[len(invalidBlocks)-1]
	// Attempt an unauthorized transfer from Alice to Eve without sufficient balance or valid signatures
	forgedTx := ledger.NewTransaction(alicePub, "Eve", 999999)
	badBlock := block.NewBlock(lastBlock.Index+1, []ledger.Transaction{forgedTx}, lastBlock.Hash)
	badBlock.Timestamp = lastBlock.Timestamp + 1
	badBlock.Mine(1)

	invalidBlocks = append(invalidBlocks, badBlock)

	replacedPostAttack, _ := localBC.ResolveFork(invalidBlocks)
	if replacedPostAttack {
		t.Error("CRITICAL SECURITY FAILURE: Local chain accepted an invalid longer chain fork!")
	}
}
