package database

import "os"

type State struct {
	Balances map[Account]uint
	txMempool []Tx

	dbFile *os.File
}