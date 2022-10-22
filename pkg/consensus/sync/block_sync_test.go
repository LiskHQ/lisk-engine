package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetHeightWithGap(t *testing.T) {
	list := getHeightWithGap(0, 0, 10, 9)
	assert.Equal(t, []uint32{0}, list)
	list = getHeightWithGap(103, 0, 103, 9)
	assert.Equal(t, []uint32{103, 0}, list)
	list = getHeightWithGap(206, 0, 103, 9)
	assert.Equal(t, []uint32{206, 103, 0}, list)
}

func TestGetCommonBlockStartSearchHeight(t *testing.T) {
	height := getCommonBlockStartSearchHeight(0, 103)
	assert.Equal(t, 0, int(height))
	height = getCommonBlockStartSearchHeight(373, 103)
	assert.Equal(t, 309, int(height))
	height = getCommonBlockStartSearchHeight(411, 103)
	assert.Equal(t, 309, int(height))
	height = getCommonBlockStartSearchHeight(412, 103)
	assert.Equal(t, 309, int(height))
	height = getCommonBlockStartSearchHeight(413, 103)
	assert.Equal(t, 412, int(height))
}
