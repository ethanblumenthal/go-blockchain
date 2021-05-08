package node

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ethanblumenthal/golang-blockchain/database"
)

const DefaultHTTPPort = 8080

type PeerNode struct {
	IP          string `json:"ip"`
	Port        uint64 `json:"port"`
	IsBootStrap bool   `json:"is_bootstrap"`
	IsActive    bool   `json:"is_active"`
}

type Node struct {
	dataDir string
	port uint64

	// To inject the State into HTTP handlers
	state *database.State
	knownPeers []PeerNode
}

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
	Hash       database.Hash `json:"block_hash"`
	Number     uint64        `json:"block_number"`
	KnownPeers []PeerNode    `json:"peers_known"`
}

func NewPeerNode(ip string, port uint64, isBootStrap bool, isActive bool) PeerNode {
	return NewPeerNode(ip, port, isBootStrap, isActive)
}

func (n *Node) Run() error {
	fmt.Println(fmt.Sprintf("Listening on HTTP port %d", n.port))

	state, err := database.NewStateFromDisk(n.dataDir)
	if err != nil {
		return err
	}
	defer state.Close()

	n.state = state

	http.HandleFunc("/balances/list", func (w http.ResponseWriter, r *http.Request) {
		listBalancesHandler(w, r, state)	
	})

	http.HandleFunc("/tx/add", func(w http.ResponseWriter, r *http.Request) {
		txAddHandler(w, r, state)
	})

	http.HandleFunc("/node/status", func(w http.ResponseWriter, r *http.Request) {
		statusHandler(w, r, n)
	})

	return http.ListenAndServe(fmt.Sprintf(":%d", n.port), nil)
}

func New(dataDir string, port uint64, bootstrap PeerNode) *Node {
	return &Node{
		dataDir: dataDir,
		port: port,
		knownPeers: []PeerNode{bootstrap},
	}
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

func writeErrRes(w http.ResponseWriter, err error) {
	jsonErrRes, _ := json.Marshal(ErrRes{err.Error()})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(jsonErrRes)
}

func writeRes(w http.ResponseWriter, content interface{}) {
	contentJson, err := json.Marshal(content)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(contentJson)
}

func readReq(r *http.Request, reqBody interface{}) error {
	reqBodyJson, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("unable to read request body. %s", err.Error())
	}
	defer r.Body.Close()

	err = json.Unmarshal(reqBodyJson, reqBody)
	if err != nil {
		return fmt.Errorf("unable to unmarshal request body. %s", err.Error())
	}

	return nil
}