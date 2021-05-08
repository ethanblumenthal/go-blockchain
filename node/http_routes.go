package node

import (
	"net/http"

	"github.com/ethanblumenthal/golang-blockchain/database"
)

type ErrRes struct {
	Error string `json:"error"`
}

type BalancesRes struct {
	Hash      database.Hash             `json:"block_hash"`
	Balances  map[database.Account]uint `json:"balances"`
}

type TxAddReq struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Value uint   `json:"value"`
	Data  string `json:"data"`
}

type TxAddRes struct {
	Hash database.Hash `json:"block_hash"`
}

type StatusRes struct {
	Hash       database.Hash          `json:"block_hash"`
	Number     uint64                 `json:"block_number"`
	KnownPeers map[string]PeerNode    `json:"peers_known"`
}

type SyncRes struct {
	Blocks []database.Block `json:"blocks"`
}

func listBalancesHandler(w http.ResponseWriter, r *http.Request, state *database.State) {
	writeRes(w, BalancesRes{state.LatestBlockHash(), state.Balances})
}

func txAddHandler(w http.ResponseWriter, r *http.Request, state *database.State) {
	req := TxAddReq{}

	// Parse the POST request body
	err := readReq(r, &req)
	if err != nil {
		writeErrRes(w, err)
		return
	}


	tx := database.NewTx(
		database.NewAccount(req.From),
		database.NewAccount(req.To),
		req.Value,
		req.Data,
	)

	// Add a new TX into the Mempool
	err = state.AddTx(tx)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	// Flush the Mempool TX to the disk
	hash, err := state.Persist()
	if err != nil {
		writeErrRes(w, err)
		return
	}

	writeRes(w, TxAddRes{hash})
}

func statusHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	res := StatusRes{
		Hash: node.state.LatestBlockHash(),
		Number: node.state.LatestBlock().Header.Number,
		KnownPeers: node.knownPeers,
	}

	writeRes(w, res)
}

func syncHandler(w http.ResponseWriter, r *http.Request, dataDir string) {
	// Query latest block and check state for newer blocks
	reqHash := r.URL.Query().Get(endpointSyncQueryKeyFromBlock)

	hash := database.Hash{}
	err := hash.UnmarshaText([]byte(reqHash))
	if err != nil {
		writeErrRes(w, err)
		return
	}

	// Read newer blocks from the database
	blocks, err := database.GetBlocksAfter(hash, dataDir)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	// JSON encode the blocks and return them in the response
	writeRes(w, SyncRes{Blocks: blocks})
}