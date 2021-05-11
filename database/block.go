package database

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

const BlockReward = 100

type Hash [32]byte

func (h *Hash) UnmarshalText(data []byte) error {
	_, err := hex.Decode(h[:], data)
	return err
}

func (h Hash) Hex() string {
	return hex.EncodeToString(h[:])
}

type Block struct {
	Header BlockHeader `json:"header"`
	TXs    []SignedTx  `json:"payload"`
}

type BlockHeader struct {
	Parent Hash           `json:"parent"`
	Number uint64         `json:"number"`
	Nonce  uint32         `json:"nonce"`
	Time   uint64         `json:"time"`
	Miner  common.Address `json:"miner"`
}

type BlockFS struct {
	Key    Hash        `json:"hash"`
	Value  Block       `json:"block"`
}

func NewBlock(parent Hash, number uint64, nonce uint32, time uint64, miner common.Address, txs []SignedTx) Block {
	return Block{BlockHeader{parent, number, nonce, time, miner}, txs}
}

func IsBlockHashValid(h Hash) bool {
	return fmt.Sprintf("%x", h[0]) == "0" &&
	fmt.Sprintf("%x", h[1]) == "0" &&
	fmt.Sprintf("%x", h[2]) == "0" &&
	fmt.Sprintf("%x", h[3]) != "0"
}


func (h Hash) IsEmpty() bool {
	emptyHash := Hash{}
	return bytes.Equal(emptyHash[:], h[:])
}

func (b Block) Hash() (Hash, error) {
	blockJson, err := json.Marshal(b)
	if err != nil {
		return Hash{}, err
	}

	return sha256.Sum256(blockJson), nil
}