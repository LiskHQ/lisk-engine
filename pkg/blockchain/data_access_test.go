package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/db"
)

type dataAccessTestCtx struct {
	database *db.DB
}

func newDataAccessTestCtx() *dataAccessTestCtx {
	database, err := db.NewInMemoryDB()
	if err != nil {
		panic(err)
	}
	return &dataAccessTestCtx{
		database: database,
	}
}

func (c *dataAccessTestCtx) close() {
	c.database.Close()
}

func TestDataAccessGetBlockHeader(t *testing.T) {
	ctx := newDataAccessTestCtx()
	defer ctx.close()

	// Set data
	block1 := createRandomBlock(1)
	ctx.database.Set(bytes.Join(DBPrefixToBytes(dbPrefixBlockIDToBlockHeader), block1.Header.ID), block1.Header.Encode())

	dataAccess := NewDataAccess(ctx.database, 1, 1)

	header, err := dataAccess.GetBlockHeader(block1.Header.ID)
	assert.NoError(t, err)
	assert.Equal(t, block1.Header.Height, header.Height)
	assert.Equal(t, block1.Header.AggregateCommit.AggregationBits, header.AggregateCommit.AggregationBits)

	// Check cache is working
	dataAccess.Cache(block1)
	assert.True(t, dataAccess.Cached(block1.Header.Height))

	header, err = dataAccess.GetBlockHeader(block1.Header.ID)
	assert.NoError(t, err)
	assert.Equal(t, block1.Header.Height, header.Height)
	assert.Equal(t, block1.Header.AggregateCommit.AggregationBits, header.AggregateCommit.AggregationBits)

	_, err = dataAccess.GetBlockHeader(crypto.RandomBytes(32))
	assert.EqualError(t, err, "data was not found")
}

func TestDataAccessGetBlockHeaders(t *testing.T) {
	ctx := newDataAccessTestCtx()
	defer ctx.close()
	dataAccess := NewDataAccess(ctx.database, 1, 1)
	blocks := []*Block{
		createRandomBlock(1),
		createRandomBlock(2),
	}
	batch := ctx.database.NewBatch()
	for _, b := range blocks {
		dataAccess.saveBlock(batch, b, []*Event{}, 0, false)
	}
	ctx.database.Write(batch)

	fetched, err := dataAccess.GetBlockHeaders([][]byte{blocks[0].Header.ID})
	assert.NoError(t, err)
	assert.Len(t, fetched, 1)

	fetched, err = dataAccess.GetBlockHeaders([][]byte{blocks[0].Header.ID, crypto.RandomBytes(32)})
	assert.NoError(t, err)
	assert.Len(t, fetched, 1)

	fetched, err = dataAccess.GetBlockHeaders([][]byte{crypto.RandomBytes(32), crypto.RandomBytes(32)})
	assert.NoError(t, err)
	assert.Len(t, fetched, 0)
}

func TestDataAccessGetBlockByHeight(t *testing.T) {
	ctx := newDataAccessTestCtx()
	defer ctx.close()

	// Set data
	block1 := createRandomBlock(1)
	ctx.database.Set(bytes.Join(DBPrefixToBytes(dbPrefixBlockHeightToBlockID), bytes.FromUint32(1)), block1.Header.ID)
	ctx.database.Set(bytes.Join(DBPrefixToBytes(dbPrefixBlockIDToBlockHeader), block1.Header.ID), block1.Header.Encode())

	block2 := createRandomBlock(2)
	ctx.database.Set(bytes.Join(DBPrefixToBytes(dbPrefixBlockHeightToBlockID), bytes.FromUint32(2)), block2.Header.ID)
	ctx.database.Set(bytes.Join(DBPrefixToBytes(dbPrefixBlockIDToBlockHeader), block2.Header.ID), block2.Header.Encode())

	dataAccess := NewDataAccess(ctx.database, 1, 1)
	blocks, err := dataAccess.GetBlocksBetweenHeight(1, 3)
	assert.Contains(t, err.Error(), "not found")
	assert.Nil(t, blocks)
	blocks, err = dataAccess.GetBlocksBetweenHeight(1, 2)
	assert.Nil(t, err)
	assert.Len(t, blocks, 2)
}

func TestDataAccessGetBlockHeadersByHeights(t *testing.T) {
	ctx := newDataAccessTestCtx()
	defer ctx.close()
	dataAccess := NewDataAccess(ctx.database, 1, 1)
	blocks := []*Block{
		createRandomBlock(1),
		createRandomBlock(2),
	}
	batch := ctx.database.NewBatch()
	for _, b := range blocks {
		dataAccess.saveBlock(batch, b, []*Event{}, 0, false)
	}
	ctx.database.Write(batch)

	fetched, err := dataAccess.GetBlockHeadersByHeights([]uint32{1})
	assert.NoError(t, err)
	assert.Len(t, fetched, 1)

	fetched, err = dataAccess.GetBlockHeadersByHeights([]uint32{2, 3})
	assert.NoError(t, err)
	assert.Len(t, fetched, 1)

	fetched, err = dataAccess.GetBlockHeadersByHeights([]uint32{10, 11})
	assert.NoError(t, err)
	assert.Len(t, fetched, 0)
}

