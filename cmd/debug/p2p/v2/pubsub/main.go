package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	p2p "github.com/LiskHQ/lisk-engine/pkg/p2p/v2"
	"github.com/LiskHQ/lisk-engine/pkg/p2p/v2/pubsub"
)

const (
	DiscoveryServiceTag = "lisk"
	P2pPrefix           = "lisk-p2p-address-2022-12-11"
	PublicPrefix        = "discovered new peer"
	ErrorPrefix         = "lisk-error-2022-12-11"
)

var (
	port    = flag.Uint("port", 8010, "listening port")
	isLocal = flag.Bool("isLocal", false, "run the application in local mode")
)

func main() {
	logger, err := log.NewDefaultProductionLogger()
	if err != nil {
		panic(err)
	}

	nickFlag := flag.String("nick", "", "nickname to use in chat. will be generated if empty")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ip6quic := fmt.Sprintf("/ip6/::/udp/%d/quic", *port)
	ip4quic := fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", *port)

	ip6tcp := fmt.Sprintf("/ip6/::/tcp/%d", *port)
	ip4tcp := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port)

	cfg := p2p.Config{
		Addresses:                []string{ip6quic, ip4quic, ip6tcp, ip4tcp},
		AllowIncomingConnections: true,
		NetworkName:              "lisk-test",
		SeedNodes:                []string{},
	}
	err = cfg.InsertDefault()
	if err != nil {
		panic(err)
	}

	p, err := p2p.NewPeer(ctx, logger, cfg)
	if err != nil {
		panic(err)
	}

	sk := pubsub.NewScoreKeeper()
	ps, err := p2p.NewGossipSub(ctx, p, sk, cfg)
	if err != nil {
		panic(err)
	}

	nick := *nickFlag
	if len(nick) == 0 {
		nick = defaultNick(p.GetHost().ID())
	}

	cr, err := JoinChatRoom(ctx, ps, p.GetHost().ID(), nick, cfg.NetworkName)
	if err != nil {
		panic(err)
	}

	ui := NewChatUI(cr)
	if *isLocal {
		if err := setupDiscovery(p.GetHost(), ui); err != nil {
			panic(err)
		}
	}
	addrs, err := p.P2PAddrs()
	if err != nil {
		panic(err)
	}
	info := []string{}
	for _, addr := range addrs {
		str := fmt.Sprintf("%s%v", P2pPrefix, addr.String())
		info = append(info, str)
	}
	if err = ui.Run(info); err != nil {
		printErr("error running text UI: %s", err)
	}
}

func topicName(roomName string) string {
	return pubsub.MessageTopic(roomName)
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery.
type discoveryNotifee struct {
	h  host.Host
	ui *ChatUI
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.ui.inputCh <- fmt.Sprintf("%s %s", PublicPrefix, pi.ID.Pretty())
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		n.ui.inputCh <- fmt.Sprintf("%serror connecting to peer %s: %s", ErrorPrefix, pi.ID.Pretty(), err)
	}
}

// setupDiscovery creates an mDNS discovery service and attaches it to the libp2p Host.
// This lets us automatically discover peers on the same LAN and connect to them.
func setupDiscovery(h host.Host, ui *ChatUI) error {
	// setup mDNS discovery to find local peers
	s := mdns.NewMdnsService(h, DiscoveryServiceTag, &discoveryNotifee{h: h, ui: ui})
	return s.Start()
}
