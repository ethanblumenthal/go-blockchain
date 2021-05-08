package node

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ethanblumenthal/golang-blockchain/database"
)

func (n *Node) sync(ctx context.Context) error {
	ticker := time.NewTicker(45 * time.Second)

	for {
		select {
		case <-ticker.C:
			n.doSync()

		case <-ctx.Done():
			ticker.Stop()
		}
	}
}

func (n *Node) doSync() {
	for _, peer := range n.knownPeers {
		status, err := queryPeerStatus(peer)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}

		err = n.joinKnownPeers(peer)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}

		err = n.syncBlocks(peer, status)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}

		err = n.syncKnownPeers(peer, status)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}
	}
}

func (n *Node) syncBlocks(peer PeerNode, status StatusRes) error {
	localBlockNumber := n.state.LatestBlock().Header.Number
	if localBlockNumber < status.Number {
		newBlocksCount := status.Number - localBlockNumber
		fmt.Printf("Found %d new blocks from peer %s\n", newBlocksCount, peer.TcpAddress())

		// Call the node's /node/sync endpoint and read new blocks
		blocks, err := fetchBlocksFromPeer(peer, n.state.LatestBlockHash())
		if err != nil {
			return err
		}

		// Write the newly downloaded blocks to this node's local database
		err = n.state.AddBlocks(blocks)
		if err != nil {
			return err
		}
	}
	
	return nil
}

func (n *Node) syncKnownPeers(peer PeerNode, status StatusRes) error {
	for _, statusPeer := range status.KnownPeers {
		if !n.IsKnownPeer(statusPeer) {
			fmt.Printf("Found new peer %s\n", statusPeer.TcpAddress())
			n.AddPeer(statusPeer)
		}
	}

	return nil
}

func (n *Node) joinKnownPeers(peer PeerNode) error {}

func queryPeerStatus(peer PeerNode) (StatusRes, error) {
	url := fmt.Sprintf("http://%s%s", peer.TcpAddress(), endpointStatus)
	res, err := http.Get(url)
	if err != nil {
		return StatusRes{}, err
	}

	statusRes := StatusRes{}
	err = readRes(res, &statusRes)
	if err != nil {
		return StatusRes{}, err
	}
	
	return statusRes, nil
}

func fetchBlocksFromPeer(peer PeerNode, fromBlock database.Hash) ([]database.Block, error) {}