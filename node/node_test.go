package node

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethanblumenthal/golang-blockchain/database"
	"github.com/ethanblumenthal/golang-blockchain/fs"
	"github.com/ethanblumenthal/golang-blockchain/wallet"
	"github.com/ethereum/go-ethereum/common"
)

const testKsAccount1 = "0x3eb92807f1f91a8d4d85bc908c7f86dcddb1df57"
const testKsAccount2 = "0x6fdc0d8d15ae6b4ebf45c52fd2aafbcbb19a65c8"
const testKsAccount1File = "test_account1--3eb92807f1f91a8d4d85bc908c7f86dcddb1df57"
const testKsAccount2File = "test_account2--6fdc0d8d15ae6b4ebf45c52fd2aafbcbb19a65c8"
const testKsAccountsPwd = "security123"

func TestNode_Run(t *testing.T) {
	// Remove the test directory if it already exists
	datadir, err := getTestDataDirPath()
	if err != nil {
		t.Fatal(err)
	}

	err = fs.RemoveDir(datadir)
	if err != nil {
		t.Fatal(err)
	}

	// Construct a new node instance
	n := New(datadir, "127.0.0.1", 8085, database.NewAccount(DefaultMiner), PeerNode{})

	// Define a context with timeout for this test
	// Node will run for 5s
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err = n.Run(ctx)
	if err.Error() != "http: Server closed" {
		t.Fatal("node server was supposed to close after 5s")
	}
}

