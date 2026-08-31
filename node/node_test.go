package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"toy-blockchain/block"
	"toy-blockchain/chain"
	"toy-blockchain/ledger"
)

func TestNodeHTTPService(t *testing.T) {
	bc := chain.NewBlockchain(1)
	nodeAddr := "localhost:8121"
	n := NewNode(nodeAddr, []string{"localhost:8122"}, bc)

	go func() { _ = n.Start() }()
	time.Sleep(100 * time.Millisecond)
	defer n.Stop()

	resp, err := http.Get("http://" + nodeAddr + "/status")
	if err != nil {
		t.Fatalf("Failed to query /status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}
}

func TestConcurrentGossipAndMining(t *testing.T) {
	bc1 := chain.NewBlockchain(1)
	bc2 := chain.NewBlockchain(1)

	addr1 := "localhost:8123"
	addr2 := "localhost:8124"

	node1 := NewNode(addr1, []string{addr2}, bc1)
	node2 := NewNode(addr2, []string{addr1}, bc2)

	go func() { _ = node1.Start() }()
	go func() { _ = node2.Start() }()
	time.Sleep(100 * time.Millisecond)
	defer node1.Stop()
	defer node2.Stop()

	sender, _ := ledger.NewWallet()
	recipient, _ := ledger.NewWallet()

	var wg sync.WaitGroup

	// Concurrently send transactions
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(val int64) {
			defer wg.Done()
			tx, _ := ledger.SignTransaction(sender, recipient.Address(), val+1)
			txBytes, _ := json.Marshal(tx)
			resp, err := http.Post("http://"+addr1+"/transaction", "application/json", bytes.NewBuffer(txBytes))
			if err == nil {
				resp.Body.Close()
			}
		}(int64(i))
	}

	// Concurrently read status & peer queries
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(fmt.Sprintf("http://%s/status", addr2))
			if err == nil {
				resp.Body.Close()
			}
		}()
	}

	// Concurrently mutate peer list
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			node1.AddPeer(fmt.Sprintf("localhost:999%d", idx))
			_ = node1.GetPeers()
		}(i)
	}

	wg.Wait()
}

func TestForkReorganizationAndMempoolResurrection(t *testing.T) {
	bc1 := chain.NewBlockchain(1)
	bc2 := chain.NewBlockchain(1)

	addr1 := "localhost:8125"
	addr2 := "localhost:8126"

	sender, _ := ledger.NewWallet()
	recipient, _ := ledger.NewWallet()

	rewardTx := ledger.NewTransaction("", sender.Address(), 1000)
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

	// Node 1 stages Tx1 and mines 1 block
	tx1, _ := ledger.SignTransaction(sender, recipient.Address(), 40)
	_ = bc1.AddTransaction(*tx1)
	b1, err := bc1.MinePendingBlock()
	if err != nil || len(b1.Transactions) == 0 {
		t.Fatalf("Failed to mine block on Node 1: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	// Node 2 mines 2 blocks (competing longer chain)
	_, _ = bc2.MinePendingBlock()
	time.Sleep(10 * time.Millisecond)
	_, _ = bc2.MinePendingBlock()

	node1 := NewNode(addr1, []string{addr2}, bc1)
	node2 := NewNode(addr2, []string{addr1}, bc2)

	go func() { _ = node1.Start() }()
	go func() { _ = node2.Start() }()
	time.Sleep(100 * time.Millisecond)
	defer node1.Stop()
	defer node2.Stop()

	err = node1.SyncWithPeers()
	if err != nil {
		t.Fatalf("SyncWithPeers failed: %v", err)
	}

	node1.mu.RLock()
	defer node1.mu.RUnlock()

	if node1.Blockchain.BlockCount() != 3 {
		t.Fatalf("Expected Node 1 to reorg to 3 blocks, got %d", node1.Blockchain.BlockCount())
	}

	if len(node1.PendingPool) != 1 || node1.PendingPool[0].Signature != tx1.Signature {
		t.Errorf("Expected Tx1 to be returned to Node 1 PendingPool after reorg, got pool length %d", len(node1.PendingPool))
	}
}
