package main

import (
	"fmt"
	"os"

	"github.com/ethanblumenthal/golang-blockchain/node"
	"github.com/spf13/cobra"
)
func runCmd() *cobra.Command {
    var runCmd = &cobra.Command{
        Use: "run",
        Short: "Launches the blockchain node and its HTTP API.",
        Run: func(cmd *cobra.Command, args []string) {
            dataDir, _ := cmd.Flags().GetString(flagDataDir)
            port, _ := cmd.Flags().GetUint64(flagPort)
            
            fmt.Println("Launching blockchain node and its HTTP API...")

            bootstrap := node.NewPeerNode("", 8080, true, true)
            n := node.New(dataDir, port, bootstrap)

            err := n.Run()
            if err != nil {
                fmt.Println(err)
                os.Exit(1)
            }
        },
    }
    addDefaultRequiredFlags(runCmd)
    runCmd.Flags().Uint64(flagPort, node.DefaultHTTPPort, "exposed HTTP port for communication with peers")

    return runCmd
}