package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"toy-blockchain/block"
	"toy-blockchain/chain"
	"toy-blockchain/ledger"
)

// Node represents a networked blockchain node process with full concurrent state protection.
type Node struct {
	Addr       string
	Peers      []string
	Blockchain *chain.Blockchain

	PendingPool []*ledger.Transaction
	seenTxs     map[string]bool
	mu          sync.RWMutex
	client      *http.Client
	server      *http.Server
}

// NewNode initializes a new Node process with its listen address, peer list, and chain instance.
func NewNode(addr string, peers []string, bc *chain.Blockchain) *Node {
	peersCopy := make([]string, len(peers))
	copy(peersCopy, peers)

	return &Node{
		Addr:        addr,
		Peers:       peersCopy,
		Blockchain:  bc,
		PendingPool: make([]*ledger.Transaction, 0),
		seenTxs:     make(map[string]bool),
		client:      &http.Client{Timeout: 2 * time.Second},
	}
}

// AddPeer safely registers a new peer in the active peer set.
func (n *Node) AddPeer(peer string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	for _, p := range n.Peers {
		if p == peer {
			return
		}
	}
	n.Peers = append(n.Peers, peer)
}

// GetPeers safely returns a copy of the peer set.
func (n *Node) GetPeers() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peersCopy := make([]string, len(n.Peers))
	copy(peersCopy, n.Peers)
	return peersCopy
}

// GetPendingPool returns a thread-safe snapshot of the pending transactions in the mempool.
func (n *Node) GetPendingPool() []*ledger.Transaction {
	n.mu.RLock()
	defer n.mu.RUnlock()

	poolCopy := make([]*ledger.Transaction, len(n.PendingPool))
	copy(poolCopy, n.PendingPool)
	return poolCopy
}

// Start launches the HTTP server for this node.
func (n *Node) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/chain", n.handleGetChain)
	mux.HandleFunc("/status", n.handleGetStatus)
	mux.HandleFunc("/transaction", n.handleTransaction)
	mux.HandleFunc("/block", n.handleBlock)
	mux.HandleFunc("/blocks", n.handleGetBlocks)

	n.server = &http.Server{
		Addr:    n.Addr,
		Handler: mux,
	}

	log.Printf("[%s] Node HTTP Server starting...", n.Addr)
	return n.server.ListenAndServe()
}

// Stop gracefully shuts down the node HTTP server.
func (n *Node) Stop() error {
	if n.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return n.server.Shutdown(ctx)
	}
	return nil
}

func (n *Node) handleGetChain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	blocks := n.Blockchain.GetBlocks()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"blocks":     blocks,
		"difficulty": n.Blockchain.Difficulty,
	})
}

func (n *Node) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	n.mu.RLock()
	peersCopy := make([]string, len(n.Peers))
	copy(peersCopy, n.Peers)
	pendingCount := len(n.PendingPool)
	n.mu.RUnlock()

	status := map[string]interface{}{
		"address":     n.Addr,
		"peers":       peersCopy,
		"block_count": n.Blockchain.BlockCount(),
		"pending_txs": pendingCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (n *Node) handleGetBlocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fromStr := r.URL.Query().Get("from")
	fromIdx := 0
	if fromStr != "" {
		parsed, err := strconv.Atoi(fromStr)
		if err == nil && parsed >= 0 {
			fromIdx = parsed
		}
	}

	allBlocks := n.Blockchain.GetBlocks()
	if fromIdx >= len(allBlocks) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]*block.Block{})
		return
	}

	requestedBlocks := allBlocks[fromIdx:]
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(requestedBlocks)
}

func (n *Node) handleTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var tx ledger.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		http.Error(w, "Invalid transaction payload", http.StatusBadRequest)
		return
	}

	if err := ledger.VerifyTransaction(&tx); err != nil {
		http.Error(w, fmt.Sprintf("Transaction verification failed: %v", err), http.StatusBadRequest)
		return
	}

	txID := tx.Signature
	if txID == "" {
		txID = tx.Hash()
	}

	n.mu.Lock()
	if n.seenTxs[txID] {
		n.mu.Unlock()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ignored", "reason": "duplicate transaction"})
		return
	}

	n.seenTxs[txID] = true
	n.PendingPool = append(n.PendingPool, &tx)
	n.mu.Unlock()

	go n.gossipTransaction(&tx)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func (n *Node) handleBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var newBlock block.Block
	if err := json.NewDecoder(r.Body).Decode(&newBlock); err != nil {
		http.Error(w, "Invalid block payload", http.StatusBadRequest)
		return
	}

	tip := n.Blockchain.LatestBlock()

	if newBlock.Index != tip.Index+1 || newBlock.PrevHash != tip.Hash {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"status": "deferred_for_sync"})
		return
	}

	n.Blockchain.AddBlock(&newBlock)

	n.mu.Lock()
	n.removePendingTxs(newBlock.Transactions)
	n.mu.Unlock()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "appended"})
}

// SyncWithPeers queries peers for their chain state and handles fork resolution / reorganisation.
func (n *Node) SyncWithPeers() error {
	peersCopy := n.GetPeers()

	for _, peer := range peersCopy {
		chainURL := fmt.Sprintf("http://%s/chain", peer)
		resp, err := n.client.Get(chainURL)
		if err != nil {
			continue
		}

		var payload struct {
			Blocks     []*block.Block `json:"blocks"`
			Difficulty int            `json:"difficulty"`
		}
		err = json.NewDecoder(resp.Body).Decode(&payload)
		resp.Body.Close()
		if err != nil {
			continue
		}

		if len(payload.Blocks) > n.Blockchain.BlockCount() {
			reorged, orphanedTxs, err := n.Blockchain.ResolveFork(payload.Blocks)
			if err == nil && reorged {
				n.mu.Lock()
				for i := range orphanedTxs {
					tx := orphanedTxs[i]
					n.PendingPool = append(n.PendingPool, &tx)
				}
				n.mu.Unlock()
			}
		}
	}
	return nil
}

// BroadcastBlock sends a newly mined block to all connected peers.
func (n *Node) BroadcastBlock(b block.Block) {
	data, err := json.Marshal(b)
	if err != nil {
		return
	}

	peersCopy := n.GetPeers()

	for _, peer := range peersCopy {
		peerURL := fmt.Sprintf("http://%s/block", peer)
		resp, err := n.client.Post(peerURL, "application/json", bytes.NewBuffer(data))
		if err == nil {
			resp.Body.Close()
		}
	}
}

func (n *Node) gossipTransaction(tx *ledger.Transaction) {
	data, err := json.Marshal(tx)
	if err != nil {
		return
	}

	peersCopy := n.GetPeers()

	for _, peer := range peersCopy {
		peerURL := fmt.Sprintf("http://%s/transaction", peer)
		resp, err := n.client.Post(peerURL, "application/json", bytes.NewBuffer(data))
		if err == nil {
			resp.Body.Close()
		}
	}
}

func (n *Node) removePendingTxs(minedTxs []ledger.Transaction) {
	minedMap := make(map[string]bool)
	for _, tx := range minedTxs {
		id := tx.Signature
		if id == "" {
			id = tx.Hash()
		}
		minedMap[id] = true
	}

	newPool := make([]*ledger.Transaction, 0)
	for _, tx := range n.PendingPool {
		id := tx.Signature
		if id == "" {
			id = tx.Hash()
		}
		if !minedMap[id] {
			newPool = append(newPool, tx)
		}
	}
	n.PendingPool = newPool
}
