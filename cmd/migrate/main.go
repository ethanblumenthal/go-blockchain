package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ethanblumenthal/golang-blockchain/database"
)

func main() {
	cwd, _ := os.Getwd()
	state, err := database.NewStateFromDisk(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer state.Close()

	block0 := database.NewBlock(
		database.Hash{},
		uint64(time.Now().Unix()),
		[]database.Tx{
			database.NewTx("ethan", "ethan", 3, ""),
			database.NewTx("ethan", "ethan", 700, "reward"),
		},
	)

	state.AddBlock(block0)
	block0hash, _ := state.Persist()

	block1 := database.NewBlock(
		block0hash,
		uint64(time.Now().Unix()),
		[]database.Tx{
			database.NewTx("ethan", "carley", 2000, ""),
			database.NewTx("ethan", "ethan", 100, "reward"),
			database.NewTx("carley", "ethan", 1, ""),
			database.NewTx("carley", "drew", 1000, ""),
			database.NewTx("carley", "ethan", 50, ""),
			database.NewTx("ethan", "ethan", 600, "reward"),
		},
	)

	state.AddBlock(block1)
	state.Persist()
}