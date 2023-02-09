package pubsub

import (
	"context"
	"testing"

	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
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

func TestMessageTopic(t *testing.T) {
	assert.Equal(t, MessageTopic("testTopic"), "/lsk/testTopic")
}

func TestHashMsgID(t *testing.T) {
	assert := assert.New(t)

	msg := pubsub_pb.Message{}
	hash := HashMsgID(&msg)
	expected := codec.Hex(crypto.Hash([]byte{})).String()
	assert.Equal(expected, hash)
}

func TestUtils_ExtractIP(t *testing.T) {
	assert := assert.New(t)

	var tests = []struct {
		name      string
		multiAddr string
		want      string
	}{
		{"ip4", "/ip4/127.0.0.1/udp/1234", "127.0.0.1"},
		{"ip6", "/ip6/2001:0db8:85a3:0000:0000:8a2e:0370:7334/p2p/12D3KooWGQTQuV6JfgpKpc847NMsxaFnKXDEpkN5kbeH1SDFW34S", "2001:db8:85a3::8a2e:370:7334"},
		{"ip4", "/ip4/172.20.10.6/tcp/49526/p2p/12D3KooWB4J4mraN1nAB9Ge5w8JrzGRKDQ3iGyDyHZdbMWDtYg3R", "172.20.10.6"},
		{"reley", "/ip4/198.51.100.0/tcp/55555/p2p/12D3KooWGQTQuV6JfgpKpc847NMsxaFnKXDEpkN5kbeH1SDFW34S/p2p-circuit/p2p/12D3KooWGQTQuV6JfgpKpc847NMsxaFnKXDEpkN5kbeH1SDFW34S", "198.51.100.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := ma.NewMultiaddr(tt.multiAddr)
			assert.Nil(err)

			got := ExtractIP(addr)
			assert.Equal(tt.want, got)
		})
	}
}
