package p2p

import (
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

const (
	testTimestamp    = int64(123456789)
	testPeerID       = "testPeerID"
	testEvent        = "testEvent"
	testProcedure    = "testProcedure"
	testRPC          = "testRPC"
	testData         = "testData"
	testReqMsgID     = "123456789"
	testRequestData  = "testRequestData"
	testResponseData = "testResponseData"

	testIPv4TCP = "/ip4/127.0.0.1/tcp/0"
	testIPv4UDP = "/ip4/127.0.0.1/udp/0/quic"

	testTopic1 = "testTopic1"
	testTopic2 = "testTopic2"
	testTopic3 = "testTopic3"

	testError = "testError"

	testTimeout = time.Second * 3
)

type testLogger struct {
	log.Logger
	logs []string
}

func (l *testLogger) Debugf(msg string, others ...interface{}) {
	l.logs = append(l.logs, msg)
}

func (l *testLogger) Warningf(msg string, others ...interface{}) {
	l.logs = append(l.logs, msg)
}

func (l *testLogger) Errorf(msg string, others ...interface{}) {
	l.logs = append(l.logs, msg)
}
