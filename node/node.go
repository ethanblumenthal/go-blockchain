package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/ethanblumenthal/golang-blockchain/database"
	"github.com/ethereum/go-ethereum/common"
)

const DefaultBootstrapIp = "127.0.0.1"
const DefaultBootstrapAcc = "0x09ee50f2f37fcba1845de6fe5c762e83e65e755c"
const DefaultMiner = "0x0000000000000000000000000000000000000000"
const DefaultIP = "127.0.0.1"
const HttpSSLPort = 443
const endpointStatus = "/node/status"

const endpointSync = "/node/sync"
const endpointSyncQueryKeyFromBlock = "fromBlock"

const endpointAddPeer = "/node/peer"
const endpointAddPeerQueryKeyIP = "ip"
const endpointAddPeerQueryKeyPort = "port"
const endpointAddPeerQueryKeyMiner = "miner"
const miningIntervalSeconds = 10

type PeerNode struct {
	IP          string         `json:"ip"`
	Port        uint64         `json:"port"`
	IsBootstrap bool           `json:"is_bootstrap"`
	Account     common.Address `json:"account"`
	connected   bool
}

type Node struct {
	dataDir         string
	info            PeerNode
	state           *database.State
	pendingState    *database.State
	knownPeers      map[string]PeerNode
	pendingTXs      map[string]database.SignedTx
	archivedTXs     map[string]database.SignedTx
	newSyncedBlocks chan database.Block
	newPendingTXs   chan database.SignedTx
	isMining        bool
}

func (pn PeerNode) TcpAddress() string {
	return fmt.Sprintf("%s:%d", pn.IP, pn.Port)
}

func (pn PeerNode) ApiProtocol() string {
	if pn.Port == HttpSSLPort {
		return "https"
	}

	return "http"
}

func New(dataDir string, ip string, port uint64, acc common.Address, bootstrap PeerNode) *Node {
	knownPeers := make(map[string]PeerNode)

	n := &Node{
		dataDir:         dataDir,
		info:            NewPeerNode(ip, port, false, acc, true),
		knownPeers:      knownPeers,
		pendingTXs:      make(map[string]database.SignedTx),
		archivedTXs:     make(map[string]database.SignedTx),
		newSyncedBlocks: make(chan database.Block),
		newPendingTXs:   make(chan database.SignedTx, 10000),
		isMining:        false,
	}

	n.AddPeer(bootstrap)
	return n
}

func NewPeerNode(ip string, port uint64, isBootstrap bool, account common.Address, connected bool) PeerNode {
	return PeerNode{ip, port, isBootstrap, account, connected}
}

func (n *Node) Run(ctx context.Context, isSSLDisabled bool, sslEmail string) error {
	fmt.Println(fmt.Sprintf("Listening on: %s:%d", n.info.IP, n.info.Port))

	state, err := database.NewStateFromDisk(n.dataDir)
	if err != nil {
		return err
	}
	defer state.Close()

	n.state = state

	pendingState := state.Copy()
	n.pendingState = &pendingState

	fmt.Println("Blockchain state:")
	fmt.Printf("\t- height: %d\n", n.state.LatestBlock().Header.Number)
	fmt.Printf("\t- hash: %s\n", n.state.LatestBlockHash().Hex())

	go n.sync(ctx)
	go n.mine(ctx)

	return n.serveHttp(ctx, isSSLDisabled, sslEmail)
}

func (n *Node) LatestBlockHash() database.Hash {
	return n.state.LatestBlockHash()
}

func (n *Node) serveHttp(ctx context.Context, isSSLDisabled bool, sslEmail string) error {
	handler := http.NewServeMux()

	handler.HandleFunc("/balances/list", func(w http.ResponseWriter, r *http.Request) {
		listBalancesHandler(w, r, n.state)
	})

	handler.HandleFunc("/tx/add", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Hi")
		txAddHandler(w, r, n)
	})

	handler.HandleFunc(endpointStatus, func(w http.ResponseWriter, r *http.Request) {
		statusHandler(w, r, n)
	})

	handler.HandleFunc(endpointSync, func(w http.ResponseWriter, r *http.Request) {
		syncHandler(w, r, n)
	})

	handler.HandleFunc(endpointAddPeer, func(w http.ResponseWriter, r *http.Request) {
		addPeerHandler(w, r, n)
	})

	if isSSLDisabled {
		server := &http.Server{Addr: fmt.Sprintf(":%d", n.info.Port), Handler: handler}

		go func() {
			<-ctx.Done()
			_ = server.Close()
		}()

		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			return err
		}

		return nil
	} else {
		certmagic.DefaultACME.Email = sslEmail
		return certmagic.HTTPS([]string{n.info.IP}, handler)
	}
}

