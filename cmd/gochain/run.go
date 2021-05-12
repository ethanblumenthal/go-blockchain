package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ethanblumenthal/golang-blockchain/database"
	"github.com/ethanblumenthal/golang-blockchain/node"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Launches the GoChain node and its HTTP API.",
		Run: func(cmd *cobra.Command, args []string) {
			miner, _ := cmd.Flags().GetString(flagMiner)
			sslEmail, _ := cmd.Flags().GetString(flagSSLEmail)
			isSSLDisabled, _ := cmd.Flags().GetBool(flagDisableSSL)
			ip, _ := cmd.Flags().GetString(flagIP)
			port, _ := cmd.Flags().GetUint64(flagPort)
			bootstrapIp, _ := cmd.Flags().GetString(flagBootstrapIp)
			bootstrapPort, _ := cmd.Flags().GetUint64(flagBootstrapPort)
			bootstrapAcc, _ := cmd.Flags().GetString(flagBootstrapAcc)

			fmt.Println("Launching GoChain node and its HTTP API...")

			bootstrap := node.NewPeerNode(
				bootstrapIp,
				bootstrapPort,
				true,
				database.NewAccount(bootstrapAcc),
				false,
			)

			if !isSSLDisabled {
				port = node.HttpSSLPort
			}

			n := node.New(getDataDirFromCmd(cmd), ip, port, database.NewAccount(miner), bootstrap)
			err := n.Run(context.Background(), isSSLDisabled, sslEmail)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}

	addDefaultRequiredFlags(runCmd)
	runCmd.Flags().Bool(flagDisableSSL, false, "should the HTTP API SSL certificate be disabled? (default false)")
	runCmd.Flags().String(flagSSLEmail, "", "your node's HTTP SSL certificate email")
	runCmd.Flags().String(flagMiner, node.DefaultMiner, "your node's miner account to receive the block rewards")
	runCmd.Flags().String(flagIP, node.DefaultIP, "your node's public IP to communication with other peers")
	runCmd.Flags().Uint64(flagPort, node.HttpSSLPort, "your node's public HTTP port for communication with other peers (configurable if SSL is disabled)")
	runCmd.Flags().String(flagBootstrapIp, node.DefaultBootstrapIp, "default GoChain bootstrap's server to interconnect peers")
	runCmd.Flags().Uint64(flagBootstrapPort, node.HttpSSLPort, "default GoChain bootstrap's server port to interconnect peers")
	runCmd.Flags().String(flagBootstrapAcc, node.DefaultBootstrapAcc, "default GoChain bootstrap's Genesis account with 1M tokens")

	return runCmd
}