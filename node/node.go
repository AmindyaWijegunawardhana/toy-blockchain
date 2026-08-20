package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"toy-blockchain/block"
	"toy-blockchain/chain"
	"toy-blockchain/ledger"
)

type Node struct {
	mu          sync.RWMutex
	Address     string
	Peers       map[string]bool
	Chain       *chain.Blockchain
	seenTxs     map[string]bool
	seenBlocks  map[string]bool
	Server      *http.Server
}

func NewNode(address string, initialPeers []string, difficulty int) *Node {
	peerMap := make(map[string]bool)
	for _, p := range initialPeers {
		if p != "" && p != address {
			peerMap[p] = true
		}
	}

	return &Node{
		Address:    address,
		Peers:      peerMap,
		Chain:      chain.NewBlockchain(difficulty),
		seenTxs:    make(map[string]bool),
		seenBlocks: make(map[string]bool),
	}
}

func (n *Node) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /chain", n.handleGetChain)
	mux.HandleFunc("GET /height", n.handleGetHeight)
	mux.HandleFunc("GET /peers", n.handleGetPeers)
	mux.HandleFunc("POST /peers", n.handleAddPeer)
	mux.HandleFunc("POST /tx", n.handleReceiveTx)
	mux.HandleFunc("POST /block", n.handleReceiveBlock)
	mux.HandleFunc("POST /mine", n.handleMine)
	mux.HandleFunc("GET /balances", n.handleGetBalances)
	mux.HandleFunc("POST /sync", n.handleSync)

	n.Server = &http.Server{
		Addr:    n.Address,
		Handler: mux,
	}

	log.Printf("[%s] Node HTTP Server started.", n.Address)
	return n.Server.ListenAndServe()
}

func (n *Node) handleReceiveTx(w http.ResponseWriter, r *http.Request) {
	var tx ledger.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	n.mu.Lock()
	txID := tx.Signature
	if txID == "" {
		txID = fmt.Sprintf("%s-%s-%d", tx.Sender, tx.Recipient, tx.Amount)
	}

	if n.seenTxs[txID] {
		n.mu.Unlock()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Transaction already seen"))
		return
	}
	n.seenTxs[txID] = true
	n.mu.Unlock()

	if err := n.Chain.AddTransaction(tx); err != nil {
		http.Error(w, fmt.Sprintf("Invalid TX: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("[%s] Accepted new TX: %s -> %s (%d)", n.Address, tx.Sender[:8], tx.Recipient[:8], tx.Amount)
	go n.broadcast("/tx", tx)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func (n *Node) handleReceiveBlock(w http.ResponseWriter, r *http.Request) {
	var newBlock block.Block
	if err := json.NewDecoder(r.Body).Decode(&newBlock); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	n.mu.Lock()
	if n.seenBlocks[newBlock.Hash] {
		n.mu.Unlock()
		w.WriteHeader(http.StatusOK)
		return
	}
	n.seenBlocks[newBlock.Hash] = true
	n.mu.Unlock()

	latestBlock := n.Chain.Blocks[len(n.Chain.Blocks)-1]
	if newBlock.PrevHash == latestBlock.Hash && newBlock.Index == latestBlock.Index+1 {
		n.Chain.Blocks = append(n.Chain.Blocks, &newBlock)
		valid, idx, err := n.Chain.ValidateChain()
		if !valid {
			n.Chain.Blocks = n.Chain.Blocks[:len(n.Chain.Blocks)-1]
			http.Error(w, fmt.Sprintf("Invalid block received: %v at %d", err, idx), http.StatusBadRequest)
			return
		}

		log.Printf("[%s] Accepted & Appended Block #%d [%s]", n.Address, newBlock.Index, newBlock.Hash[:8])
		go n.broadcast("/block", newBlock)
		w.WriteHeader(http.StatusAccepted)
		return
	}

	if newBlock.Index > latestBlock.Index {
		go n.SyncWithPeers()
	}

	w.WriteHeader(http.StatusOK)
}

func (n *Node) handleMine(w http.ResponseWriter, r *http.Request) {
	minedBlock, err := n.Chain.MinePendingBlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	n.mu.Lock()
	n.seenBlocks[minedBlock.Hash] = true
	n.mu.Unlock()

	log.Printf("[%s] Mined Block #%d [%s]", n.Address, minedBlock.Index, minedBlock.Hash[:8])
	go n.broadcast("/block", minedBlock)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(minedBlock)
}

func (n *Node) handleGetChain(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(n.Chain.Blocks)
}

func (n *Node) handleGetHeight(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]int{"height": len(n.Chain.Blocks)})
}

func (n *Node) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peers := make([]string, 0, len(n.Peers))
	for p := range n.Peers {
		peers = append(peers, p)
	}
	json.NewEncoder(w).Encode(peers)
}

func (n *Node) handleAddPeer(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	newPeer := body["peer"]
	if newPeer != "" && newPeer != n.Address {
		n.mu.Lock()
		n.Peers[newPeer] = true
		n.mu.Unlock()
	}

	w.WriteHeader(http.StatusOK)
}

func (n *Node) handleGetBalances(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(n.Chain.GetBalances())
}

func (n *Node) handleSync(w http.ResponseWriter, r *http.Request) {
	go n.SyncWithPeers()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Sync triggered"))
}

func (n *Node) SyncWithPeers() {
	n.mu.RLock()
	peers := make([]string, 0, len(n.Peers))
	for p := range n.Peers {
		peers = append(peers, p)
	}
	n.mu.RUnlock()

	for _, peer := range peers {
		resp, err := http.Get(fmt.Sprintf("http://%s/chain", peer))
		if err != nil {
			continue
		}

		var remoteBlocks []*block.Block
		if err := json.NewDecoder(resp.Body).Decode(&remoteBlocks); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		if len(remoteBlocks) > len(n.Chain.Blocks) {
			log.Printf("[%s] Found longer chain on peer %s (len %d vs local %d). Attempting reorg...",
				n.Address, peer, len(remoteBlocks), len(n.Chain.Blocks))

			replaced, err := n.Chain.ResolveFork(remoteBlocks)
			if replaced {
				log.Printf("[%s] Reorg successful! Adopted longer chain of length %d", n.Address, len(remoteBlocks))
				break
			} else {
				log.Printf("[%s] Fork resolution rejected chain: %v", n.Address, err)
			}
		}
	}
}

func (n *Node) broadcast(endpoint string, payload interface{}) {
	n.mu.RLock()
	peers := make([]string, 0, len(n.Peers))
	for p := range n.Peers {
		peers = append(peers, p)
	}
	n.mu.RUnlock()

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	for _, peer := range peers {
		url := fmt.Sprintf("http://%s%s", peer, endpoint)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}
}