func TestDataAccessGetBlockHeaderByHeight(t *testing.T) {
	ctx := newDataAccessTestCtx()
	defer ctx.close()
	dataAccess := NewDataAccess(ctx.database, 1, 1)
	blocks := []*Block{
		createRandomBlock(1),
		createRandomBlock(2),
	}
	batch := ctx.database.NewBatch()
	for _, b := range blocks {
		dataAccess.saveBlock(batch, b, []*Event{}, 0, false)
	}
	ctx.database.Write(batch)

	dataAccess.Cache(blocks[0])

	fetched, err := dataAccess.GetBlockHeaderByHeight(1)
	assert.NoError(t, err)
	assert.Equal(t, blocks[0].Header, fetched)

	fetched, err = dataAccess.GetBlockHeaderByHeight(2)
	assert.NoError(t, err)
	assert.Equal(t, blocks[1].Header, fetched)
}

func TestDataAccessGetLastBlockHeader(t *testing.T) {
	ctx := newDataAccessTestCtx()
	defer ctx.close()
	dataAccess := NewDataAccess(ctx.database, 1, 1)
	blocks := []*Block{
		createRandomBlock(1),
		createRandomBlock(2),
	}
	batch := ctx.database.NewBatch()
	for _, b := range blocks {
		dataAccess.saveBlock(batch, b, []*Event{}, 0, false)
	}
	ctx.database.Write(batch)

	fetched, err := dataAccess.GetLastBlockHeader()
	assert.NoError(t, err)
	assert.Equal(t, blocks[1].Header, fetched)
}

func TestDataAccessGetLastBlock(t *testing.T) {
	ctx := newDataAccessTestCtx()
	defer ctx.close()
	dataAccess := NewDataAccess(ctx.database, 1, 1)
	blocks := []*Block{
		createRandomBlock(1),
		createRandomBlock(2),
	}
	batch := ctx.database.NewBatch()
	for _, b := range blocks {
		dataAccess.saveBlock(batch, b, []*Event{}, 0, false)
		dataAccess.Cache(b)
	}
	ctx.database.Write(batch)

	fetched, err := dataAccess.GetLastBlock()
	assert.NoError(t, err)
	assert.Equal(t, blocks[1], fetched)
}

func TestDataAccessGetTransaction(t *testing.T) {
	ctx := newDataAccessTestCtx()
	defer ctx.close()
	dataAccess := NewDataAccess(ctx.database, 1, 1)
	blocks := []*Block{
		createRandomBlock(1),
		createRandomBlock(2),
	}
	batch := ctx.database.NewBatch()
	for _, b := range blocks {
		dataAccess.saveBlock(batch, b, []*Event{}, 0, false)
	}
	ctx.database.Write(batch)
	dataAccess.Cache(blocks[0])

	fetched, err := dataAccess.getTransaction(blocks[1].Transactions[0].ID)
	assert.NoError(t, err)
	assert.Equal(t, blocks[1].Transactions[0], fetched)

	fetched, err = dataAccess.getTransaction(blocks[0].Transactions[0].ID)
	assert.NoError(t, err)
	assert.Equal(t, blocks[0].Transactions[0], fetched)
}

func TestDataAccessGetTransactions(t *testing.T) {
	ctx := newDataAccessTestCtx()
	defer ctx.close()
	dataAccess := NewDataAccess(ctx.database, 1, 1)
	blocks := []*Block{
		createRandomBlock(1),
		createRandomBlock(2),
	}
	batch := ctx.database.NewBatch()
	for _, b := range blocks {
		dataAccess.saveBlock(batch, b, []*Event{}, 0, false)
		dataAccess.Cache(b)
	}
	ctx.database.Write(batch)

	fetched, err := dataAccess.GetTransactions([][]byte{blocks[0].Transactions[0].ID})
	assert.NoError(t, err)
	assert.Len(t, fetched, 1)
	assert.Equal(t, []*Transaction{blocks[0].Transactions[0]}, fetched)

	fetched, err = dataAccess.GetTransactions([][]byte{blocks[0].Transactions[0].ID, crypto.RandomBytes(32)})
	assert.NoError(t, err)
	assert.Len(t, fetched, 1)
	assert.Equal(t, []*Transaction{blocks[0].Transactions[0]}, fetched)
}

