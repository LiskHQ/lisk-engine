package p2p

import (
	"testing"
	"time"

	liskLog "github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestNewP2P(t *testing.T) {
	p2p := NewP2P()
	assert.NotNil(t, p2p)
	assert.Equal(t, true, p2p.conf.DummyConfigurationFeatureEnable)
}

func TestStart(t *testing.T) {
	p2p := NewP2P()
	logger, _ := liskLog.NewDefaultProductionLogger()
	err := p2p.Start(logger)
	assert.Nil(t, err)
	assert.Equal(t, logger, p2p.logger)
	assert.NotNil(t, p2p.host)
}

func TestStop(t *testing.T) {
	p2p := NewP2P()
	logger, _ := liskLog.NewDefaultProductionLogger()
	_ = p2p.Start(logger)

	ch := make(chan struct{})
	defer close(ch)

	go func() {
		err := p2p.Stop()
		assert.Nil(t, err)
		ch <- struct{}{}
	}()

	select {
	case <-ch:
		break
	case <-time.After(time.Second):
		t.Fatalf("timeout occurs, P2P stop is not working")
	}
}
