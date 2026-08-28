package cli

import (
	"flag"
	"fmt"
	"os"

	"toy-blockchain/chain"
	"toy-blockchain/ledger"
)

// CLI provides command-line interface capabilities to interact with the blockchain.
type CLI struct {
	bc       *chain.Blockchain
	dataFile string
}

// NewCLI initializes a new CLI instance. Accepts an optional dataFile argument.
func NewCLI(bc *chain.Blockchain, dataFile ...string) *CLI {
	file := "blockchain.json"
	if len(dataFile) > 0 && dataFile[0] != "" {
		file = dataFile[0]
	}
	return &CLI{bc: bc, dataFile: file}
}

// Run parses arguments and executes corresponding commands.
func (c *CLI) Run() {
	if len(os.Args) < 2 {
		c.printUsage()
		return
	}

	_ = c.bc.LoadFromFile(c.dataFile)

	switch os.Args[1] {
	case "mine":
		c.handleMine()
	case "balances":
		c.handleBalances()
	case "validate":
		c.handleValidate()
	default:
		c.printUsage()
	}
}

func (c *CLI) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  mine -recipient <address> -amount <amount>")
	fmt.Println("  balances")
	fmt.Println("  validate")
}

func (c *CLI) handleMine() {
	mineCmd := flag.NewFlagSet("mine", flag.ExitOnError)
	recipient := mineCmd.String("recipient", "", "Recipient address")
	amount := mineCmd.Int64("amount", 0, "Amount to transfer")
	mineCmd.Parse(os.Args[2:])

	if *recipient != "" && *amount > 0 {
		tx := ledger.NewTransaction("", *recipient, *amount)
		_ = c.bc.AddTransaction(tx)
	}

	b, err := c.bc.MinePendingBlock()
	if err != nil {
		fmt.Printf("Mining failed: %v\n", err)
		return
	}

	_ = c.bc.SaveToFile(c.dataFile)
	fmt.Printf("Block #%d mined! Hash: %s\n", b.Index, b.Hash)
}

func (c *CLI) handleBalances() {
	balances := c.bc.GetBalances()
	fmt.Println("Ledger Balances:")
	for addr, bal := range balances {
		fmt.Printf("  %s: %d\n", addr, bal)
	}
}

func (c *CLI) handleValidate() {
	valid, err := c.bc.ValidateChain()
	if valid && err == nil {
		fmt.Println("Blockchain is valid.")
	} else {
		fmt.Printf("Blockchain is invalid: %v\n", err)
	}
}
