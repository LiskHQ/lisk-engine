package socket

// DisconnectCode defines the closing code for socket.
type DisconnectCode int

const (
	CloseIntentional            DisconnectCode = 1000
	CloseSeedPeer               DisconnectCode = 4000
	CloseInvalidSelfConnection  DisconnectCode = 4101
	CloseInvalidChainID         DisconnectCode = 4102
	CloseInvalidVersionCode     DisconnectCode = 4103
	CloseInvalidPeer            DisconnectCode = 4104
	CloseInvalidPeerInfo        DisconnectCode = 4105
	CloseForbidden              DisconnectCode = 4403
	CloseDuplicate              DisconnectCode = 4404
	CloseEvicted                DisconnectCode = 4418
	CloseInvalidConnectionURL   DisconnectCode = 4501
	CloseInvalidConnectionQuery DisconnectCode = 4502
)

// CloseCodes defines various socket close code and reason.
var closeCodes = map[DisconnectCode]string{
	CloseIntentional:            "Closing connection",
	CloseSeedPeer:               "Disconnect from SeedPeer after discovery",
	CloseInvalidSelfConnection:  "Peer cannot connect to itself",
	CloseInvalidChainID:         "Peer chainID did not match our own",
	CloseInvalidVersionCode:     "Peer has incompatible protocol version",
	CloseInvalidPeer:            "Peer is incompatible with the node for unknown reasons",
	CloseInvalidPeerInfo:        "Peer has invalid PeerInfo",
	CloseForbidden:              "Peer is not allowed to connect",
	CloseDuplicate:              "Peer has a duplicate connection",
	CloseEvicted:                "Peer has been evicted per protocol",
	CloseInvalidConnectionURL:   "Peer did not provide a valid URL as part of the WebSocket connection",
	CloseInvalidConnectionQuery: "Peer did not provide valid query parameters as part of the WebSocket connection",
}
