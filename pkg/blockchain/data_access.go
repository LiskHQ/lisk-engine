package blockchain

import (
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/collection/ints"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/db"
)

// DBPrefix is a type to define prefix for the data type
//
//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen
type DBPrefix uint8

const (
	idSize = 32
)

// DBPrefix for all the low level common data.
var (
	dbPrefixBlockIDToBlockHeader DBPrefix = 3
	dbPrefixBlockHeightToBlockID DBPrefix = 4
	dbPrefixBlockIDToTxs         DBPrefix = 5
	dbPrefixTxIDToTx             DBPrefix = 6
	dbPrefixTemp                 DBPrefix = 7
	dbPrefixBlockIDToAssets      DBPrefix = 8
	dbPrefixBlockHeightToEvents  DBPrefix = 9

	dbPrefixFinalizedHeight DBPrefix = 27

	DBPrefixState     DBPrefix = 10
	DBPrefixStateDiff DBPrefix = 51
)

// DBPrefixToBytes aconverts prefix to slice.
func DBPrefixToBytes(prefix DBPrefix) []byte {
	return []byte{uint8(prefix)}
}

// DataAccess gives access to the blockchain data.
type DataAccess struct {
	database             *db.DB
	cache                *blockCache
	keepEventsForHeights int
}

// NewDataAccess returns new instance of data access.
func NewDataAccess(db *db.DB, maxCacheSize, keepEventsForHeights int) *DataAccess {
	dataAccess := &DataAccess{
		database:             db,
		cache:                newBlockCache(maxCacheSize),
		keepEventsForHeights: keepEventsForHeights,
	}
	// get last block
	// fill cache
	return dataAccess
}

// CachedLastBlock returns the latest block without popping.
func (d *DataAccess) CachedLastBlock() *Block {
	last, _ := d.cache.last()
	return last
}

// Cache a new block.
func (d *DataAccess) Cache(block *Block) error {
	return d.cache.push(block)
}

func (d *DataAccess) Cached(height uint32) bool {
	_, exist := d.cache.getByHeight(height)
	return exist
}

// RemoveCache removes last block from cache.
func (d *DataAccess) RemoveCache() {
	d.cache.pop()
}

// GetBlockHeader returns block header if data exists.
func (d *DataAccess) GetBlockHeader(id []byte) (*BlockHeader, error) {
	cachedBlock, exist := d.cache.get(id)
	if exist {
		return cachedBlock.Header, nil
	}
	return d.getBlockHeader(id)
}

