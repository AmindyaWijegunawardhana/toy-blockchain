package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"toy-blockchain/chain"
	"toy-blockchain/ledger"
)

const dbFile = "blockchain.json"

// CLI holds the reference to our running blockchain instance.
type CLI struct {
	bc *chain.Blockchain
}

// NewCLI initializes a new command-line helper.
func NewCLI(bc *chain.Blockchain) *CLI {
	return &CLI{bc: bc}
}

// Run executes the continuous interactive terminal loop (FR-7).
func (c *CLI) Run() {
	// Attempt to reload data from disk file on startup if it exists
	if err := c.bc.LoadFromFile(dbFile); err == nil {
		fmt.Printf("[System] Loaded existing blockchain from %s\n", dbFile)
	} else {
		fmt.Println("[System] No previous chain database found. Starting with a fresh Genesis block.")
		_ = c.bc.SaveToFile(dbFile)
	}

	scanner := bufio.NewScanner(os.Stdin)
	c.printHelp()

	for {
		fmt.Print("\ntoy-chain > ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := strings.ToLower(parts[0])

		switch command {
		case "tx":
			c.handleAddTransaction(parts)
		case "mine":
			c.handleMineBlock()
		case "balance":
			c.handleBalances()
		case "print":
			c.handlePrintChain()
		case "validate":
			c.handleValidateChain()
		case "help":
			c.printHelp()
		case "exit":
			fmt.Println("Exiting blockchain interface. Goodbye!")
			return
		default:
			fmt.Println("Unknown command. Type 'help' to see available options.")
		}
	}
}

func (c *CLI) printHelp() {
	fmt.Println("\n=== Toy Blockchain Command Terminal ===")
	fmt.Println("Available Commands:")
	fmt.Println("  tx <sender> <recipient> <amount>  : Add a pending transaction")
	fmt.Println("  mine                             : Mine a new block from the pending pool")
	fmt.Println("  balance                          : Display current account balances")
	fmt.Println("  print                            : Print out the complete block sequence")
	fmt.Println("  validate                         : Run full-chain integrity validation checks")
	fmt.Println("  help                             : Show this menu screen")
	fmt.Println("  exit                             : Terminate the simulator securely")
	fmt.Println("=======================================")
}

func (c *CLI) handleAddTransaction(parts []string) {
	if len(parts) != 4 {
		fmt.Println("Error: Invalid transaction format. Use: tx <sender> <recipient> <amount>")
		return
	}

	sender := parts[1]
	recipient := parts[2]
	amount, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil || amount <= 0 {
		fmt.Println("Error: Invalid amount value. Must be a valid integer coin unit.")
		return
	}

	tx := ledger.NewTransaction(sender, recipient, amount)
	if err := c.bc.AddTransaction(tx); err != nil {
		fmt.Printf("Transaction Rejected: %v\n", err)
	} else {
		fmt.Printf("Transaction queued successfully! (Sender: %s -> Recipient: %s | Amount: %d)\n", sender, recipient, amount)
	}
}

func (c *CLI) handleMineBlock() {
	block, err := c.bc.MinePendingBlock()
	if err != nil {
		fmt.Printf("Mining Failed: %v\n", err)
		return
	}

	// Auto-save the state change to file storage
	if err := c.bc.SaveToFile(dbFile); err != nil {
		fmt.Printf("Warning: Failed to save chain state to file: %v\n", err)
	} else {
		fmt.Printf("State automatically synchronized to %s.\n", dbFile)
	}
	fmt.Printf("Block #%d linked securely! Hash: %s\n", block.Index, block.Hash)
}

func (c *CLI) handleBalances() {
	balances := c.bc.GetBalances()
	fmt.Println("--- Account Balances ---")
	if len(balances) == 0 {
		fmt.Println("No accounts register ledger transactions yet.")
		return
	}
	for account, balance := range balances {
		fmt.Printf("  %s: %d\n", account, balance)
	}
}

func (c *CLI) handlePrintChain() {
	fmt.Println("--- Traversing Blockchain Core ---")
	for _, b := range c.bc.Blocks {
		fmt.Printf("\nBlock Index:  %d\n", b.Index)
		fmt.Printf("  Timestamp:  %d\n", b.Timestamp)
		fmt.Printf("  Prev Hash:  %s\n", b.PrevHash)
		fmt.Printf("  Block Hash: %s\n", b.Hash)
		fmt.Printf("  Nonce:      %d\n", b.Nonce)
		fmt.Printf("  Transactions (%d total):\n", len(b.Transactions))
		for _, tx := range b.Transactions {
			fmt.Printf("    - %s -> %s: %d\n", tx.Sender, tx.Recipient, tx.Amount)
		}
	}
}

func (c *CLI) handleValidateChain() {
	valid, brokenIdx, err := c.bc.ValidateChain()
	if valid {
		fmt.Println("Success: Blockchain validation passed! Network structure is fully secure.")
	} else {
		fmt.Printf("ALERT: Chain validation failed at Block Index %d!\n", brokenIdx)
		fmt.Printf("Reason for failure: %v\n", err)
	}
}
