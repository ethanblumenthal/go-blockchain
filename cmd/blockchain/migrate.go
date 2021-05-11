package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ethanblumenthal/golang-blockchain/database"
	"github.com/ethanblumenthal/golang-blockchain/node"
	"github.com/ethanblumenthal/golang-blockchain/wallet"
	"github.com/spf13/cobra"
)

var migrateCmd = func() *cobra.Command {
	var migrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Migrates the gochain database according to new business rules.",
		Run: func(cmd *cobra.Command, args []string) {
			ip, _ := cmd.Flags().GetString(flagIP)
			port, _ := cmd.Flags().GetUint64(flagPort)
			miner, _ := cmd.Flags().GetString(flagMiner)

			acc1 := database.NewAccount(wallet.Account1)
			acc2 := database.NewAccount(wallet.Account2)
			acc3 := database.NewAccount(wallet.Account3)

			peer := node.NewPeerNode("127.0.0.1", 8080, true, acc1, false)
			n := node.New(getDataDirFromCmd(cmd), ip, port, database.NewAccount(miner), peer)

			n.AddPendingTX(database.NewTx(acc1, acc1, 3, ""), peer)
			n.AddPendingTX(database.NewTx(acc1, acc2, 2000, ""), peer)
			n.AddPendingTX(database.NewTx(acc2, acc1, 1, ""), peer)
			n.AddPendingTX(database.NewTx(acc2, acc3, 1000, ""), peer)
			n.AddPendingTX(database.NewTx(acc2, acc1, 50, ""), peer)

			ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*15)

			go func() {
				ticker := time.NewTicker(time.Second * 10)

				for {
					select {
					case <-ticker.C:
						if !n.LatestBlockHash().IsEmpty() {
							closeNode()
							return
						}
					}
				}
			}()

			err := n.Run(ctx)
			if err != nil {
				fmt.Println(err)
			}
		},
	}

	addDefaultRequiredFlags(migrateCmd)
	migrateCmd.Flags().String(flagMiner, node.DefaultMiner, "miner account of this node to receive block rewards")
	migrateCmd.Flags().String(flagIP, node.DefaultIP, "exposed IP for communication with peers")
	migrateCmd.Flags().Uint64(flagPort, node.DefaultHTTPPort, "exposed HTTP port for communication with peers")

	return migrateCmd
}