package main

import (
	"flag"
	"fmt"
	"toy-blockchain/chain"
	"toy-blockchain/cli"
)

func main() {
	// Add configurable flag parameter (Fix #4)
	diffFlag := flag.Int("difficulty", 4, "Set mining zero bits difficulty runtime variable")
	flag.Parse()

	bc := chain.NewBlockchain(*diffFlag)

	fmt.Printf("[Init] Starting network instance with Difficulty target set to: %d\n", *diffFlag)

	terminal := cli.NewCLI(bc)
	terminal.Run()
}