func TestDataAccessGetEvents(t *testing.T) {
	ctx := newDataAccessTestCtx()
	defer ctx.close()
	dataAccess := NewDataAccess(ctx.database, 1, 1)
	blocks := []*Block{
		createRandomBlock(1),
		createRandomBlock(2),
	}
	batch := ctx.database.NewBatch()
	for _, b := range blocks {
		dataAccess.saveBlock(batch, b, []*Event{
			NewEventFromValues("token", "transfer", crypto.RandomBytes(20), []codec.Hex{crypto.RandomBytes(32)}, b.Header.Height, 0),
			NewEventFromValues("token", "transfer", crypto.RandomBytes(20), []codec.Hex{crypto.RandomBytes(32)}, b.Header.Height, 1),
		}, 0, false)
		dataAccess.Cache(b)
	}
	ctx.database.Write(batch)

	events, err := dataAccess.GetEvents(2)
	assert.NoError(t, err)
	assert.Len(t, events, 2)

	_, err = dataAccess.GetEvents(10)
	assert.EqualError(t, err, "data was not found")
}

func TestDataAccessTempBlocks(t *testing.T) {
	ctx := newDataAccessTestCtx()
	defer ctx.close()
	dataAccess := NewDataAccess(ctx.database, 1, 1)
	blocks := []*Block{
		createRandomBlock(1),
		createRandomBlock(2),
		createRandomBlock(3),
		createRandomBlock(4),
	}
	batch := ctx.database.NewBatch()
	for _, b := range blocks {
		dataAccess.saveBlock(batch, b, []*Event{}, 0, false)
		dataAccess.Cache(b)
	}
	ctx.database.Write(batch)

	batch = ctx.database.NewBatch()
	dataAccess.removeBlock(batch, blocks[3], true)
	dataAccess.removeBlock(batch, blocks[2], true)
	ctx.database.Write(batch)

	tempBlocks, err := dataAccess.GetTempBlocks()
	assert.NoError(t, err)
	assert.Len(t, tempBlocks, 2)
	assert.Equal(t, blocks[3], tempBlocks[0])
	assert.Equal(t, blocks[2], tempBlocks[1])

	dataAccess.ClearTempBlocks()

	tempBlocks, err = dataAccess.GetTempBlocks()
	assert.NoError(t, err)
	assert.Len(t, tempBlocks, 0)
}

func TestDataAccessSaveBlock(t *testing.T) {
	ctx := newDataAccessTestCtx()
	defer ctx.close()

	block := createRandomBlock(1)

	dataAccess := NewDataAccess(ctx.database, 1, 1)
	events := []*Event{createRandomEvent(block.Header.Height, 0), createRandomEvent(block.Header.Height, 1)}
	{
		batch := ctx.database.NewBatch()
		dataAccess.saveBlock(batch, block, events, 0, false)
		ctx.database.Write(batch)
	}

	fetched, err := dataAccess.getBlock(block.Header.ID)
	assert.NoError(t, err)
	assert.Equal(t, block.Header, fetched.Header)
	assert.Equal(t, block.Assets, fetched.Assets)
	assert.Equal(t, block.Transactions, fetched.Transactions)
	fetchedEvents, err := dataAccess.GetEvents(block.Header.Height)
	assert.NoError(t, err)
	assert.Len(t, fetchedEvents, len(events))

	block = createRandomBlock(2)
	{
		batch := ctx.database.NewBatch()
		dataAccess.saveBlock(batch, block, events, 1, true)
		ctx.database.Write(batch)
	}

	block = createRandomBlock(3)
	{
		batch := ctx.database.NewBatch()
		dataAccess.saveBlock(batch, block, events, 1, true)
		ctx.database.Write(batch)
	}
	_, err = dataAccess.GetEvents(1)
	assert.EqualError(t, err, "data was not found")
}

func TestDataAccessRemoveBlock(t *testing.T) {
	ctx := newDataAccessTestCtx()
	defer ctx.close()
	dataAccess := NewDataAccess(ctx.database, 1, 1)
	blocks := []*Block{
		createRandomBlock(1),
		createRandomBlock(2),
		createRandomBlock(3),
		createRandomBlock(4),
	}
	batch := ctx.database.NewBatch()
	for _, b := range blocks {
		dataAccess.saveBlock(batch, b, []*Event{}, 0, false)
		dataAccess.Cache(b)
	}
	ctx.database.Write(batch)

	batch = ctx.database.NewBatch()
	dataAccess.removeBlock(batch, blocks[3], true)
	dataAccess.removeBlock(batch, blocks[2], true)
	ctx.database.Write(batch)

	_, err := dataAccess.getTransaction(blocks[3].Transactions[1].ID)
	assert.EqualError(t, err, "data was not found")
	_, err = dataAccess.getTransaction(blocks[3].Transactions[0].ID)
	assert.EqualError(t, err, "data was not found")
	_, err = dataAccess.getBlockHeader(blocks[3].Header.ID)
	assert.EqualError(t, err, "data was not found")

	_, exist := ctx.database.Get(bytes.Join(DBPrefixToBytes(dbPrefixBlockIDToAssets), blocks[3].Header.ID))
	assert.False(t, exist)
}
