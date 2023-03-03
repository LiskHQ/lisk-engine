package p2p

import (
	"errors"
	"testing"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

func TestAddr_PeerInfoFromMultiAddrs(t *testing.T) {
	assert := assert.New(t)

	type Want struct {
		id   string
		addr string
		err  error
	}

	var tests = []struct {
		name string
		addr string
		want Want
	}{
		{"correct ipv4 address with tcp", "/ip4/162.11.231.222/tcp/39781/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXL", Want{
			"12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXL",
			"/ip4/162.11.231.222/tcp/39781", nil},
		},
		{"correct ipv4 address with udp", "/ip4/162.11.231.222/udp/9781/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXL", Want{
			"12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXL",
			"/ip4/162.11.231.222/udp/9781", nil},
		},
		{"correct ipv6 address with tcp", "/ip6/2001:0db8:85a3:0000:0000:8a2e:0370:7334/tcp/39781/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXL", Want{
			"12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXL",
			"/ip6/2001:db8:85a3::8a2e:370:7334/tcp/39781", nil},
		},
		{"correct ipv6 address with udp", "/ip6/::1/udp/781/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXL", Want{
			"12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXL",
			"/ip6/::1/udp/781", nil},
		},
		{"incorrect address ip4 is missing", "/162.11.231.222/tcp/39781/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXL", Want{
			"",
			"", errors.New("unknown protocol 162.11.231.222")},
		},
		{"incorrect address IP number is wrong", "/ip4/162.11.231/tcp/39781/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXL", Want{
			"",
			"", errors.New("failed to parse ip4 addr")},
		},
		{"incorrect address port is missing", "/ip6/::1/tcp/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXL", Want{
			"",
			"", errors.New("failed to parse port addr")},
		},
		{"incorrect address p2p address is wrong", "/ip4/162.11.231.222/tcp/39781/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4oD8jJdXL", Want{
			"",
			"", errors.New("failed to parse p2p addr")},
		},
		{"incorrect address p2p address is missing", "/ip6/2001:0db8:85a3:0000:0000:8a2e:0370:7334/tcp/39781/p2p/", Want{
			"",
			"", errors.New("unexpected end of multiaddr")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peer, err := AddrInfoFromMultiAddr(tt.addr)

			if tt.want.err == nil {
				assert.Nil(err)
			} else {
				assert.Contains(err.Error(), tt.want.err.Error())
				return // if error occurs skip the rest of body because it must not be checked
			}

			assert.Equal(tt.want.id, peer.ID.String())
			assert.Equal(tt.want.addr, peer.Addrs[0].String())
		})
	}
}

func TestParseAddresses(t *testing.T) {
	assert := assert.New(t)

	// empty array
	stringAddresses := []string{}
	_, err := parseAddresses(stringAddresses)
	assert.Nilf(err, "empty array should not return an error")

	// empty addresses
	stringAddresses = []string{"", ""}
	_, err = parseAddresses(stringAddresses)
	assert.Errorf(err, "empty string as an address should return an error")

	// valid address
	stringAddresses = []string{
		"/ip4/7.7.7.7/tcp/4242/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N",
		"/ip4/172.20.10.6/tcp/49526/p2p/12D3KooWB4J4mraN1nAB9Ge5w8JrzGRKDQ3iGyDyHZdbMWDtYg3R",
		"/ip4/172.20.10.6/udp/49529/p2p/12D3KooWGQTQuV6JfgpKpc847NMsxaFnKXDEpkN5kbeH1REW41BR",
		"/ip6/2001:0db8:85a3:0000:0000:8a2e:0370:7334/p2p/12D3KooWGQTQuV6JfgpKpc847NMsxaFnKXDEpkN5kbeH1SDFW34S",
	}
	_, err = parseAddresses(stringAddresses)
	assert.Nilf(err, "valid address should parse successfully")

	// invalid address
	stringAddresses = []string{
		"/ip4/127.0.0.1/udp/1234",
		"/ip4/7.7.7.7/tcp/4242/lsk/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N",
		"/ip4/7.7.7.7/lsk/4242/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N",
		"/ip4/7.7.7.7/tcp/4242/p2p/QmRelay/p2p-circuit/p2p/QmRelayedPeer",
		"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
	}
	_, err = parseAddresses(stringAddresses)
	assert.Errorf(err, "invalid address are not acceptable")
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

			got := extractIP(addr)
			assert.Equal(tt.want, got)
		})
	}
}
