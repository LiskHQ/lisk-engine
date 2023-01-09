package p2p

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	liskLog "github.com/LiskHQ/lisk-engine/pkg/log"
)

func isEqual(lCfg, rCfg Config) bool {
	var lBuf, rBuf bytes.Buffer
	gob.NewDecoder(&lBuf).Decode(&lCfg.SeedNodes)
	gob.NewDecoder(&rBuf).Decode(&rCfg.SeedNodes)
	if !bytes.Equal(lBuf.Bytes(), rBuf.Bytes()) {
		return false
	}

	if lCfg.IsSeedNode != rCfg.IsSeedNode {
		return false
	}

	if lCfg.NetworkName != rCfg.NetworkName {
		return false
	}

	if lCfg.AllowIncomingConnections != rCfg.AllowIncomingConnections {
		return false
	}

	if lCfg.DummyConfigurationFeatureEnable != rCfg.DummyConfigurationFeatureEnable {
		return false
	}

	return true
}

func TestConfigJson(t *testing.T) {
	assert := assert.New(t)

	var c Config
	dump, err := json.Marshal(&c)
	assert.Nilf(err, fmt.Sprintf("should not be fail to convert an empty config to json: %s", dump))

	p2p := NewP2P()
	wantedCfg := Config{
		DummyConfigurationFeatureEnable: true,
		AllowIncomingConnections:        true,
		IsSeedNode:                      false,
		NetworkName:                     "xxxxxxxx",
		SeedNodes:                       []string{},
	}
	assert.Truef(isEqual(p2p.conf, wantedCfg), fmt.Sprintf("the default config is wrong \nwant(%+v)\n but got (%+v)", wantedCfg, p2p.conf))

	newCfg := Config{}
	err = json.Unmarshal([]byte(""), &newCfg)
	assert.Errorf(err, "converting an empty text to pubsub config shall return an error")

	c = Config{
		DummyConfigurationFeatureEnable: true,
		AllowIncomingConnections:        false,
		IsSeedNode:                      true,
		NetworkName:                     "lisk",
		SeedNodes: []string{
			"/ip4/172.20.10.6/tcp/49526/p2p/12D3KooWB4J4mraN1nAB9Ge5w8JrzGRKDQ3iGyDyHZdbMWDtYg3R",
			"/ip4/172.20.10.6/tcp/49529/p2p/12D3KooWGQTQuV6JfgpKpc847NMsxaFnKXDEpkN5kbeH1REW41BR",
		},
	}
	ndump := `{"dummyConfigurationFeatureEnable":true, "isSeedNode":true, "networkName":"lisk", "seedNodes": ["/ip4/172.20.10.6/tcp/49526/p2p/12D3KooWB4J4mraN1nAB9Ge5w8JrzGRKDQ3iGyDyHZdbMWDtYg3R","/ip4/172.20.10.6/tcp/49529/p2p/12D3KooWGQTQuV6JfgpKpc847NMsxaFnKXDEpkN5kbeH1REW41BR"]}`
	err = json.Unmarshal([]byte(ndump), &newCfg)
	assert.Nil(err)
	assert.Truef(isEqual(newCfg, c), fmt.Sprintf("dumped and parsed jsons are not the same \n got(%+v) \n want(%+v)", newCfg, c))
}

func TestP2P_NewP2P(t *testing.T) {
	p2p := NewP2P()
	assert.NotNil(t, p2p)
	assert.Equal(t, true, p2p.conf.DummyConfigurationFeatureEnable)
}

func TestP2P_Start(t *testing.T) {
	p2p := NewP2P()
	logger, _ := liskLog.NewDefaultProductionLogger()
	err := p2p.Start(logger)
	assert.Nil(t, err)
	assert.Equal(t, logger, p2p.logger)
	assert.NotNil(t, p2p.host)
}

func TestP2P_Stop(t *testing.T) {
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
