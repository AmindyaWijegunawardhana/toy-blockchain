package main

import (
	"fmt"
	"toy-blockchain/chain"
	"toy-blockchain/ledger"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("        DAY 2: MINING & DIFFICULTY TUNING         ")
	fmt.Println("==================================================")

	// We will loop through difficulty levels 1 to 5 to see how the effort scales
	for diff := 1; diff <= 5; diff++ {
		fmt.Printf("--- Initializing Chain Test with Difficulty: %d ---\n", diff)
		bc := chain.NewBlockchain(diff)

		// Queue up an initial transaction from the faucet
		_ = bc.AddTransaction(ledger.NewTransaction("faucet", "Amindya", 250.0))

		// Mine the block (FR-5) -> this will print out our nonces and elapsed time!
		_, _ = bc.MinePendingBlock()
	}

	fmt.Println("==================================================")
	fmt.Println("Tuning Complete! Choose a sweet spot for Day 3.")
}
