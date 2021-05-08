package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const flagDataDir = "datadir"
const flagPort = "port"

func main() {
	var blockchainCmd = &cobra.Command{
		Use: "blockchain",
		Short: "Golang blockchain CLI",
		Run: func(cmd *cobra.Command, args []string) {},
	}

	blockchainCmd.AddCommand(migrateCmd())
	blockchainCmd.AddCommand(versionCmd)
	blockchainCmd.AddCommand(balancesCmd())
	blockchainCmd.AddCommand(runCmd())

	err := blockchainCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func addDefaultRequiredFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagDataDir, "", "Absolute path to the node data dir where the DB will/is stored")
	cmd.MarkFlagRequired(flagDataDir)
}

func incorrectUsageErr() error {
	return fmt.Errorf(("incorrect usage"))
}