// GetBlockHeaders returns all block header with id if exist.
func (d *DataAccess) GetBlockHeaders(ids [][]byte) ([]*BlockHeader, error) {
	eg := new(errgroup.Group)
	headers := make([]*BlockHeader, len(ids))
	hasEmpty := false
	for i, id := range ids {
		i, id := i, id // https://golang.org/doc/faq#closures_and_goroutines
		eg.Go(func() error {
			header, err := d.GetBlockHeader(id)
			if err != nil {
				if !errors.Is(err, db.ErrDataNotFound) {
					return err
				}
				hasEmpty = true
				return nil
			}
			headers[i] = header
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	if !hasEmpty {
		return headers, nil
	}
	nonNilHeaders := []*BlockHeader{}
	for _, h := range headers {
		if h != nil {
			nonNilHeaders = append(nonNilHeaders, h)
		}
	}
	return nonNilHeaders, nil
}

// GetBlockHeadersByHeights returns all block header with heights.
func (d *DataAccess) GetBlockHeadersByHeights(heights []uint32) ([]*BlockHeader, error) {
	eg := new(errgroup.Group)
	headers := make([]*BlockHeader, len(heights))
	hasEmpty := false
	for i, height := range heights {
		i, height := i, height // https://golang.org/doc/faq#closures_and_goroutines
		eg.Go(func() error {
			header, err := d.GetBlockHeaderByHeight(height)
			if err != nil {
				if !errors.Is(err, db.ErrDataNotFound) {
					return err
				}
				hasEmpty = true
				return nil
			}
			headers[i] = header
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	if !hasEmpty {
		return headers, nil
	}
	nonNilHeaders := []*BlockHeader{}
	for _, h := range headers {
		if h != nil {
			nonNilHeaders = append(nonNilHeaders, h)
		}
	}
	return nonNilHeaders, nil
}

// GetBlockHeaderByHeight returns block header with height.
func (d *DataAccess) GetBlockHeaderByHeight(height uint32) (*BlockHeader, error) {
	cachedBlock, exist := d.cache.getByHeight(height)
	if exist {
		return cachedBlock.Header, nil
	}
	heightBytes := bytes.FromUint32(height)
	id, err := d.database.Get(bytes.Join(DBPrefixToBytes(dbPrefixBlockHeightToBlockID), heightBytes))
	if err != nil {
		return nil, err
	}
	return d.getBlockHeader(id)
}

// GetLastBlockHeader returns last block header.
func (d *DataAccess) GetLastBlockHeader() (*BlockHeader, error) {
	block, err := d.getLastBlock()
	if err != nil {
		return nil, err
	}
	return block.Header, nil
}

// GetBlock returns a block by id if exist.
func (d *DataAccess) GetBlock(id []byte) (*Block, error) {
	cachedBlock, exist := d.cache.get(id)
	if exist {
		return cachedBlock, nil
	}
	return d.getBlock(id)
}

// GetLastBlock of the blockcchain.
func (d *DataAccess) GetLastBlock() (*Block, error) {
	block, exist := d.cache.last()
	if !exist {
		return nil, db.ErrDataNotFound
	}
	return block, nil
}

// GetBlockByHeight returns block by height.
func (d *DataAccess) GetBlockByHeight(height uint32) (*Block, error) {
	cachedBlock, exist := d.cache.getByHeight(height)
	if exist {
		return cachedBlock, nil
	}
	heightBytes := bytes.FromUint32(height)
	id, err := d.database.Get(bytes.Join(DBPrefixToBytes(dbPrefixBlockHeightToBlockID), heightBytes))
	if err != nil {
		return nil, err
	}
	return d.getBlock(id)
}

// GetBlocksBetweenHeight returns all block between the height.
func (d *DataAccess) GetBlocksBetweenHeight(from, to uint32) ([]*Block, error) {
	eg := new(errgroup.Group)
	blocks := make([]*Block, to-from+1)
	for height := from; height <= to; height++ {
		h := height // https://golang.org/doc/faq#closures_and_goroutines
		eg.Go(func() error {
			block, err := d.GetBlockByHeight(h)
			if err != nil {
				return fmt.Errorf("failed to get block with height %d with %w", h, err)
			}
			blocks[h-from] = block
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	SortBlockByHeightAsc(blocks)
	return blocks, nil
}

// GetTransaction return transaction by id.
func (d *DataAccess) GetTransaction(id []byte) (*Transaction, error) {
	return d.getTransaction(id)
}

// GetTransaction return transaction by id.
func (d *DataAccess) GetTransactions(ids [][]byte) ([]*Transaction, error) {
	eg := new(errgroup.Group)
	txs := make([]*Transaction, len(ids))
	hasEmpty := false
	for i, id := range ids {
		i, id := i, id // https://golang.org/doc/faq#closures_and_goroutines
		eg.Go(func() error {
			tx, err := d.getTransaction(id)
			if err != nil {
				if !errors.Is(err, db.ErrDataNotFound) {
					return err
				}
				hasEmpty = true
				return nil
			}
			txs[i] = tx
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	if !hasEmpty {
		return txs, nil
	}
	nonNilTxs := []*Transaction{}
	for _, tx := range txs {
		if tx != nil {
			nonNilTxs = append(nonNilTxs, tx)
		}
	}
	return nonNilTxs, nil
}

// GetTempBlocks returns all temp block exisst.
func (d *DataAccess) GetTempBlocks() ([]*Block, error) {
	tempBlocks, err := d.database.Iterate(DBPrefixToBytes(dbPrefixTemp), -1, true)
	if err != nil {
		return nil, err
	}
	if len(tempBlocks) == 0 {
		return []*Block{}, nil
	}
	blocks := make([]*Block, len(tempBlocks))
	for i, data := range tempBlocks {
		block, err := NewBlock(data.Value())
		if err != nil {
			return nil, err
		}
		blocks[i] = block
	}
	return blocks, nil
}

// ClearTempBlocks removes all temp blocks.
func (d *DataAccess) ClearTempBlocks() error {
	tempBlockKeys, err := d.database.Iterate(DBPrefixToBytes(dbPrefixTemp), -1, true)
	if err != nil {
		return err
	}
	if len(tempBlockKeys) == 0 {
		return nil
	}
	batch := d.database.NewBatch()
	for _, data := range tempBlockKeys {
		if err := batch.Del(data.Key()); err != nil {
			return err
		}
	}
	return d.database.Write(batch)
}

func (d *DataAccess) GetEvents(height uint32) ([]*Event, error) {
	encodedEvents, err := d.database.Get(bytes.Join(DBPrefixToBytes(dbPrefixBlockHeightToEvents), bytes.FromUint32(height)))
	if err != nil {
		return nil, err
	}
	events, err := bytesToEvents(encodedEvents)
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		event.UpdateID()
	}
	return events, nil
}

func (d *DataAccess) GetFinalizedHeight() (uint32, error) {
	finalizedHeightByte, err := d.database.Get(DBPrefixToBytes(dbPrefixFinalizedHeight))
	if err != nil {
		return 0, err
	}
	return bytes.ToUint32(finalizedHeightByte), nil
}

func (d *DataAccess) getBlock(id []byte) (*Block, error) {
	header, err := d.getBlockHeader(id)
	if err != nil {
		return nil, err
	}
	txs, err := d.getTransactions(id)
	if err != nil {
		return nil, err
	}
	assets, err := d.getBlockAssets(id)
	if err != nil {
		return nil, err
	}
	block := &Block{
		Header:       header,
		Assets:       assets,
		Transactions: txs,
	}
	return block, nil
}

func (d *DataAccess) getBlockHeader(id []byte) (*BlockHeader, error) {
	headerBytes, err := d.database.Get(bytes.Join(DBPrefixToBytes(dbPrefixBlockIDToBlockHeader), id))
	if err != nil {
		return nil, err
	}
	header := &BlockHeader{
		ID: crypto.Hash(headerBytes),
	}
	if err := header.Decode(headerBytes); err != nil {
		return nil, err
	}
	return header, nil
}

func (d *DataAccess) getTransactions(blockID []byte) ([]*Transaction, error) {
	txIDs, err := d.database.Get(bytes.Join(DBPrefixToBytes(dbPrefixBlockIDToTxs), blockID))
	if err != nil {
		if errors.Is(err, db.ErrDataNotFound) {
			return []*Transaction{}, nil
		}
		return nil, err
	}
	resultSize := len(txIDs)
	size := resultSize / idSize
	txs := make([]*Transaction, size)
	eg := new(errgroup.Group)
	for i := 0; i < size; i++ {
		i := i // https://golang.org/doc/faq#closures_and_goroutines
		eg.Go(func() error {
			offset := i * idSize
			nextID := txIDs[offset : offset+idSize]
			tx, err := d.getTransaction(nextID)
			if err != nil {
				return err
			}
			txs[i] = tx
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return txs, nil
}

func (d *DataAccess) getBlockAssets(blockID []byte) (BlockAssets, error) {
	assets, err := d.database.Get(bytes.Join(DBPrefixToBytes(dbPrefixBlockIDToAssets), blockID))
	if err != nil {
		if errors.Is(err, db.ErrDataNotFound) {
			return BlockAssets{}, nil
		}
		return nil, err
	}
	return bytesToAssets(assets)
}

func (d *DataAccess) getTransaction(id []byte) (*Transaction, error) {
	txBytes, err := d.database.Get(bytes.Join(DBPrefixToBytes(dbPrefixTxIDToTx), id))
	if err != nil {
		return nil, err
	}
	tx, err := NewTransaction(txBytes)
	if err != nil {
		return nil, err
	}
	if err := tx.Decode(txBytes); err != nil {
		return nil, err
	}
	return tx, nil
}

func (d *DataAccess) getLastBlock() (*Block, error) {
	ids, err := d.database.Iterate(DBPrefixToBytes(dbPrefixBlockHeightToBlockID), 1, true)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, db.ErrDataNotFound
	}
	return d.GetBlock(ids[0].Value())
}

func (d *DataAccess) saveBlock(batch *db.Batch, block *Block, events []*Event, finalizedHeight uint32, removeTemp bool) error {
	height := bytes.FromUint32(block.Header.Height)
	encodedBlockHeader, err := block.Header.Encode()
	if err != nil {
		return err
	}
	if err := batch.Set(bytes.Join(DBPrefixToBytes(dbPrefixBlockIDToBlockHeader), block.Header.ID), encodedBlockHeader); err != nil {
		return err
	}
	if err := batch.Set(bytes.Join(DBPrefixToBytes(dbPrefixBlockHeightToBlockID), height), block.Header.ID); err != nil {
		return err
	}
	if len(block.Transactions) > 0 {
		idsBytes := make([][]byte, len(block.Transactions))
		for i, tx := range block.Transactions {
			encodedTx, err := tx.Encode()
			if err != nil {
				return err
			}
			if err := batch.Set(bytes.Join(DBPrefixToBytes(dbPrefixTxIDToTx), tx.ID), encodedTx); err != nil {
				return err
			}
			idsBytes[i] = tx.ID
		}
		if err := batch.Set(bytes.Join(DBPrefixToBytes(dbPrefixBlockIDToTxs), block.Header.ID), bytes.Join(idsBytes...)); err != nil {
			return err
		}
	}
	if len(events) > 0 {
		eventBytes, err := encodableListToBytes(events)
		if err != nil {
			return err
		}
		if err := batch.Set(bytes.Join(DBPrefixToBytes(dbPrefixBlockHeightToEvents), height), eventBytes); err != nil {
			return err
		}
	}
	if len(block.Assets) > 0 {
		assetBytes, err := encodableListToBytes(block.Assets)
		if err != nil {
			return err
		}
		if err := batch.Set(bytes.Join(DBPrefixToBytes(dbPrefixBlockIDToAssets), block.Header.ID), assetBytes); err != nil {
			return err
		}
	}
	if err := batch.Set(DBPrefixToBytes(dbPrefixFinalizedHeight), bytes.FromUint32(finalizedHeight)); err != nil {
		return err
	}
	if d.keepEventsForHeights > -1 {
		minEventDeleteHeight := ints.Min(
			int(finalizedHeight),
			ints.Max(0, int(block.Header.Height)-d.keepEventsForHeights),
		)
		if minEventDeleteHeight > 0 {
			kvs, err := d.database.IterateRange(
				bytes.Join(DBPrefixToBytes(dbPrefixBlockHeightToEvents), bytes.FromUint32(0)),
				bytes.Join(DBPrefixToBytes(dbPrefixBlockHeightToEvents), bytes.FromUint32(uint32(minEventDeleteHeight))),
				-1, false,
			)
			if err != nil {
				return err
			}
			for _, kv := range kvs {
				if err := batch.Del(kv.Key()); err != nil {
					return err
				}
			}
		}
	}
	if removeTemp {
		if err := batch.Del(bytes.Join(DBPrefixToBytes(dbPrefixTemp), height)); err != nil {
			return err
		}
	}
	return nil
}

func (d *DataAccess) removeBlock(batch *db.Batch, block *Block, saveTemp bool) error {
	height := bytes.FromUint32(block.Header.Height)
	encodedBlock, err := block.Encode()
	if err != nil {
		return err
	}
	if err := batch.Del(bytes.Join(DBPrefixToBytes(dbPrefixBlockIDToBlockHeader), block.Header.ID)); err != nil {
		return err
	}
	if err := batch.Del(bytes.Join(DBPrefixToBytes(dbPrefixBlockHeightToBlockID), height)); err != nil {
		return err
	}
	if len(block.Transactions) > 0 {
		for _, tx := range block.Transactions {
			if err := batch.Del(bytes.Join(DBPrefixToBytes(dbPrefixTxIDToTx), tx.ID)); err != nil {
				return err
			}
		}
		if err := batch.Del(bytes.Join(DBPrefixToBytes(dbPrefixBlockIDToTxs), block.Header.ID)); err != nil {
			return err
		}
	}
	if len(block.Assets) > 0 {
		if err := batch.Del(bytes.Join(DBPrefixToBytes(dbPrefixBlockIDToAssets), block.Header.ID)); err != nil {
			return err
		}
	}
	if err := batch.Del(bytes.Join(DBPrefixToBytes(dbPrefixBlockHeightToEvents), height)); err != nil {
		return err
	}
	if saveTemp {
		if err := batch.Set(bytes.Join(DBPrefixToBytes(dbPrefixTemp), height), encodedBlock); err != nil {
			return err
		}
	}
	return nil
}

type bytesList struct {
	items [][]byte `fieldNumber:"1"`
}

func encodableListToBytes[T codec.EncodeDecodable](assets []T) ([]byte, error) {
	items := make([][]byte, len(assets))
	for i, asset := range assets {
		encoded, err := asset.Encode()
		if err != nil {
			return nil, err
		}
		items[i] = encoded
	}
	list := &bytesList{
		items: items,
	}
	return list.Encode()
}

func bytesToAssets(data []byte) (BlockAssets, error) {
	if len(data) == 0 {
		return BlockAssets{}, nil
	}
	list := &bytesList{}
	if err := list.Decode(data); err != nil {
		return nil, err
	}
	result := make(BlockAssets, len(list.items))
	for i, d := range list.items {
		asset := &BlockAsset{}
		if err := asset.Decode(d); err != nil {
			return nil, err
		}
		result[i] = asset
	}
	return result, nil
}

func bytesToEvents(data []byte) ([]*Event, error) {
	if len(data) == 0 {
		return []*Event{}, nil
	}
	list := &bytesList{}
	if err := list.Decode(data); err != nil {
		return nil, err
	}
	result := make([]*Event, len(list.items))
	for i, item := range list.items {
		event, err := NewEvent(item)
		if err != nil {
			return nil, err
		}
		result[i] = event
	}
	return result, nil
}
