package main

import (
	"fmt"
	"toy-blockchain/block"
	"toy-blockchain/ledger"
)

func main() {
	fmt.Println("=== Initializing Toy Blockchain Simulator ===")

	// 1. Generate the deterministic Genesis Block (FR-2)
	genesis := block.NewGenesisBlock()
	fmt.Printf("Genesis Block Created!\n")
	fmt.Printf("  Index:     %d\n", genesis.Index)
	fmt.Printf("  Timestamp: %d\n", genesis.Timestamp)
	fmt.Printf("  Prev Hash: %s\n", genesis.PrevHash)
	fmt.Printf("  Block Hash:%s\n\n", genesis.Hash)

	// 2. Simulate creating a new block with a transaction batch (FR-1, FR-4)
	fmt.Println("=== Creating Block 1 ===")
	txs := []ledger.Transaction{
		ledger.NewTransaction("faucet", "Amindya", 100.0),
		ledger.NewTransaction("Amindya", "Bob", 30.0),
	}

	block1 := block.NewBlock(1, txs, genesis.Hash)
	fmt.Printf("Block 1 Struct Initialized!\n")
	fmt.Printf("  Index:     %d\n", block1.Index)
	fmt.Printf("  Prev Hash: %s\n", block1.PrevHash)
	fmt.Printf("  Block Hash:%s\n", block1.Hash)
}
