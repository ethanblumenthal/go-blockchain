package node

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethanblumenthal/golang-blockchain/database"
	"github.com/ethereum/go-ethereum/common"
)

type PendingBlock struct {
	parent database.Hash
	number uint64
	time   uint64
	miner  common.Address
	txs    []database.SignedTx
}

func NewPendingBlock(parent database.Hash, number uint64, miner common.Address, txs []database.SignedTx) PendingBlock {
	return PendingBlock{parent, number, uint64(time.Now().Unix()), miner, txs}
}

func Mine(ctx context.Context, pb PendingBlock) (database.Block, error) {
	// Skip empty blocks
	if len(pb.txs) == 0 {
		err := fmt.Errorf("mining empty blocks is not allowed")
		return database.Block{}, err
	}

	// Prepare necessary values
	start := time.Now()
	attempt := 0
	var block database.Block
	var hash database.Hash
	var nonce uint32

	// Repeat the hash generation process until a valid hash is found
	for !database.IsBlockHashValid(hash) {
		select {
			// Close the program if the mining process is stopped
			case <- ctx.Done():
				fmt.Println("Mining cancelled!")
				err := fmt.Errorf("mining cancelled. %s", ctx.Err())
				return database.Block{}, err
			default:
		}

		// Generate a big random number
		attempt++
		nonce = generateNonce()

		// Print update every 1 million attempts
		if attempt == 0 || attempt == 1 {
			fmt.Printf("Mining %d pending TXs. Attempt: %d\n", len(pb.txs), attempt)
		}

		// Try to construct block with random nonce
		block = database.NewBlock(
			pb.parent,
			pb.number,
			nonce,
			pb.time,
			pb.miner,
			pb.txs,
		)

		// Hash block and hope for the best!
		blockHash, err := block.Hash()
		if err != nil {
			err = fmt.Errorf("couldn't mine block. %s", err.Error())
			return database.Block{}, err
		}

		// Break loop if hash is valid
		hash = blockHash
	}

	fmt.Printf("\nMined new Block '%x' using PoW:\n", hash)
    fmt.Printf("\tHeight: '%v'\n", block.Header.Number)
    fmt.Printf("\tNonce: '%v'\n", block.Header.Nonce)
    fmt.Printf("\tCreated: '%v'\n", block.Header.Time)
    fmt.Printf("\tMiner: '%v'\n", block.Header.Miner.String())
    fmt.Printf("\tParent: '%v'\n\n", block.Header.Parent.Hex())
    fmt.Printf("\tAttempt: '%v'\n", attempt)
    fmt.Printf("\tTime: %s\n\n", time.Since(start))

	return block, nil
}

func generateNonce() uint32 {
	rand.Seed(time.Now().UTC().UnixNano())
	return rand.Uint32()
}