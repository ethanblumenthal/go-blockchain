package database

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type State struct {
	Balances        map[Account]uint
	txMempool       []Tx
	dbFile          *os.File
	latestBlock     Block
	latestBlockHash Hash
}

func NewStateFromDisk(dataDir string) (*State, error) {
	err := initDataDirIfNotExists(dataDir)
	if err != nil {
		return nil, err
	}

	gen, err := loadGenesis(getGenesisJsonFilePath(dataDir))
	if err != nil {
		return nil, err
	}

	balances := make(map[Account]uint)
	for account, balance := range gen.Balances {
		balances[account] = balance
	}

	blockDbFilePath := filepath.Join(dataDir, "database", "block.db")
	f, err := os.OpenFile(blockDbFilePath, os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(f)
	state := &State{balances, make([]Tx, 0), f, Block{}, Hash{}}

	// Iterate over each the tx.db file's line
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		blockFSJson := scanner.Bytes()

		// Convert JSON encoded TX into an object (struct)
		var blockFs BlockFS
		json.Unmarshal(blockFSJson, &blockFs)
		if err != nil {
			return nil, err
		}

		// Rebuild the state (user balances),
		// as a series of events
		err := state.applyBlock(blockFs.Value)
		if err != nil {
			return nil, err
		}
		
		state.latestBlock = blockFs.Value
		state.latestBlockHash = blockFs.Key
	}
	return state, nil
}

func (s *State) LatestBlock() Block {
	return s.latestBlock
}

func (s *State) LatestBlockHash() Hash {
	return s.latestBlockHash
}

func (s *State) AddBlock(b Block) error {
	for _, tx := range b.TXs {
		if err := s.AddTx(tx); err != nil {
			return err
		}
	}

	return nil
}

func (s *State) AddTx(tx Tx) error {
	if err := s.apply(tx); err != nil {
		return err
	}

	s.txMempool = append(s.txMempool, tx)

	return nil
}

func (s *State) Persist() (Hash, error) {
	// Creates a new block with ONLY the new TXs
	latestBlockHash, err := s.latestBlock.Hash()
	if err != nil {
		return Hash{}, err
	}

	block := NewBlock(
		latestBlockHash,
		s.latestBlock.Header.Number + 1,
		uint64(time.Now().Unix()),
		s.txMempool,
	)
	
	blockHash, err := block.Hash()
	if err != nil {
		return Hash{}, err
	}

	blockFs := BlockFS{blockHash, block}

	// Encode it into a JSON string
	blockFsJson, err := json.Marshal(blockFs)
	if err != nil {
		return Hash{}, err
	}

	fmt.Printf("Persisting new block to disk:\n")
	fmt.Printf("\t%s\n", blockFsJson)

	// Write it to the DB file on a new line
	if _, err = s.dbFile.Write(append(blockFsJson, '\n')); err != nil {
		return Hash{}, err
	}
	s.latestBlockHash = blockHash

	// Reset the mempool
	s.latestBlock = block
	s.latestBlockHash = latestBlockHash
	s.txMempool = []Tx{}

	return latestBlockHash, nil
}

func (s *State) Close() {
	s.dbFile.Close()
}

func (s *State) applyBlock(b Block) error {
	for _, tx := range b.TXs {
		if err := s.apply(tx); err != nil {
			return err
		}
	}

	return nil
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
