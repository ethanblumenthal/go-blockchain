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
	datadir, err := getTestDataDirPath()
	if err != nil {
		t.Fatal(err)
	}
	err = fs.RemoveDir(datadir)
	if err != nil {
		t.Fatal(err)
	}

	n := New(datadir, "127.0.0.1", 8085, database.NewAccount(DefaultMiner), PeerNode{})

	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	err = n.Run(ctx, true, "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestNode_Mining(t *testing.T) {
	dataDir, account1, account2, err := setupTestNodeDir(1000000)
	if err != nil {
		t.Error(err)
	}
	defer fs.RemoveDir(dataDir)

	// Required for AddPendingTX() to describe
	// from what node the TX came from (local node in this case)
	nInfo := NewPeerNode(
		"127.0.0.1",
		8085,
		false,
		account2,
		true,
	)

	// Construct a new Node instance and configure
	// account1 as a miner
	n := New(dataDir, nInfo.IP, nInfo.Port, account1, nInfo)

	// Allow the mining to run for 30 mins, in the worst case
	ctx, closeNode := context.WithTimeout(
		context.Background(),
		time.Minute*30,
	)

	// Schedule a new TX in 3 seconds from now, in a separate thread
	// because the n.Run() few lines below is a blocking call
	go func() {
		time.Sleep(time.Second * miningIntervalSeconds / 3)

		tx := database.NewTx(account1, account2, 1, 1, "")
		signedTx, err := wallet.SignTxWithKeystoreAccount(tx, account1, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
		if err != nil {
			t.Error(err)
			return
		}

		_ = n.AddPendingTX(signedTx, nInfo)
	}()

	// Schedule a TX with insufficient funds in 4 seconds validating
	// the AddPendingTX won't add it to the Mempool
	go func() {
		time.Sleep(time.Second*(miningIntervalSeconds/3) + 1)

		tx := database.NewTx(account2, account1, 50, 1, "")
		signedTx, err := wallet.SignTxWithKeystoreAccount(tx, account2, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
		if err != nil {
			t.Error(err)
			return
		}

		err = n.AddPendingTX(signedTx, nInfo)
		t.Log(err)
		if err == nil {
			t.Errorf("TX should not be added to Mempool because account2 doesn't have %d tokens", tx.Value)
			return
		}
	}()

	// Schedule a new TX in 12 seconds from now simulating
	// that it came in - while the first TX is being mined
	go func() {
		time.Sleep(time.Second * (miningIntervalSeconds + 2))

		tx := database.NewTx(account1, account2, 2, 2, "")
		signedTx, err := wallet.SignTxWithKeystoreAccount(tx, account1, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
		if err != nil {
			t.Error(err)
			return
		}

		err = n.AddPendingTX(signedTx, nInfo)
		if err != nil {
			t.Error(err)
			return
		}
	}()

	go func() {
		// Periodically check if we mined the 2 blocks
		ticker := time.NewTicker(10 * time.Second)

		for {
			select {
			case <-ticker.C:
				if n.state.LatestBlock().Header.Number == 1 {
					closeNode()
					return
				}
			}
		}
	}()

	// Run the node, mining and everything in a blocking call (hence the go-routines before)
	_ = n.Run(ctx, true, "")

	if n.state.LatestBlock().Header.Number != 1 {
		t.Fatal("2 pending TX not mined into 2 blocks under 30m")
	}
}

func TestNode_ForgedTx(t *testing.T) {
	dataDir, account1, account2, err := setupTestNodeDir(1000000)
	if err != nil {
		t.Error(err)
	}
	defer fs.RemoveDir(dataDir)

	n := New(dataDir, "127.0.0.1", 8085, account1, PeerNode{})
	ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*30)
	account1PeerNode := NewPeerNode("127.0.0.1", 8085, false, account1, true)

	txValue := uint(5)
	txNonce := uint(1)
	tx := database.NewTx(account1, account2, txValue, txNonce, "")

	validSignedTx, err := wallet.SignTxWithKeystoreAccount(tx, account1, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
	if err != nil {
		t.Error(err)
		closeNode()
		return
	}

	go func() {
		// Wait for the node to run
		time.Sleep(time.Second * 1)

		err = n.AddPendingTX(validSignedTx, account1PeerNode)
		if err != nil {
			t.Error(err)
			closeNode()
			return
		}
	}()

	go func() {
		ticker := time.NewTicker(time.Second * (miningIntervalSeconds - 3))
		wasForgedTxAdded := false

		for {
			select {
			case <-ticker.C:
				if !n.state.LatestBlockHash().IsEmpty() {
					if wasForgedTxAdded && !n.isMining {
						closeNode()
						return
					}

					if !wasForgedTxAdded {
						// Attempt to forge the same TX but with modified time
						// Because the TX.time changed, the TX.signature will be considered forged
						// database.NewTx() changes the TX time
						forgedTx := database.NewTx(account1, account2, txValue, txNonce, "")
						// Use the signature from a valid TX
						forgedSignedTx := database.NewSignedTx(forgedTx, validSignedTx.Sig)

						err = n.AddPendingTX(forgedSignedTx, account1PeerNode)
						t.Log(err)
						if err == nil {
							t.Errorf("adding a forged TX to the Mempool should not be possible")
							closeNode()
							return
						}

						wasForgedTxAdded = true

						time.Sleep(time.Second * (miningIntervalSeconds + 3))
					}
				}
			}
		}
	}()

	_ = n.Run(ctx, true, "")

	if n.state.LatestBlock().Header.Number != 0 {
		t.Fatal("was suppose to mine only one TX. The second TX was forged")
	}

	if n.state.Balances[account2] != txValue {
		t.Fatal("forged tx succeeded")
	}
}

func TestNode_ReplayedTx(t *testing.T) {
	dataDir, account1, account2, err := setupTestNodeDir(1000000)
	if err != nil {
		t.Error(err)
	}
	defer fs.RemoveDir(dataDir)

	n := New(dataDir, "127.0.0.1", 8085, account1, PeerNode{})
	ctx, closeNode := context.WithCancel(context.Background())
	account1PeerNode := NewPeerNode("127.0.0.1", 8085, false, account1, true)
	account2PeerNode := NewPeerNode("127.0.0.1", 8086, false, account2, true)

	txValue := uint(5)
	txNonce := uint(1)
	tx := database.NewTx(account1, account2, txValue, txNonce, "")

	signedTx, err := wallet.SignTxWithKeystoreAccount(tx, account1, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
	if err != nil {
		t.Error(err)
		closeNode()
		return
	}

	go func() {
		// Wait for the node to run
		time.Sleep(time.Second * 1)

		err = n.AddPendingTX(signedTx, account1PeerNode)
		if err != nil {
			t.Error(err)
			closeNode()
			return
		}
	}()

	go func() {
		ticker := time.NewTicker(time.Second * (miningIntervalSeconds - 3))
		wasReplayedTxAdded := false

		for {
			select {
			case <-ticker.C:
				if !n.state.LatestBlockHash().IsEmpty() {
					if wasReplayedTxAdded && !n.isMining {
						closeNode()
						return
					}

					// The account1's original TX got mined.
					// Execute the attack by replaying the TX again!
					if !wasReplayedTxAdded {
						// Simulate the TX was submitted to different node
						n.archivedTXs = make(map[string]database.SignedTx)
						// Execute the attack
						err = n.AddPendingTX(signedTx, account2PeerNode)
						t.Log(err)
						if err == nil {
							t.Errorf("re-adding a TX to the Mempool should not be possible because of Nonce")
							closeNode()
							return
						}

						wasReplayedTxAdded = true

						time.Sleep(time.Second * (miningIntervalSeconds + 3))
					}
				}
			}
		}
	}()

	_ = n.Run(ctx, true, "")

	if n.state.Balances[account2] == txValue*2 {
		t.Errorf("replayed attack was successful :( Damn digital signatures!")
		return
	}

	if n.state.Balances[account2] != txValue {
		t.Errorf("replayed attack was successful :( Damn digital signatures!")
		return
	}

	if n.state.LatestBlock().Header.Number == 1 {
		t.Errorf("the second block was not suppose to be persisted because it contained a malicious TX")
		return
	}
}

func TestNode_MiningStopsOnNewSyncedBlock(t *testing.T) {
	account2 := database.NewAccount(testKsAccount1)
	account1 := database.NewAccount(testKsAccount2)

	dataDir, err := getTestDataDirPath()
	if err != nil {
		t.Fatal(err)
	}

	genesisBalances := make(map[common.Address]uint)
	genesisBalances[account1] = 1000000
	genesis := database.Genesis{Balances: genesisBalances}
	genesisJson, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}

	err = database.InitDataDirIfNotExists(dataDir, genesisJson)
	defer fs.RemoveDir(dataDir)

	err = copyKeystoreFilesIntoTestDataDirPath(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	// Required for AddPendingTX() to describe
	// from what node the TX came from (local node in this case)
	nInfo := NewPeerNode(
		"127.0.0.1",
		8085,
		false,
		database.NewAccount(""),
		true,
	)

	n := New(dataDir, nInfo.IP, nInfo.Port, account2, nInfo)

	// Allow the test to run for 30 mins, in the worst case
	ctx, closeNode := context.WithTimeout(context.Background(), time.Minute*30)

	tx1 := database.NewTx(account1, account2, 1, 1, "")
	tx2 := database.NewTx(account1, account2, 2, 2, "")

	signedTx1, err := wallet.SignTxWithKeystoreAccount(tx1, account1, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
	if err != nil {
		t.Error(err)
		return
	}

	signedTx2, err := wallet.SignTxWithKeystoreAccount(tx2, account1, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
	if err != nil {
		t.Error(err)
		return
	}
	tx2Hash, err := signedTx2.Hash()
	if err != nil {
		t.Error(err)
		return
	}

	// Pre-mine a valid block without running the `n.Run()`
	// with account1 as a miner who will receive the block reward,
	// to simulate the block came on the fly from another peer
	validPreMinedPb := NewPendingBlock(database.Hash{}, 0, account1, []database.SignedTx{signedTx1})
	validSyncedBlock, err := Mine(ctx, validPreMinedPb)
	if err != nil {
		t.Fatal(err)
	}

	// Add 2 new TXs into the account2's node, triggers mining
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

	go func() {
		time.Sleep(time.Second * (miningIntervalSeconds + 2))
		if !n.isMining {
			t.Fatal("should be mining")
		}

		_, err := n.state.AddBlock(validSyncedBlock)
		if err != nil {
			t.Fatal(err)
		}
		// Mock the account1's block came from a network
		n.newSyncedBlocks <- validSyncedBlock

		time.Sleep(time.Second * 2)
		if n.isMining {
			t.Fatal("synced block should have canceled mining")
		}

		// Mined TX1 by account1 should be removed from the Mempool
		_, onlyTX2IsPending := n.pendingTXs[tx2Hash.Hex()]

		if len(n.pendingTXs) != 1 && !onlyTX2IsPending {
			t.Fatal("synced block should have canceled mining of already mined TX")
		}

		time.Sleep(time.Second * (miningIntervalSeconds + 2))
		if !n.isMining {
			t.Fatal("should be mining again the 1 TX not included in synced block")
		}
	}()

	go func() {
		// Regularly check whenever both TXs are now mined
		ticker := time.NewTicker(time.Second * 10)

		for {
			select {
			case <-ticker.C:
				if n.state.LatestBlock().Header.Number == 1 {
					closeNode()
					return
				}
			}
		}
	}()

	go func() {
		time.Sleep(time.Second * 2)

		// Take a snapshot of the DB balances
		// before the mining is finished and the 2 blocks
		// are created.
		startingAccount1Balance := n.state.Balances[account1]
		startingAccount2Balance := n.state.Balances[account2]

		// Wait until the 30 mins timeout is reached or
		// the 2 blocks got already mined and the closeNode() was triggered
		<-ctx.Done()

		endAccount1Balance := n.state.Balances[account1]
		endAccount2Balance := n.state.Balances[account2]

		// In TX1 account1 transferred 1 token to account2
		// In TX2 account1 transferred 2 tokens to account2
		expectedEndAccount1Balance := startingAccount1Balance - tx1.Cost() - tx2.Cost() + database.BlockReward + database.TxFee
		expectedEndAccount2Balance := startingAccount2Balance + tx1.Value + tx2.Value + database.BlockReward + database.TxFee

		if endAccount1Balance != expectedEndAccount1Balance {
			t.Errorf("account1 expected end balance is %d not %d", expectedEndAccount1Balance, endAccount1Balance)
		}

		if endAccount2Balance != expectedEndAccount2Balance {
			t.Errorf("account2 expected end balance is %d not %d", expectedEndAccount2Balance, endAccount2Balance)
		}

		t.Logf("Starting account1 balance: %d", startingAccount1Balance)
		t.Logf("Starting account2 balance: %d", startingAccount2Balance)
		t.Logf("Ending account1 balance: %d", endAccount1Balance)
		t.Logf("Ending account2 balance: %d", endAccount2Balance)
	}()

	_ = n.Run(ctx, true, "")

	if n.state.LatestBlock().Header.Number != 1 {
		t.Fatal("was suppose to mine 2 pending TX into 2 valid blocks under 30m")
	}

	if len(n.pendingTXs) != 0 {
		t.Fatal("no pending TXs should be left to mine")
	}
}

func TestNode_MiningSpamTransactions(t *testing.T) {
	account1Balance := uint(1000)
	account2Balance := uint(0)
	minerBalance := uint(0)
	minerKey, err := wallet.NewRandomKey()
	if err != nil {
		t.Fatal(err)
	}
	miner := minerKey.Address
	dataDir, account1, account2, err := setupTestNodeDir(account1Balance)
	if err != nil {
		t.Fatal(err)
	}
	defer fs.RemoveDir(dataDir)

	n := New(dataDir, "127.0.0.1", 8085, miner, PeerNode{})
	ctx, closeNode := context.WithCancel(context.Background())
	minerPeerNode := NewPeerNode("127.0.0.1", 8085, false, miner, true)

	txValue := uint(200)
	txCount := uint(4)

	go func() {
		// Wait for the node to run and initialize its state and other components
		time.Sleep(time.Second)

		// Schedule 4 transfers from account1 -> account2
		for i := uint(1); i <= txCount; i++ {
			// Ensure every TX has a unique timestamp
			time.Sleep(time.Second)

			txNonce := i
			tx := database.NewTx(account1, account2, txValue, txNonce, "")

			signedTx, err := wallet.SignTxWithKeystoreAccount(tx, account1, testKsAccountsPwd, wallet.GetKeystoreDirPath(dataDir))
			if err != nil {
				t.Fatal(err)
			}

			_ = n.AddPendingTX(signedTx, minerPeerNode)
		}
	}()

	go func() {
		// Periodically check if we mined the block
		ticker := time.NewTicker(10 * time.Second)

		for {
			select {
			case <-ticker.C:
				if !n.state.LatestBlockHash().IsEmpty() {
					closeNode()
					return
				}
			}
		}
	}()

	// Run the node, mining and everything in a blocking call (hence the go-routines before)
	_ = n.Run(ctx, true, "")

	expectedAccount1Balance := account1Balance - (txCount * txValue) - (txCount * database.TxFee)
	expectedAccount2Balance := account2Balance + (txCount * txValue)
	expectedMinerBalance := minerBalance + database.BlockReward + (txCount * database.TxFee)

	if n.state.Balances[account1] != expectedAccount1Balance {
		t.Errorf("account1 balance is incorrect. Expected: %d. Got: %d", expectedAccount1Balance, n.state.Balances[account1])
	}

	if n.state.Balances[account2] != expectedAccount2Balance {
		t.Errorf("account2 balance is incorrect. Expected: %d. Got: %d", expectedAccount2Balance, n.state.Balances[account2])
	}

	if n.state.Balances[miner] != expectedMinerBalance {
		t.Errorf("Miner balance is incorrect. Expected: %d. Got: %d", expectedMinerBalance, n.state.Balances[miner])
	}

	t.Logf("account1 final balance: %d tokens", n.state.Balances[account1])
	t.Logf("account2 final balance: %d tokens", n.state.Balances[account2])
	t.Logf("Miner final balance: %d tokens", n.state.Balances[miner])
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

// Creates a default testing node directory with 2 keystore accounts
func setupTestNodeDir(account1Balance uint) (dataDir string, account1, account2 common.Address, err error) {
	account2 = database.NewAccount(testKsAccount1)
	account1 = database.NewAccount(testKsAccount2)

	dataDir, err = getTestDataDirPath()
	if err != nil {
		return "", common.Address{}, common.Address{}, err
	}

	genesisBalances := make(map[common.Address]uint)
	genesisBalances[account1] = account1Balance
	genesis := database.Genesis{Balances: genesisBalances}
	genesisJson, err := json.Marshal(genesis)
	if err != nil {
		return "", common.Address{}, common.Address{}, err
	}

	err = database.InitDataDirIfNotExists(dataDir, genesisJson)
	if err != nil {
		return "", common.Address{}, common.Address{}, err
	}

	err = copyKeystoreFilesIntoTestDataDirPath(dataDir)
	if err != nil {
		return "", common.Address{}, common.Address{}, err
	}

	return dataDir, account1, account2, nil
}