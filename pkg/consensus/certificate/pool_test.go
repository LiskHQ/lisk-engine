package certificate

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

func TestPoolSelect(t *testing.T) {
	self := codec.Lisk32(crypto.RandomBytes(20))
	validator := codec.Lisk32(crypto.RandomBytes(20))
	block1 := crypto.RandomBytes(32)
	block2 := crypto.RandomBytes(32)

	singleCommits := SingleCommits{
		{
			blockID:              block1,
			height:               100,
			validatorAddress:     self,
			certificateSignature: codec.Hex(crypto.RandomBytes(64)),
			internal:             true,
		},
		{
			blockID:              block1,
			height:               100,
			validatorAddress:     validator,
			certificateSignature: codec.Hex(crypto.RandomBytes(64)),
			internal:             false,
		},
		{
			blockID:              block2,
			height:               101,
			validatorAddress:     validator,
			certificateSignature: codec.Hex(crypto.RandomBytes(64)),
			internal:             false,
		},
	}

	pool := NewPool()

	for _, commit := range singleCommits {
		pool.Add(commit)
	}

	assert.True(t, pool.Has(singleCommits[0]))

	// get in order of height lowest
	selected := pool.Select(300, 1)
	assert.Len(t, selected, 1)
	assert.Equal(t, selected[0].height, uint32(100))
	assert.Equal(t, selected[0].validatorAddress, self)

	// get in order of height lowest
	selected = pool.Select(300, 2)
	assert.Len(t, selected, 2)
	assert.Equal(t, selected[0].height, uint32(100))
	assert.Equal(t, selected[1].height, uint32(100))
	assert.Equal(t, selected[0].validatorAddress, self)
	assert.Equal(t, selected[1].validatorAddress, validator)

	// get from heights with internal has priority
	selected = pool.Select(0, 2)
	assert.Len(t, selected, 2)
	assert.Equal(t, selected[0].height, uint32(100))
	assert.Equal(t, selected[1].height, uint32(101))
	assert.Equal(t, selected[0].validatorAddress, self)
	assert.Equal(t, selected[1].validatorAddress, validator)
}

func TestPoolUpgradeCleanup(t *testing.T) {
	self := codec.Lisk32(crypto.RandomBytes(20))
	validator := codec.Lisk32(crypto.RandomBytes(20))
	block1 := crypto.RandomBytes(32)
	block2 := crypto.RandomBytes(32)

	singleCommits := SingleCommits{
		{
			blockID:              block1,
			height:               100,
			validatorAddress:     self,
			certificateSignature: codec.Hex(crypto.RandomBytes(64)),
			internal:             true,
		},
		{
			blockID:              block1,
			height:               100,
			validatorAddress:     validator,
			certificateSignature: codec.Hex(crypto.RandomBytes(64)),
			internal:             false,
		},
		{
			blockID:              block2,
			height:               101,
			validatorAddress:     validator,
			certificateSignature: codec.Hex(crypto.RandomBytes(64)),
			internal:             false,
		},
		{
			blockID:              codec.Hex(crypto.RandomBytes(32)),
			height:               102,
			validatorAddress:     validator,
			certificateSignature: codec.Hex(crypto.RandomBytes(64)),
			internal:             false,
		},
	}

	pool := NewPool()

	for _, commit := range singleCommits {
		pool.Add(commit)
	}

	selected := pool.Get(100)
	assert.Len(t, selected, 2)
	assert.Equal(t, selected[0].height, uint32(100))
	assert.Equal(t, selected[1].height, uint32(100))

	pool.Upgrade(SingleCommits{singleCommits[0], singleCommits[2]})
	assert.Len(t, pool.nonGossiped, 2)
	assert.Len(t, pool.gossiped, 2)

	selected = pool.Get(100)
	assert.Len(t, selected, 2)
	assert.Equal(t, selected[0].height, uint32(100))
	assert.Equal(t, selected[1].height, uint32(100))

	pool.Cleanup(func(h uint32) bool {
		return h != 100
	})

	assert.Len(t, pool.nonGossiped, 1)
	assert.Len(t, pool.gossiped, 1)
}
