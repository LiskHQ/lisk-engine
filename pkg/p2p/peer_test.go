package p2p

// func TestPeerCleanup(t *testing.T) {
// 	address := addressbook.NewAddress("127.0.0.1", 5001)
// 	nodeInfo := &NodeInfo{
// 		Port:              4000,
// 		ChainID: "00000000",
// 		NetworkVersion:    "2.0",
// 		Nonce:             "O2wTkjqplHIIabab",
// 	}
// 	ctx := context.Background()
// 	connector := createConnector(ctx, address, nodeInfo)
// 	conn, err := connector()
// 	assert.Nil(t, err)
// 	peer, err := newOutboundPeer(ctx, address, conn, connector, nil, log.DefaultLogger)
// 	assert.Nil(t, err)
// 	go peer.bind()

// 	<-time.After(10 * time.Second)
// 	peer.close()
// 	goleak.VerifyNone(t)
// }
