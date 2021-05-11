package main

import (
	"fmt"
	"os"

	"github.com/ethanblumenthal/golang-blockchain/fs"
	"github.com/spf13/cobra"
)

const flagDataDir = "datadir"
const flagIP = "ip"
const flagPort = "port"
const flagMiner = "miner"
const flagKeystoreFile = "keystore"
const flagBootstrapAcc = "bootstrap-account"
const flagBootstrapIp = "bootstrap-ip"
const flagBootstrapPort = "bootstrap-port"

func main() {
	var gochainCmd = &cobra.Command{
		Use: "gochain",
		Short: "Golang blockchain command line interface (CLI).",
		Run: func(cmd *cobra.Command, args []string) {},
	}

	gochainCmd.AddCommand(migrateCmd())
	gochainCmd.AddCommand(versionCmd)
	gochainCmd.AddCommand(balancesCmd())
	gochainCmd.AddCommand(walletCmd())
	gochainCmd.AddCommand(runCmd())

	err := gochainCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func addDefaultRequiredFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagDataDir, "", "Absolute path to the node data dir where the database will be stored")
	cmd.MarkFlagRequired(flagDataDir)
}

func addKeystoreFlag(cmd *cobra.Command) {
	cmd.Flags().String(flagDataDir, "", "Absolute path to the encrypted keystore file")
	cmd.MarkFlagRequired(flagKeystoreFile)
}

func getDataDirFromCmd(cmd *cobra.Command) string {
	dataDir, _ := cmd.Flags().GetString(flagDataDir)
	return fs.ExpandPath(dataDir)
}

func incorrectUsageErr() error {
	return fmt.Errorf(("incorrect usage"))
}