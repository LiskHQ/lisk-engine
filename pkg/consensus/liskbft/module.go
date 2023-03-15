package liskbft

import (
	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/collection/ints"
	"github.com/LiskHQ/lisk-engine/pkg/db/diffdb"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

const (
	ModuleID uint32 = 11
)

var (
	// state store.
	storePrefixBFTParams     uint16 = 0x0000
	storePrefixGeneratorKeys uint16 = 0x4000
	storePrefixBFTVotes      uint16 = 0x8000
	emptyKey                        = []byte{}
)

func dbPrefix(storePrefix uint16) []byte {
	return bytes.Join(bytes.FromUint32(ModuleID), bytes.FromUint16(storePrefix))
}

type ModuleConfig struct {
	BatchSize int `json:"batchSize"`
}

func NewModule() *Module {
	return &Module{
		api:      &API{},
		endpoint: &Endpoint{},
	}
}

type Module struct {
	batchSize      int
	maxLengthBlock int
	api            *API
	endpoint       *Endpoint
}

func (m *Module) Init(batchSize int) error {
	m.batchSize = batchSize
	m.maxLengthBlock = 3 * m.batchSize
	m.endpoint.init(m.ID(), batchSize)
	m.api.init(m.ID(), batchSize)
	return nil
}

// Properties to satisfy state machine module.
func (m *Module) ID() uint32 {
	return ModuleID
}
func (m *Module) Name() string {
	return "bft"
}

func (m *Module) Endpoint() statemachine.Endpoint {
	return m.endpoint
}

func (m *Module) API() *API {
	return m.api
}

func (m *Module) InitGenesisState(blockHeader blockchain.SealedBlockHeader, diffStore *diffdb.Database) error {
	voteStore := diffStore.WithPrefix(dbPrefix(storePrefixBFTVotes))
	bftVotes := &BFTVotes{
		maxHeightPrevoted:        blockHeader.Height(),
		maxHeightPrecommited:     blockHeader.Height(),
		maxHeightCertified:       blockHeader.Height(),
		blockBFTInfos:            []*BFTBlockHeader{},
		activeValidatorsVoteInfo: []*ActiveValidator{},
	}
	diffdb.SetEncodable(voteStore, emptyKey, bftVotes)
	return nil
}

// AfterBlockApply applies BFT properties.
func (m *Module) BeforeTransactionsExecute(blockHeader blockchain.ReadableBlockHeader, diffStore *diffdb.Database) error {
	voteStore := diffStore.WithPrefix(dbPrefix(storePrefixBFTVotes))
	paramsStore := diffStore.WithPrefix(dbPrefix(storePrefixBFTParams))
	paramsCache := newBFTParamsCache(paramsStore)

	bftVotes := &BFTVotes{}
	if err := diffdb.GetDecodable(voteStore, emptyKey, bftVotes); err != nil {
		return err
	}
	if err := bftVotes.insertBlockBFTInfo(blockHeader, m.maxLengthBlock); err != nil {
		return err
	}
	if err := paramsCache.cache(bftVotes.blockBFTInfos[len(bftVotes.blockBFTInfos)-1].height, bftVotes.blockBFTInfos[0].height); err != nil {
		return err
	}

	if err := bftVotes.updatePrevotesPrecommits(paramsCache); err != nil {
		return err
	}
	if err := bftVotes.updateMaxHeightPrevoted(paramsCache); err != nil {
		return err
	}
	if err := bftVotes.updateMaxHeightPrecommitted(paramsCache); err != nil {
		return err
	}
	if err := bftVotes.updateMaxHeightCertified(blockHeader); err != nil {
		return err
	}
	// store updated bft votes
	diffdb.SetEncodable(voteStore, emptyKey, bftVotes)
	minHeightBFTParamsRequired := ints.Min(
		bftVotes.blockBFTInfos[len(bftVotes.blockBFTInfos)-1].height,
		bftVotes.maxHeightCertified+1,
	)

	if err := deleteBFTParams(paramsStore, minHeightBFTParamsRequired); err != nil {
		return err
	}
	keysStore := diffStore.WithPrefix(dbPrefix(storePrefixGeneratorKeys))
	if err := deleteGeneratorKeys(keysStore, minHeightBFTParamsRequired); err != nil {
		return err
	}
	return nil
}