func TestNode_Mining(t *testing.T) {
	acc1 := database.NewAccount(testKsAccount1)
	acc2 := database.NewAccount(testKsAccount2)

	genesisBalances := make(map[common.Address]uint)
	genesisBalances[acc1] = 1000000
	genesis := database.Genesis{Balances: genesisBalances}
	genesisJson, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}

	dataDir, err := getTestDataDirPath()
	if err != nil {
		t.Fatal(err)
	}

	err = database.InitDataDirIfNotExists(dataDir, genesisJson)
	defer fs.RemoveDir(dataDir)

	err = copyKeystoreFilesIntoTestDataDirPath(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	// Construct a new node instance where the TX originated from
	nInfo := NewPeerNode("127.0.0.1", 8085, false, database.NewAccount(""), true)

	// Construct a new node instance and configure Ethan as a miner
	n := New(dataDir, nInfo.IP, nInfo.Port, acc1, nInfo)

	// Allow the mining to run for 30mins at most
	ctx, closeNode := context.WithTimeout(context.Background(), time.Minute * 30)

	// Schedule a new TX 3 seconds from now in a separate thread
	go func() {
		time.Sleep(time.Second * miningIntervalSeconds / 3)
		tx := database.NewTx(acc1, acc2, 1, "")

		signedTx, err := wallet.SignTxWithKeystoreAccount(tx, acc1, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
		if err != nil {
			t.Error(err)
			return
		}

		// Add it to the Mempool
		_ = n.AddPendingTX(signedTx, nInfo)
	}()

	// Schedule a new TX 12 seconds from now in a separate thread
	go func() {
		time.Sleep(time.Second * miningIntervalSeconds + 2)

		tx := database.NewTx(acc1, acc2, 2, "")
		signedTx, err := wallet.SignTxWithKeystoreAccount(tx, acc1, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
		if err != nil {
			t.Error(err)
			return
		}

		// Add it to the Mempool
		_ = n.AddPendingTX(signedTx, nInfo)
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

func TestNode_MiningStopsOnNewSyncedBlock(t *testing.T) {
	acc1 := database.NewAccount(testKsAccount1)
	acc2 := database.NewAccount(testKsAccount2)

	genesisBalances := make(map[common.Address]uint)
	genesisBalances[acc1] = 1000000
	genesis := database.Genesis{Balances: genesisBalances}
	genesisJson, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}

	dataDir, err := getTestDataDirPath()
	if err != nil {
		t.Fatal(err)
	}

	err = database.InitDataDirIfNotExists(dataDir, genesisJson)
	defer fs.RemoveDir(dataDir)

	err = copyKeystoreFilesIntoTestDataDirPath(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	// Construct a new node instance where the TX originated from
	nInfo := NewPeerNode("127.0.0.1", 8085, false, database.NewAccount(""), true)

	n := New(dataDir, nInfo.IP, nInfo.Port, acc2, nInfo)

	// Allow the mining to run for 30mins at most
	ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*30)

	tx1 := database.NewTx(acc1, acc2, 1, "")
	tx2 := database.NewTx(acc1, acc2, 2, "")

	signedTx1, err := wallet.SignTxWithKeystoreAccount(tx1, acc1, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
	if err != nil {
		t.Error(err)
		return
	}

	signedTx2, err := wallet.SignTxWithKeystoreAccount(tx2, acc1, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
	if err != nil {
		t.Error(err)
		return
	}
	tx2Hash, err := signedTx2.Hash()
	if err != nil {
		t.Error(err)
		return
	}

	// Pre-mine a valid block to simulate a block incoming from a peer
	validPreMinedPb := NewPendingBlock(database.Hash{}, 0, acc1, []database.SignedTx{signedTx1})
	validSyncedBlock, err := Mine(ctx, validPreMinedPb)
	if err != nil {
		t.Fatal(err)
	}

	// Add 2 new TXs to Ethan's node
	go func() {
		time.Sleep(time.Second * (miningIntervalSeconds - 2))

		err := n.AddPendingTX(signedTx1, nInfo)
		if err != nil {
			t.Fatal(err)
		}

		err = n.AddPendingTX(signedTx2, nInfo)
		if err != nil {
			t.Fatal(err)
		}
	}()

	// Simulate that Carley mined the block with TX1 faster
	go func () {
		time.Sleep(time.Second * (miningIntervalSeconds + 2))
		if !n.isMining {
			t.Fatal("should be mining")
		}

		_, err := n.state.AddBlock(validSyncedBlock)
		if !n.isMining {
			t.Fatal(err)
		}

		// Mock that Ethan's block came from a network
		n.newSyncedBlocks <- validSyncedBlock

		time.Sleep(time.Second * 2)
		if n.isMining {
			t.Fatal("synced block should have canceled mining")
		}

		// Mined TX1 by Ethan should be removed from the Mempool
		_, onlyTX2IsPending := n.pendingTXs[tx2Hash.Hex()]

		if len(n.pendingTXs) != 1 && !onlyTX2IsPending {
			t.Fatal("TX1 should still be pending")
		}

		time.Sleep(time.Second * (miningIntervalSeconds + 2))
		if !n.isMining {
			t.Fatal("should attempt to mine TX1 again")
		}
	}()

	go func() {
		// Regularly check if both TXs are now mined
		ticker := time.NewTicker(time.Second * 10)

		for {
			select {
			case <- ticker.C:
				if n.state.LatestBlock().Header.Number == 1 {
					closeNode()
					return
				}
			}
		}
	}()

	go func() {
		time.Sleep(time.Second * 2)

		startEthanBalance := n.state.Balances[acc1]
		startCarleyBalance := n.state.Balances[acc2]

		<- ctx.Done()

		// Query balances again
		endEthanBalance := n.state.Balances[acc1]
		endCarleyBalance := n.state.Balances[acc2]

		// In TX1 ethan transferred 1 gochain token to carley
		// In TX2 ethan transferred 2 gochain tokens to carley
		expectedEndEthanBalance := startEthanBalance - tx1.Value - tx2.Value + database.BlockReward
		expectedEndCarleyBalance := startCarleyBalance + tx1.Value + tx2.Value + database.BlockReward

		if endEthanBalance != expectedEndEthanBalance {
			t.Fatalf("Ethan expected end balance is %d not %d", expectedEndEthanBalance, startEthanBalance)
		}

		if endCarleyBalance != expectedEndCarleyBalance {
			t.Fatalf("Carley expected end balance is %d not %d", expectedEndCarleyBalance, startCarleyBalance)
		}

		t.Logf("Start Ethan balance: %d", startEthanBalance)
        t.Logf("Start Carley balance: %d", startCarleyBalance)
        t.Logf("End Ethan balance: %d", endEthanBalance)
        t.Logf("End Carley balance: %d", endCarleyBalance)
	}()
	
	_ = n.Run(ctx)

	if n.state.LatestBlock().Header.Number != 1 {
		t.Fatal("2 pending TX not mined into 2 blocks under 30m")
	}

	if len(n.pendingTXs) != 0 {
		t.Fatal("no pending TXs should be left to mine")
	}
}

func getTestDataDirPath() (string, error) {
	return ioutil.TempDir(os.TempDir(), "gochain_test")
}

func copyKeystoreFilesIntoTestDataDirPath(dataDir string) error {
	account1SrcKs, err := os.Open(testKsAccount1File)
	if err != nil {
		return err
	}
	defer account1SrcKs.Close()

	ksDir := filepath.Join(wallet.GetKeystoreDirPath(dataDir))

	err = os.Mkdir(ksDir, 0777)
	if err != nil {
		return err
	}

	account1DstKs, err := os.Create(filepath.Join(ksDir, testKsAccount1File))
	if err != nil {
		return err
	}
	defer account1DstKs.Close()

	_, err = io.Copy(account1DstKs, account1SrcKs)
	if err != nil {
		return err
	}

	account2SrcKs, err := os.Open(testKsAccount2File)
	if err != nil {
		return err
	}
	defer account2SrcKs.Close()

	account2DstKs, err := os.Create(filepath.Join(ksDir, testKsAccount2File))
	if err != nil {
		return err
	}
	defer account2DstKs.Close()

	_, err = io.Copy(account2DstKs, account2SrcKs)
	if err != nil {
		return err
	}

	return nil
} 