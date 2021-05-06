package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var blockchainCmd = &cobra.Command{
		Use: "blockchain",
		Short: "Golang blockchain CLI",
		Run: func(cmd *cobra.Command, args []string) {},
	}

	blockchainCmd.AddCommand(versionCmd)
	blockchainCmd.AddCommand(balancesCmd())
	blockchainCmd.AddCommand(txCmd())

	err := blockchainCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func incorrectUsageErr() error {
	return fmt.Errorf(("incorrect usage"))
}