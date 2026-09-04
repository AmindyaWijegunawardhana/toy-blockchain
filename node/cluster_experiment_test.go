package node

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"toy-blockchain/block"
	"toy-blockchain/chain"
	"toy-blockchain/ledger"
)

// Experiment 1: Gossip Propagation & Pool Consistency in a 3-Node Mesh
func TestExperiment1_GossipPropagation(t *testing.T) {
	bc1 := chain.NewBlockchain(1)
	bc2 := chain.NewBlockchain(1)
	bc3 := chain.NewBlockchain(1)

	addr1 := "localhost:8301"
	addr2 := "localhost:8302"
	addr3 := "localhost:8303"

	n1 := NewNode(addr1, []string{addr2, addr3}, bc1)
	n2 := NewNode(addr2, []string{addr1, addr3}, bc2)
	n3 := NewNode(addr3, []string{addr1, addr2}, bc3)

	go func() { _ = n1.Start() }()
	go func() { _ = n2.Start() }()
	go func() { _ = n3.Start() }()
	time.Sleep(150 * time.Millisecond)
	defer n1.Stop()
	defer n2.Stop()
	defer n3.Stop()

	sender, _ := ledger.NewWallet()
	recipient, _ := ledger.NewWallet()

	txCount := 10
	start := time.Now()

	for i := 0; i < txCount; i++ {
		tx, _ := ledger.SignTransaction(sender, recipient.Address(), int64(10+i))
		payload, _ := json.Marshal(tx)
		resp, err := http.Post("http://"+addr1+"/transaction", "application/json", bytes.NewBuffer(payload))
		if err != nil || resp.StatusCode != http.StatusCreated {
			t.Fatalf("Failed to post tx %d: %v", i, err)
		}
		resp.Body.Close()
	}

	time.Sleep(300 * time.Millisecond)
	elapsed := time.Since(start)

	p1 := n1.GetPendingPool()
	p2 := n2.GetPendingPool()
	p3 := n3.GetPendingPool()

	t.Logf("[Exp 1 Result] Broadcasted %d txs in %v", txCount, elapsed)
	t.Logf("[Exp 1 Result] Mempool counts -> N1: %d, N2: %d, N3: %d", len(p1), len(p2), len(p3))

	if len(p2) != txCount || len(p3) != txCount {
		t.Errorf("Gossip incomplete: expected %d in all mempools", txCount)
	}
}

// Experiment 2: Network Partition, Fork Resolution, and Mempool Restoration
func TestExperiment2_PartitionAndRecovery(t *testing.T) {
	bc1 := chain.NewBlockchain(1)
	bc2 := chain.NewBlockchain(1)

	addr1 := "localhost:8304"
	addr2 := "localhost:8305"

	sender, _ := ledger.NewWallet()
	recipient, _ := ledger.NewWallet()

	rewardTx := ledger.NewTransaction("", sender.Address(), 5000)
	sharedGenesis := &block.Block{
		Index:        0,
		Timestamp:    0,
		Transactions: []ledger.Transaction{rewardTx},
		PrevHash:     "0",
		Nonce:        0,
	}
	sharedGenesis.Mine(1)

	bc1.Blocks = []*block.Block{sharedGenesis}
	bc2.Blocks = []*block.Block{sharedGenesis}

	n1 := NewNode(addr1, []string{addr2}, bc1)
	n2 := NewNode(addr2, []string{addr1}, bc2)

	go func() { _ = n1.Start() }()
	go func() { _ = n2.Start() }()
	time.Sleep(150 * time.Millisecond)
	defer n1.Stop()
	defer n2.Stop()

	// Isolated Partition: Node 1 mines 1 block with Tx1
	tx1, _ := ledger.SignTransaction(sender, recipient.Address(), 150)
	_ = bc1.AddTransaction(*tx1)
	_, _ = bc1.MinePendingBlock()

	// Isolated Partition: Node 2 mines 3 blocks
	tx2, _ := ledger.SignTransaction(sender, recipient.Address(), 50)
	_ = bc2.AddTransaction(*tx2)
	_, _ = bc2.MinePendingBlock()
	_, _ = bc2.MinePendingBlock()
	_, _ = bc2.MinePendingBlock()

	t.Logf("[Exp 2 Pre-Sync] Node 1 height: %d | Node 2 height: %d", bc1.BlockCount(), bc2.BlockCount())

	// Reconnect and Sync
	reorgStart := time.Now()
	err := n1.SyncWithPeers()
	duration := time.Since(reorgStart)

	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	t.Logf("[Exp 2 Post-Sync] Node 1 adopted height: %d in %v", bc1.BlockCount(), duration)
	t.Logf("[Exp 2 Post-Sync] Node 1 resurrected mempool txs: %d", len(n1.GetPendingPool()))

	if bc1.BlockCount() != bc2.BlockCount() {
		t.Fatalf("Reorganization failed: height mismatch")
	}
}

// Experiment 3: PoW Mining Scaling
func TestExperiment3_DifficultyBenchmark(t *testing.T) {
	for diff := 1; diff <= 4; diff++ {
		b := &block.Block{
			Index:        1,
			Timestamp:    time.Now().UnixNano(),
			Transactions: []ledger.Transaction{},
			PrevHash:     "00000000000000000000000000000000",
			Nonce:        0,
		}

		start := time.Now()
		b.Mine(diff)
		elapsed := time.Since(start)

		t.Logf("[Exp 3 Benchmark] Difficulty: %d | Nonce: %d | Time: %v | Hash: %s",
			diff, b.Nonce, elapsed, b.Hash[:12])
	}
}
