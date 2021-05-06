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

	err := blockchainCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}