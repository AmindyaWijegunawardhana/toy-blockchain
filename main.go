package main

import (
	"toy-blockchain/chain"
	"toy-blockchain/cli"
)

func main() {
	bc := chain.NewBlockchain(1)
	c := cli.NewCLI(bc)
	c.Run()
}