func (n *Node) mine(ctx context.Context) error {
	var miningCtx context.Context
	var stopCurrentMining context.CancelFunc

	ticker := time.NewTicker(time.Second * miningIntervalSeconds)

	for {
		select {
		case <-ticker.C:
			go func() {
				if len(n.pendingTXs) > 0 && !n.isMining {
					n.isMining = true

					miningCtx, stopCurrentMining = context.WithCancel(ctx)
					err := n.minePendingTXs(miningCtx)
					if err != nil {
						fmt.Printf("ERROR: %s\n", err)
					}

					n.isMining = false
				}
			}()

		case block, _ := <-n.newSyncedBlocks:
			if n.isMining {
				blockHash, _ := block.Hash()
				fmt.Printf("\nPeer mined next Block '%s' faster :(\n", blockHash.Hex())

				n.removeMinedPendingTXs(block)
				stopCurrentMining()
			}

		case <-ctx.Done():
			ticker.Stop()
			return nil
		}
	}
}

func (n *Node) minePendingTXs(ctx context.Context) error {
	blockToMine := NewPendingBlock(
		n.state.LatestBlockHash(),
		n.state.NextBlockNumber(),
		n.info.Account,
		n.getPendingTXsAsArray(),
	)

	minedBlock, err := Mine(ctx, blockToMine)
	if err != nil {
		return err
	}

	n.removeMinedPendingTXs(minedBlock)

	err = n.addBlock(minedBlock)
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) removeMinedPendingTXs(block database.Block) {
	if len(block.TXs) > 0 && len(n.pendingTXs) > 0 {
		fmt.Println("Updating in-memory pending TXs pool:")
	}

	for _, tx := range block.TXs {
		txHash, _ := tx.Hash()
		if _, exists := n.pendingTXs[txHash.Hex()]; exists {
			fmt.Printf("\t-archiving mined TX: %s\n", txHash.Hex())

			n.archivedTXs[txHash.Hex()] = tx
			delete(n.pendingTXs, txHash.Hex())
		}
	}
}

func (n *Node) AddPeer(peer PeerNode) {
	n.knownPeers[peer.TcpAddress()] = peer
}

func (n *Node) RemovePeer(peer PeerNode) {
	delete(n.knownPeers, peer.TcpAddress())
}

func (n *Node) IsKnownPeer(peer PeerNode) bool {
	if peer.IP == n.info.IP && peer.Port == n.info.Port {
		return true
	}

	_, isKnownPeer := n.knownPeers[peer.TcpAddress()]
	return isKnownPeer
}

func (n *Node) AddPendingTX(tx database.SignedTx, fromPeer PeerNode) error {
	txHash, err := tx.Hash()
	if err != nil {
		return err
	}

	txJson, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	err = n.validateTxBeforeAddingToMempool(tx)
	if err != nil {
		return err
	}

	_, isAlreadyPending := n.pendingTXs[txHash.Hex()]
	_, isArchived := n.archivedTXs[txHash.Hex()]

	if !isAlreadyPending && !isArchived {
		fmt.Printf("Added pending TX %s from peer %s\n", txJson, fromPeer.TcpAddress())
		n.pendingTXs[txHash.Hex()] = tx
		n.newPendingTXs <- tx
	}

	return nil
}

func (n *Node) addBlock(block database.Block) error {
	_, err := n.state.AddBlock(block)
	if err != nil {
		return err
	}

	// Reset the pending state
	pendingState := n.state.Copy()
	n.pendingState = &pendingState

	return nil
}

func (n *Node) validateTxBeforeAddingToMempool(tx database.SignedTx) error {
	return database.ApplyTx(tx, n.pendingState)
}

func (n *Node) getPendingTXsAsArray() []database.SignedTx {
	txs := make([]database.SignedTx, len(n.pendingTXs))

	i := 0
	for _, tx := range n.pendingTXs {
		txs[i] = tx
		i++
	}

	return txs
}