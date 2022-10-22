package smt

func treeHasher(nodeHashes [][]byte, structure []uint8, height uint8) []byte {
	if len(nodeHashes) == 1 {
		return nodeHashes[0]
	}
	nextHashes := [][]byte{}
	nextStructure := []uint8{}
	i := 0
	for i < len(nodeHashes) {
		if structure[i] != height {
			nextHashes = append(nextHashes, nodeHashes[i])
			nextStructure = append(nextStructure, structure[i])
			i += 1
			continue
		}
		node := newBranchNode(nodeHashes[i], nodeHashes[i+1])
		nextHashes = append(nextHashes, node.hash)
		nextStructure = append(nextStructure, structure[i]-1)
		i += 2
	}
	if height == 1 {
		return nextHashes[0]
	}
	return treeHasher(nextHashes, nextStructure, height-1)
}
