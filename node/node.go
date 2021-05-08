package node

import (
	"fmt"
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