package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var cmd = &cobra.Command{
		Use: "goChain",
		Short: "Golang blockchain CLI",
		Run: func(cmd *cobra.Command, args []string) {},
	}

	err := cmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}