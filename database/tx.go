package database

import (
	"crypto/sha256"
	"encoding/json"
)

type Account string

func NewAccount(value string) Account {
	return Account(value)
}

type Tx struct {
	From  Account `json:"from"`
	To    Account `json:"to"`
	Value uint    `json:"value"`
	Data  string  `json:"data"`
}

func NewTx(from Account, to Account, value uint, data string) Tx {
	return Tx{from, to, value, data}
}

func (t Tx) IsReward() bool {
	return t.Data == "reward"
}

func (t Tx) Hash() (Hash, error) {
	blockJson, err := json.Marshal(t)
	if err != nil {
		return Hash{}, err
	}

	return sha256.Sum256(blockJson), nil
}