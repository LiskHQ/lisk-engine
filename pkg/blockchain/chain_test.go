package blockchain

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/db"
)

type chainTestCtx struct {
	chain    *Chain
	genesis  *Block
	database *db.DB
}

func createChain() (*chainTestCtx, error) {
	fixtureFile, err := os.ReadFile("./fixtures_test/genesis_block.json")
	if err != nil {
		return nil, err
	}
	fixture := struct {
		ValidBlock *Block `json:"validBlock"`
	}{}

	if err := json.Unmarshal(fixtureFile, &fixture); err != nil {
		return nil, err
	}

	chain := NewChain(&ChainConfig{
		ChainID:               []byte{0, 0, 0, 0},
		MaxTransactionsLength: 15 * 1024,
		MaxBlockCache:         5,
	})

	database, err := db.NewInMemoryDB()
	if err != nil {
		return nil, err
	}

	fixture.ValidBlock.Init()
	chain.Init(fixture.ValidBlock, database)
	return &chainTestCtx{
		chain:    chain,
		genesis:  fixture.ValidBlock,
		database: database,
	}, nil
}

func TestChainInit(t *testing.T) {
	ctx, err := createChain()
	assert.NoError(t, err)
	assert.NotNil(t, ctx.chain.ChainID())
	assert.NotNil(t, ctx.chain.DataAccess())
}

func TestChainGenesisBlock(t *testing.T) {
	ctx, err := createChain()
	assert.NoError(t, err)

	exist, err := ctx.chain.GenesisBlockExist(ctx.genesis)
	assert.NoError(t, err)
	assert.False(t, exist)
	err = ctx.chain.PrepareCache()
	assert.NoError(t, err)

	batch := ctx.database.NewBatch()

	err = ctx.chain.AddBlock(batch, ctx.genesis, []*Event{}, 0, false)
	assert.NoError(t, err)
	exist, err = ctx.chain.GenesisBlockExist(ctx.genesis)
	assert.NoError(t, err)
	assert.True(t, exist)

	_, err = ctx.chain.GenesisBlockExist(createRandomBlock(0))
	assert.Contains(t, err.Error(), "do not match with existing genesis block")

	assert.Equal(t, ctx.genesis, ctx.chain.LastBlock())
	err = ctx.chain.PrepareCache()
	assert.NoError(t, err)

	err = ctx.chain.RemoveBlock(batch, false)
	assert.EqualError(t, err, "genesis block cannot be removed")
}

func TestChainBlockAddAndRemove(t *testing.T) {
	ctx, err := createChain()
	assert.NoError(t, err)
	{
		batch := ctx.database.NewBatch()
		err = ctx.chain.AddBlock(batch, ctx.genesis, []*Event{}, 0, false)
		assert.NoError(t, err)
	}

	assert.Equal(t, ctx.genesis, ctx.chain.LastBlock())

	blocks := []*Block{
		createRandomBlock(1),
		createRandomBlock(2),
		createRandomBlock(3),
	}
	for _, block := range blocks {
		batch := ctx.database.NewBatch()
		err = ctx.chain.AddBlock(batch, block, []*Event{}, 0, false)
		assert.NoError(t, err)
	}

	assert.Equal(t, blocks[2], ctx.chain.LastBlock())

	newChain := NewChain(&ChainConfig{
		ChainID:               []byte{0, 0, 0, 0},
		MaxTransactionsLength: 1,
		MaxBlockCache:         5,
	})
	newChain.Init(ctx.chain.genesisBlock, ctx.chain.database)
	err = newChain.PrepareCache()
	assert.NoError(t, err)

	{
		batch := ctx.database.NewBatch()
		err = ctx.chain.RemoveBlock(batch, false)
		assert.NoError(t, err)
		assert.Equal(t, blocks[1], ctx.chain.LastBlock())
	}

	lastBlocks, err := ctx.chain.GetLastNBlocks(2)
	assert.NoError(t, err)
	assert.Len(t, lastBlocks, 2)
	assert.Equal(t, blocks[0].Header.Height, lastBlocks[0].Header.Height)
	assert.Equal(t, blocks[1].Header.Height, lastBlocks[1].Header.Height)

	lastBlocks, err = ctx.chain.GetLastNBlocks(4)
	assert.NoError(t, err)
	assert.Len(t, lastBlocks, 3)
	assert.Equal(t, ctx.genesis.Header.Height, lastBlocks[0].Header.Height)
	assert.Equal(t, blocks[0].Header.Height, lastBlocks[1].Header.Height)
	assert.Equal(t, blocks[1].Header.Height, lastBlocks[2].Header.Height)
}
