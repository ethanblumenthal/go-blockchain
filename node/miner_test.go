package node

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/ethanblumenthal/golang-blockchain/database"
)

func TestValidBlockHash(t *testing.T) {
	// Creates a random hex string starting with 6 zeroes
	hexHash := "000000fa04f816039...a4db586086168edfa"
	var hash = database.Hash{}

	// Convert it to raw bytes
	hex.Decode(hash[:], []byte(hexHash))

	// Validate the hash
	isValid := database.IsBlockHashValid(hash)
	if !isValid {
		t.Fatalf("hash '%s' with 6 zeroes should be valid", hexHash)
	}
}

func TestMine(t *testing.T) {
	miner := database.NewAccount("ethan")
	pendingBlock := createRandomPendingBlock(miner)

	// Contes
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
}

func createRandomPendingBlock(miner database.Account) PendingBlock {
	return NewPendingBlock(
		database.Hash{},
		0,
		miner,
		[]database.Tx{
			database.NewTx("ethan", "ethan", 3, ""),
        	database.NewTx("ethan", "ethan", 700, "reward"),
		},
	)
}