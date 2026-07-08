package main

import (
	"toy-blockchain/chain"
	"toy-blockchain/cli"
)

func main() {
	// Initialize with a default difficulty of 4 leading zeros
	bc := chain.NewBlockchain(4)

	// Transfer execution control to our interactive terminal shell
	terminal := cli.NewCLI(bc)
	terminal.Run()
}
