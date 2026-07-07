package main

import (
	"fmt"
	"toy-blockchain/chain"
	"toy-blockchain/ledger"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("       DAY 3: LEDGER STATE & VALIDATION ENGINE    ")
	fmt.Println("==================================================")

	// Default difficulty set to 3 for immediate local feedback
	bc := chain.NewBlockchain(3)

	// 1. Initial funding via faucet mechanism (FR-4)
	fmt.Println("\n[1] Minting initial tokens via faucet...")
	_ = bc.AddTransaction(ledger.NewTransaction("faucet", "Amindya", 100.0))
	_, _ = bc.MinePendingBlock()

	// 2. Attempting to spend more than available balance (FR-4 Scenario)
	fmt.Println("\n[2] Testing overspending validation...")
	err := bc.AddTransaction(ledger.NewTransaction("Amindya", "Bob", 150.0))
	if err != nil {
		fmt.Printf(" >>> Caught successfully: %v\n", err)
	}

	// 3. Submitting a legitimate payment
	fmt.Println("\n[3] Processing a valid transaction...")
	_ = bc.AddTransaction(ledger.NewTransaction("Amindya", "Bob", 40.0))
	_, _ = bc.MinePendingBlock()

	// Displaying dynamic state changes
	balances := bc.GetBalances()
	fmt.Printf(" >>> Current Balances -> Amindya: %.2f | Bob: %.2f\n", balances["Amindya"], balances["Bob"])

	// 4. Checking structural validation of the honest ledger state (FR-6)
	valid, brokenIdx, valErr := bc.ValidateChain()
	fmt.Printf("\n[4] Honest Chain Integrity Check: Valid = %t, Broken Index = %d, Err = %v\n", valid, brokenIdx, valErr)

	// 5. Simulating malicious history alteration (Research Component 7.1)
	fmt.Println("\n[5] ATTACK SCENARIO: Modifying historical transaction inside Block 1...")
	bc.Blocks[1].Transactions[0].Amount = 5000.0 // Fraudulent shift

	// Run full validation check again
	valid, brokenIdx, valErr = bc.ValidateChain()
	fmt.Printf(" >>> Post-Tamper Status: Valid = %t\n", valid)
	fmt.Printf(" >>> Identified Offending Block Index: %d\n", brokenIdx)
	fmt.Printf(" >>> Validator Reason: %v\n", valErr)
}
