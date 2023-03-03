package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	p2p "github.com/LiskHQ/lisk-engine/pkg/p2p"
)

const (
	DiscoveryServiceTag = "lisk"
	P2pPrefix           = "lisk-p2p-address-2022-12-11"
	PublicPrefix        = "discovered new peer"
	ErrorPrefix         = "lisk-error-2022-12-11"
)

var (
	port         = flag.Uint("port", 8010, "listening port")
	hasValidator = flag.Bool("hasValidator", true, "run the application with message validator")
)

const roomName = "test-chat"

func main() {
	nickFlag := flag.String("nick", "", "nickname to use in chat. will be generated if empty")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ip6quic := fmt.Sprintf("/ip6/::/udp/%d/quic", *port)
	ip4quic := fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", *port)

	ip6tcp := fmt.Sprintf("/ip6/::/tcp/%d", *port)
	ip4tcp := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port)

	cfgNet := config.NetworkConfig{
		Addresses:                []string{ip6quic, ip4quic, ip6tcp, ip4tcp},
		AllowIncomingConnections: true,
		SeedPeers:                []string{},
		Version:                  "2.0",
		ChainID:                  []byte{0x04, 0x00, 0x01, 0x02},
	}
	if err := cfgNet.InsertDefault(); err != nil {
		panic(err)
	}

	conn := p2p.NewConnection(&cfgNet)

	ch := make(chan *ChatMessage, ChatRoomBufSize)
	var validator p2p.Validator
	if *hasValidator {
		validator = func(ctx context.Context, msg *p2p.Message) p2p.ValidationResult {
			cm := new(ChatMessage)
			err := json.Unmarshal(msg.Data, cm)
			if err != nil {
				return p2p.ValidationIgnore
			}

			if strings.Contains(cm.Message, "invalid") {
				return p2p.ValidationReject
			} else {
				return p2p.ValidationAccept
			}
		}
	}
	if err := conn.RegisterEventHandler(topicName(roomName), func(event *p2p.Event) {
		readMessage(event, ch)
	}, validator); err != nil {
		panic(err)
	}

	if err := conn.Start(log.DefaultLogger, nil); err != nil {
		panic(err)
	}

	nick := *nickFlag
	if len(nick) == 0 {
		nick = defaultNick(conn.ID())
	}

	cr, err := JoinChatRoom(ctx, conn, ch, nick, roomName)
	if err != nil {
		panic(err)
	}

	ui := NewChatUI(cr)
	addrs, err := conn.MultiAddress()
	if err != nil {
		panic(err)
	}
	info := []string{}
	for _, addr := range addrs {
		str := fmt.Sprintf("%s%v", P2pPrefix, addr)
		info = append(info, str)
	}
	if err = ui.Run(info); err != nil {
		printErr("error running text UI: %s", err)
	}
}

func topicName(roomName string) string {
	return fmt.Sprintf("/lsk/%s", roomName)
}
