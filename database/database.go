package database

import (
	"bufio"
	"encoding/json"
	"os"
)

func GetBlocksAfter(blockHash Hash, dataDir string) ([]Block, error) {
	// Open the database
	f, err := os.OpenFile(getBlocksDbFilePath(dataDir), os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}

	blocks := make([]Block, 0)
	shouldStartCollecting := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Read the database file one line at a time
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		var blockFs BlockFS
		err = json.Unmarshal(scanner.Bytes(), &blockFs)
		if err != nil {
			return nil, err
		}

		if shouldStartCollecting {
			blocks = append(blocks, blockFs.Value)
			continue
		}

		// Collect new blocks when block hash found
		if blockHash == blockFs.Key {
			shouldStartCollecting = true
		}
	}
	return blocks, nil
}