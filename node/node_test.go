package node

import (
	"encoding/hex"
	"testing"
	"time"
	"toy-blockchain/ledger"
)

func TestNodeNetworkGossipAndSync(t *testing.T) {
	// Initialize Node A and Node B
	nodeA := NewNode("localhost:8011", []string{"localhost:8012"}, 1)
	nodeB := NewNode("localhost:8012", []string{"localhost:8011"}, 1)

	go func() {
		_ = nodeA.Start()
	}()
	go func() {
		_ = nodeB.Start()
	}()

	// Allow servers to boot
	time.Sleep(100 * time.Millisecond)

	// Generate authenticated wallet
	priv, pub, err := ledger.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}
	pubHex := hex.EncodeToString(pub)

	// Add funding tx to Node A
	fundingTx := ledger.NewTransaction("faucet", pubHex, 500)
	err = nodeA.Chain.AddTransaction(fundingTx)
	if err != nil {
		t.Fatalf("Failed to add transaction to Node A: %v", err)
	}

	// Mine block on Node A
	b1, err := nodeA.Chain.MinePendingBlock()
	if err != nil {
		t.Fatalf("Node A failed to mine block: %v", err)
	}

	// Submit signed spend from Node A
	_, recipientPub, _ := ledger.GenerateKeyPair()
	recipientHex := hex.EncodeToString(recipientPub)

	spendTx := ledger.NewTransaction(pubHex, recipientHex, 150)
	if err := spendTx.Sign(priv); err != nil {
		t.Fatalf("Failed to sign spend transaction: %v", err)
	}

	if err := nodeA.Chain.AddTransaction(spendTx); err != nil {
		t.Fatalf("Node A rejected valid signed spend: %v", err)
	}

	// Trigger Node B sync from Node A
	nodeB.SyncWithPeers()

	if len(nodeB.Chain.Blocks) != len(nodeA.Chain.Blocks) {
		t.Errorf("Node B failed to sync chain history from Node A. Expected height %d, got %d",
			len(nodeA.Chain.Blocks), len(nodeB.Chain.Blocks))
	}

	if nodeB.Chain.Blocks[1].Hash != b1.Hash {
		t.Error("Block hash mismatch between Node A and Node B after network sync.")
	}
}
