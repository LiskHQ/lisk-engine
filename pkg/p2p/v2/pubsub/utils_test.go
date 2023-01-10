package pubsub

import (
	"context"
	"fmt"
	"testing"

	"github.com/libp2p/go-libp2p"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

func TestParseAddresses(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// empty array
	stringAddresses := []string{}
	_, err := ParseAddresses(ctx, stringAddresses)
	assert.Nilf(err, "empty array should not return an error")

	// empty addresses
	stringAddresses = []string{"", ""}
	_, err = ParseAddresses(ctx, stringAddresses)
	assert.Errorf(err, "empty string as an address should return an error")

	// valid address
	stringAddresses = []string{
		"/ip4/7.7.7.7/tcp/4242/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N",
		"/ip4/172.20.10.6/tcp/49526/p2p/12D3KooWB4J4mraN1nAB9Ge5w8JrzGRKDQ3iGyDyHZdbMWDtYg3R",
		"/ip4/172.20.10.6/udp/49529/p2p/12D3KooWGQTQuV6JfgpKpc847NMsxaFnKXDEpkN5kbeH1REW41BR",
		"/ip6/2001:0db8:85a3:0000:0000:8a2e:0370:7334/p2p/12D3KooWGQTQuV6JfgpKpc847NMsxaFnKXDEpkN5kbeH1SDFW34S",
	}
	_, err = ParseAddresses(ctx, stringAddresses)
	assert.Nilf(err, "valid address should parse successfully")

	// invalid address
	stringAddresses = []string{
		"/ip4/127.0.0.1/udp/1234",
		"/ip4/7.7.7.7/tcp/4242/lsk/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N",
		"/ip4/7.7.7.7/lsk/4242/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N",
		"/ip4/7.7.7.7/tcp/4242/p2p/QmRelay/p2p-circuit/p2p/QmRelayedPeer",
		"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
	}
	_, err = ParseAddresses(ctx, stringAddresses)
	assert.Errorf(err, "invalid address are not acceptable")
}

func TestP2pAddrs(t *testing.T) {
	assert := assert.New(t)

	ip4quic := fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", 12345)
	ip4tcp := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 12345)
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(ip4quic, ip4tcp))
	assert.Nil(err)

	addrs, err := P2pAddrs(h)
	assert.Nil(err)
	assert.Equal(len(addrs), 4)
}

func TestMessageTopic(t *testing.T) {
	assert.Equal(t, MessageTopic("testTopic"), "/lsk/testTopic")
}

func TestHashMsgID(t *testing.T) {
	msg := pubsub_pb.Message{}
	hash := HashMsgID(&msg)
	expected := string(crypto.Hash([]byte{}))
	assert.Equal(t, expected, hash)
}
