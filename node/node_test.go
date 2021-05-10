package node

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethanblumenthal/golang-blockchain/database"
	"github.com/ethanblumenthal/golang-blockchain/fs"
)

func getTestDataDirPath() string {
	return filepath.Join(os.TempDir(), ".gochain_test")
}

func TestNode_Run(t *testing.T) {
	// Remove the test directory if it already exists
	datadir := getTestDataDirPath()
	err := fs.RemoveDir(datadir)
	if err != nil {
		t.Fatal(err)
	}

	// Construct a new node instance
	n := New(datadir, "127.0.0.1", 8085, database.NewAccount("ethan"), PeerNode{})

	// Define a context with timeout for this test
	// Node will run for 5s
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	err = n.Run(ctx)
	if err.Error() != "http: Server closed" {
		// Assert expected behavior
		t.Fatal("node server was supposed to close after 5s")
	}
}

func TestNode_Mining(t *testing.T) {
	datadir := getTestDataDirPath()
	err := fs.RemoveDir(datadir)
	if err != nil {
		t.Fatal(err)
	}

	// Construct a new node instance where the TX originated from
	nInfo := NewPeerNode("127.0.0.1", 8085, false, database.NewAccount(""), true)

	// Construct a new node instance and configure Ethan as a miner
	n := New(datadir, nInfo.IP, nInfo.Port, database.NewAccount("ethan"), nInfo)

	// Allow the mining to run for 30mins at most
	ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*30)

	// Schedule a new TX 3 seconds from now in a separate thread
	go func() {
		time.Sleep(time.Second * miningIntervalSeconds / 3)
		tx := database.NewTx("ethan", "carley", 1, "")

		// Add it to the Mempool
		_ = n.AddPendingTX(tx, nInfo)
	}()

	go func() {
		// Periodically check if we mined the 2 blocks
		ticker := time.NewTicker(time.Second * 10)

		for {
			select {
			case <- ticker.C:
				// Close node if 2 blocks are mined as expected
				if n.state.LatestBlock().Header.Number == 1 {
					closeNode()
					return
				}
			}
		}
	}()

	_ = n.Run(ctx)

	// Assert test result
	if n.state.LatestBlock().Header.Number != 1 {
		t.Fatal("2 pending TX not mined into 2 under 30min")
	}
}