package p2p_v2

import (
	"errors"
	"strings"
	"testing"
)

func TestPeerInfoFromMultiAddrs(t *testing.T) {
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
			peer, err := PeerInfoFromMultiAddr(tt.addr)

			if tt.want.err == nil {
				if err != nil {
					t.Errorf("got %v, want %v", err, tt.want.err)
				}
			} else {
				if strings.Contains(err.Error(), tt.want.err.Error()) != true {
					t.Errorf("got %v, want %v", err, tt.want.err)
				}
				return // if error occurs skip the rest of body because it must not be checked
			}

			if peer.ID.String() != tt.want.id {
				t.Errorf("got %s, want %s", peer.ID.String(), tt.want.id)
			}
			if peer.Addrs[0].String() != tt.want.addr {
				t.Errorf("got %s, want %s", peer.Addrs[0].String(), tt.want.addr)
			}
		})
	}
}
