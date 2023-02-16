package p2p

import (
	"testing"

	"github.com/libp2p/go-libp2p/p2p/net/conngater"
	"github.com/stretchr/testify/assert"
)

func TestBlacklistIPs(t *testing.T) {
	assert := assert.New(t)

	cg, err := conngater.NewBasicConnectionGater(nil)
	assert.Nil(err)

	containsInvalidIPs := []string{
		"128.20.12.11",
		"1.222.2222.12.12",
		"12.30.28.110",
	}
	_, err = ConnectionGaterOption(cg, containsInvalidIPs)
	assert.ErrorContains(err, "is invalid")

	validIPs := []string{
		"127.0.0.1",
		"192.168.2.30",
	}
	_, err = ConnectionGaterOption(cg, validIPs)
	assert.Nil(err)
}
