package database

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type State struct {
	Balances map[Account]uint
	txMempool []Tx

	dbFile *os.File
	snapshot Snapshot
}

type Snapshot [32]byte

func NewStateFromDisk() (*State, error) {
	// get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	genFilePath := filepath.Join(cwd, "database", "genesis.json")
	gen, err := loadGenesis(genFilePath)
	if err != nil {
		return nil, err
	}

	balances := make(map[Account]uint)
	for account, balance := range gen.Balances {
		balances[account] = balance
	}

	txDbFilePath := filepath.Join(cwd, "database", "tx.db")
	f, err := os.OpenFile(txDbFilePath, os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(f)
	state := &State{balances, make([]Tx, 0), f, Snapshot{}}

	// Iterate over each the tx.db file's line
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		// Convert JSON encoded TX into an object (struct)
		var tx Tx
		json.Unmarshal(scanner.Bytes(), &tx)

		// Rebuild the state (user balances),
		// as a series of events
		if err := state.apply(tx); err != nil {
			return nil, err
		}
	}
	return state, nil
}

func (s *State) Add(tx Tx) error {
	if err := s.apply(tx); err != nil {
		return err
	}

	s.txMempool = append(s.txMempool, tx)

	return nil
}

func (s *State) Persist() (Snapshot, error) {
	// Make a copy of mempool beacuse the s.txMempool will be modified
	// in the loop below
	mempool := make([]Tx, len(s.txMempool))
	copy(mempool, s.txMempool)

	for i := 0; i < len(mempool); i++ {
		txJson, err := json.Marshal(mempool[i])
		if err != nil {
			return Snapshot{}, err
		}

		fmt.Printf("Persisting new TX to disk:\n")
		fmt.Printf("\t%s\n", txJson)
		if _, err = s.dbFile.Write(append(txJson, '\n')); err != nil {
			return Snapshot{}, err
		}

		err = s.doSnapshot()
		if err != nil {
			return Snapshot{}, err
		}
		fmt.Printf("New DB Snapshot: %x\n", s.snapshot)

		// Remove the TX written to a file from the mempool
		s.txMempool = s.txMempool[1:]
	}

	return s.snapshot, nil
}

func (s *State) Close() {
	s.dbFile.Close()
}

func (s *State) apply(tx Tx) error {
	if tx.IsReward() {
		s.Balances[tx.To] += tx.Value
		return nil
	}

	if tx.Value > s.Balances[tx.From] {
		return fmt.Errorf("insufficient balance")
	}

	s.Balances[tx.From] -= tx.Value
	s.Balances[tx.To] += tx.Value

	return nil
}

func (s *State) doSnapshot() error {
	// Re-read the whole file from the first byte
	_, err := s.dbFile.Seek(0, 0)
	if err != nil {
		return err
	}

	txsData, err := ioutil.ReadAll(s.dbFile)
	if err != nil {
		return err
	}
	s.snapshot = sha256.Sum256(txsData)
	return nil
}