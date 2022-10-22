package sync

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen
type NodeInfo struct {
	height            uint32 `fieldNumber:"1"`
	maxHeightPrevoted uint32 `fieldNumber:"2"`
	blockVersion      uint32 `fieldNumber:"3"`
	lastBlockID       []byte `fieldNumber:"4"`
	PeerID            string
}

func NewNodeInfo(
	height uint32,
	maxHeightPrevoted uint32,
	blockVersion uint32,
	lastBlockID []byte,
) *NodeInfo {
	return &NodeInfo{
		height:            height,
		maxHeightPrevoted: maxHeightPrevoted,
		blockVersion:      blockVersion,
		lastBlockID:       lastBlockID,
	}
}
