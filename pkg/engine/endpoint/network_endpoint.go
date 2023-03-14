package endpoint

import (
	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/consensus"
	"github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
	"github.com/LiskHQ/lisk-engine/pkg/router"
	"github.com/LiskHQ/lisk-engine/pkg/txpool"
)

type networkEndpoint struct {
	config        *config.Config
	chain         *blockchain.Chain
	consensusExec *consensus.Executer
	p2pConn       *p2p.Connection
	txPool        *txpool.TransactionPool
	abi           labi.ABI
}

func NewNetworkEndpoint(
	config *config.Config,
	chain *blockchain.Chain,
	consensusExec *consensus.Executer,
	p2pConn *p2p.Connection,
	txPool *txpool.TransactionPool,
	abi labi.ABI,
) *networkEndpoint {
	return &networkEndpoint{
		config:        config,
		chain:         chain,
		consensusExec: consensusExec,
		p2pConn:       p2pConn,
		txPool:        txPool,
		abi:           abi,
	}
}

func (a *networkEndpoint) Endpoint() router.EndpointHandlers {
	return map[string]router.EndpointHandler{
		"getConnectedPeers": a.HandleGetConnectedPeers,
	}
}

type GetConnectedPeersResponsePeer struct {
	ID string `json:"id" fieldNumber:"1"`
}

type GetConnectedPeersResponse struct {
	Peers []*GetConnectedPeersResponsePeer `json:"peers" fieldNumber:"1"`
}

func (a *networkEndpoint) HandleGetConnectedPeers(w router.EndpointResponseWriter, r *router.EndpointRequest) {
	peers := a.p2pConn.ConnectedPeers()
	resultPeers := make([]*GetConnectedPeersResponsePeer, len(peers))
	for i, peer := range peers {
		resultPeers[i] = &GetConnectedPeersResponsePeer{
			ID: peer.String(),
		}
	}
	resp := &GetConnectedPeersResponse{
		Peers: resultPeers,
	}
	w.Write(resp)
}
