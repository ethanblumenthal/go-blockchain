package node

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethanblumenthal/golang-blockchain/database"
	"github.com/ethanblumenthal/golang-blockchain/fs"
)

func getTestDataDirPath() string {
	return filepath.Join(os.TempDir(), ".gochain_test")
}

func TestNode_Run(t *testing.T) {
	// Remove the test directory if it already exists
	datadir := getTestDataDirPath()
	err := fs.RemoveDir(datadir)
	if err != nil {
		t.Fatal(err)
	}

	// Construct a new node instance
	n := New(datadir, "127.0.0.1", 8085, database.NewAccount("ethan"), PeerNode{})

	// Define a context with timeout for this test
	// Node will run for 5s
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	err = n.Run(ctx)
	if err.Error() != "http: Server closed" {
		// Assert expected behavior
		t.Fatal("node server was supposed to close after 5s")
	}
}