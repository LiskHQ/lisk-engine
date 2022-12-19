package p2p

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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

	return true
}

func TestConfigJson(t *testing.T) {
	assert := assert.New(t)

	var c Config
	dump, err := json.Marshal(&c)
	assert.Nilf(err, fmt.Sprintf("should not be fail to convert an empty config to json: %s", dump))

	defaultCfg := DefaultConfig()
	wantedCfg := Config{
		IsSeedNode:  false,
		NetworkName: "lisk-testnet",
		SeedNodes:   []string{},
	}
	assert.Truef(isEqual(defaultCfg, wantedCfg), fmt.Sprintf("the default config is wrong \nwant(%+v)\n but got (%+v)", wantedCfg, defaultCfg))

	newCfg := Config{}
	err = json.Unmarshal([]byte(""), &newCfg)
	assert.Errorf(err, "converting an empty text to pubsub config shall return an error")

	c = Config{
		IsSeedNode:  true,
		NetworkName: "lisk",
		SeedNodes: []string{
			"/ip4/172.20.10.6/tcp/49526/p2p/12D3KooWB4J4mraN1nAB9Ge5w8JrzGRKDQ3iGyDyHZdbMWDtYg3R",
			"/ip4/172.20.10.6/tcp/49529/p2p/12D3KooWGQTQuV6JfgpKpc847NMsxaFnKXDEpkN5kbeH1REW41BR",
		},
	}
	ndump := `{"isSeedNode":true, "networkName":"lisk", "seedNodes": ["/ip4/172.20.10.6/tcp/49526/p2p/12D3KooWB4J4mraN1nAB9Ge5w8JrzGRKDQ3iGyDyHZdbMWDtYg3R","/ip4/172.20.10.6/tcp/49529/p2p/12D3KooWGQTQuV6JfgpKpc847NMsxaFnKXDEpkN5kbeH1REW41BR"]}`
	err = json.Unmarshal([]byte(ndump), &newCfg)
	assert.Nil(err)
	assert.Truef(isEqual(newCfg, c), fmt.Sprintf("dumped and parsed jsons are not the same \n got(%+v) \n want(%+v)", newCfg, c))
}
