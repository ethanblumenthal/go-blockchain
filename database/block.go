package database

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type Hash [32]byte

func (h *Hash) UnmarshaText(data []byte) error {
	_, err := hex.Decode(h[:], data)
	return err
}

type Block struct {
	Header BlockHeader `json:"header"`
	TXs    []Tx        `json:"payload"`
}

type BlockHeader struct {
	Parent Hash        `json:"parent"`
	Number uint64      `json:"number"`
	Time   uint64      `json:"time"`
}

type BlockFS struct {
	Key    Hash        `json:"hash"`
	Value  Block       `json:"block"`
}

func NewBlock(parent Hash, number uint64, time uint64, txs []Tx) Block {
	return Block{BlockHeader{parent, number, time}, txs}
}

func (b Block) Hash() (Hash, error) {
	blockJson, err := json.Marshal(b)
	if err != nil {
		return Hash{}, err
	}

	return sha256.Sum256(blockJson), nil
}