package node

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"testing"
	"time"

	"github.com/ethanblumenthal/golang-blockchain/database"
	"github.com/ethanblumenthal/golang-blockchain/wallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestValidBlockHash(t *testing.T) {
	// Creates a random hex string starting with 6 zeroes
	hexHash := "000000fa04f8160395c387277f8b2f14837603383d33809a4db586086168edfa"
	var hash = database.Hash{}

	// Convert it to raw bytes
	hex.Decode(hash[:], []byte(hexHash))

	// Validate the hash
	isValid := database.IsBlockHashValid(hash)
	if !isValid {
		t.Fatalf("hash '%s' with 6 zeroes should be valid", hexHash)
	}
}

func TestInvalidBlockHash(t *testing.T) {
	hexHash := "000001fa04f8160395c387277f8b2f14837603383d33809a4db586086168edfa"
	var hash = database.Hash{}

	hex.Decode(hash[:], []byte(hexHash))

	isValid := database.IsBlockHashValid(hash)
	if isValid {
		t.Fatal("hash is not suppose to be valid")
	}
}
func TestMine(t *testing.T) {
	minerPrivKey, _, miner, err := generateKey()
	if err != nil {
		t.Fatal(err)
	}

	pendingBlock, err := createRandomPendingBlock(minerPrivKey, miner)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	minedBlock, err := Mine(ctx, pendingBlock)
	if err != nil {
		t.Fatal(err)
	}

	minedBlockHash, err := minedBlock.Hash()
	if err != nil {
		t.Fatal(err)
	}

	if !database.IsBlockHashValid(minedBlockHash) {
		t.Fatal()
	}

	if minedBlock.Header.Miner.String() != miner.String() {
		t.Fatal("mined block miner should equal miner from pending block")
	}
}

func TestMineWithTimeout(t *testing.T) {
	minerPrivKey, _, miner, err := generateKey()
	if err != nil {
		t.Fatal(err)
	}

	pendingBlock, err := createRandomPendingBlock(minerPrivKey, miner)
	if err != nil {
		t.Fatal(err)
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Microsecond*100)

	_, err = Mine(ctx, pendingBlock)
	if err == nil {
		t.Fatal(err)
	}
}

func generateKey() (*ecdsa.PrivateKey, ecdsa.PublicKey, common.Address, error) {
	privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return nil, ecdsa.PublicKey{}, common.Address{}, err
	}

	pubKey := privKey.PublicKey
	pubKeyBytes := elliptic.Marshal(crypto.S256(), pubKey.X, pubKey.Y)
	pubKeyBytesHash := crypto.Keccak256(pubKeyBytes[1:])
	account := common.BytesToAddress(pubKeyBytesHash[12:])

	return privKey, pubKey, account, nil
}

func createRandomPendingBlock(privKey *ecdsa.PrivateKey, account common.Address) (PendingBlock, error) {
	tx := database.NewTx(account, database.NewAccount(testKsAccount2), 1, "")
	signedTx, err := wallet.SignTx(tx, privKey)
	if err != nil {
		return PendingBlock{}, err
	}

	return NewPendingBlock(database.Hash{}, 0, account, []database.SignedTx{signedTx}), err
}