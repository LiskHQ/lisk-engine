package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlacklistIPs(t *testing.T) {
	assert := assert.New(t)

	containsInvalidIPs := []string{
		"128.20.12.11",
		"1.222.2222.12.12",
		"12.30.28.110",
	}
	_, err := ConnectionGaterOption(containsInvalidIPs)
	assert.ErrorContains(err, "is invalid")

	validIPs := []string{
		"127.0.0.1",
		"192.168.2.30",
	}
	_, err = ConnectionGaterOption(validIPs)
	assert.Nil(err)
}
