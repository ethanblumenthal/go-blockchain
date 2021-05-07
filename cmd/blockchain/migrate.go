package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ethanblumenthal/golang-blockchain/database"
	"github.com/spf13/cobra"
)

var migrateCmd = func() *cobra.Command {
	var migrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Migrates the blockchain database according to new business rules.",
		Run: func(cmd *cobra.Command, args []string) {
			dataDir, _ := cmd.Flags().GetString(flagDataDir)

			state, err := database.NewStateFromDisk(dataDir)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			defer state.Close()

			block0 := database.NewBlock(
				database.Hash{},
				0,
				uint64(time.Now().Unix()),
				[]database.Tx{
					database.NewTx("ethan", "ethan", 3, ""),
					database.NewTx("ethan", "ethan", 700, "reward"),
				},
			)

			state.AddBlock(block0)
			block0hash, err := state.Persist()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			block1 := database.NewBlock(
				block0hash,
				1,
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
			block1hash, err := state.Persist()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			block2 := database.NewBlock(
				block1hash,
				2,
				uint64(time.Now().Unix()),
				[]database.Tx{
					database.NewTx("ethan", "ethan", 24700, "reward"),
				},
			)

			state.AddBlock(block2)
			_, err = state.Persist()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}

	addDefaultRequiredFlags(migrateCmd)

	return migrateCmd
}