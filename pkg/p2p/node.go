package p2p

import "fmt"

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

// NodeInfo contains the information about the host node.
type NodeInfo struct {
	ChainID          string `json:"chainID" fieldNumber:"1"`
	NetworkVersion   string `json:"networkVersion"  fieldNumber:"2"`
	Nonce            string `json:"nonce"  fieldNumber:"3"`
	AdvertiseAddress bool   `json:"-"  fieldNumber:"4"`
	Port             int    `json:"port"`
	Options          []byte `json:"-" fieldNumber:"5"`
}

// Query returns URL query string.
func (n NodeInfo) Query() string {
	return fmt.Sprintf("chainID=%s&networkVersion=%s&nonce=%s&port=%d", n.ChainID, n.NetworkVersion, n.Nonce, n.Port)
}
