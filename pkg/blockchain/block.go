package blockchain

import (
	"errors"
	"fmt"
	"sort"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/trie/rmt"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

var (
	emptyHash     = crypto.Hash([]byte{})
	AddressLength = 20
	IDLength      = 32
)

// RawBlock represents a envelope of a block header.
type RawBlock struct {
	Header       []byte   `fieldNumber:"1"`
	Transactions [][]byte `fieldNumber:"2"`
	Assets       [][]byte `fieldNumber:"3"`
}

// Block is an envelope for header and payload.
type Block struct {
	Header       *BlockHeader   `fieldNumber:"1" json:"header"`
	Transactions []*Transaction `fieldNumber:"2" json:"transactions"`
	Assets       []*BlockAsset  `fieldNumber:"3" json:"assets"`
}

// NewBlock from val and assign ID.
func NewBlock(val []byte) (*Block, error) {
	raw := &RawBlock{}
	err := raw.DecodeStrict(val)
	if err != nil {
		return nil, err
	}
	header, err := NewBlockHeader(raw.Header)
	if err != nil {
		return nil, err
	}
	assets := make([]*BlockAsset, len(raw.Assets))
	for i, assetBytes := range raw.Assets {
		asset, err := NewBlockAsset(assetBytes)
		if err != nil {
			return nil, err
		}
		assets[i] = asset
	}
	txs := make([]*Transaction, len(raw.Transactions))
	for i, txBytes := range raw.Transactions {
		tx, err := NewTransaction(txBytes)
		if err != nil {
			return nil, err
		}
		txs[i] = tx
	}
	block := &Block{
		Header:       header,
		Assets:       assets,
		Transactions: txs,
	}
	return block, nil
}

// Init the calculated value if decoded directly without NewBlock.
func (b *Block) Init() {
	b.Header.Init()
	for _, tx := range b.Transactions {
		tx.Init()
	}
}

// Validate a block.
func (b *Block) Validate() error {
	if err := b.Header.Validate(); err != nil {
		return err
	}
	txIDs := make([][]byte, len(b.Transactions))
	for i, tx := range b.Transactions {
		txIDs[i] = tx.ID
	}
	transactionRoot := rmt.CalculateRoot(txIDs)
	if !bytes.Equal(b.Header.TransactionRoot, transactionRoot) {
		return errors.New("transaction root must match the payload")
	}
	assets := BlockAssets(b.Assets)
	if err := assets.Valid(); err != nil {
		return err
	}
	assetsRoot := assets.GetRoot()
	if !bytes.Equal(b.Header.AssetRoot, assetsRoot) {
		return errors.New("assets root must match the assets")
	}
	return nil
}

// ValidateGenesis validates block specific for genesis.
func (b *Block) ValidateGenesis() error {
	if err := b.Header.ValidateGenesis(); err != nil {
		return err
	}
	if len(b.Transactions) != 0 {
		return errors.New("genesis block payload must be 0")
	}
	assets := BlockAssets(b.Assets)
	if err := assets.Valid(); err != nil {
		return err
	}
	assetsRoot := assets.GetRoot()
	if !bytes.Equal(b.Header.AssetRoot, assetsRoot) {
		return errors.New("asset root must match the assets")
	}
	return nil
}

// BlockHeader holds general block header regardless of version.
type BlockHeader struct {
	ID                 codec.Hex        `json:"id"`
	Version            uint32           `json:"version" fieldNumber:"1"`
	Timestamp          uint32           `json:"timestamp" fieldNumber:"2"`
	Height             uint32           `json:"height" fieldNumber:"3"`
	PreviousBlockID    codec.Hex        `json:"previousBlockID" fieldNumber:"4"`
	GeneratorAddress   codec.Lisk32     `json:"generatorAddress" fieldNumber:"5"`
	TransactionRoot    codec.Hex        `json:"transactionRoot" fieldNumber:"6"`
	AssetRoot          codec.Hex        `json:"assetRoot" fieldNumber:"7"`
	EventRoot          codec.Hex        `json:"eventRoot" fieldNumber:"8"`
	StateRoot          codec.Hex        `json:"stateRoot" fieldNumber:"9"`
	MaxHeightPrevoted  uint32           `json:"maxHeightPrevoted" fieldNumber:"10"`
	MaxHeightGenerated uint32           `json:"maxHeightGenerated" fieldNumber:"11"`
	ValidatorsHash     codec.Hex        `json:"validatorsHash" fieldNumber:"12"`
	AggregateCommit    *AggregateCommit `json:"aggregateCommit" fieldNumber:"13"`
	Signature          codec.Hex        `json:"signature" fieldNumber:"14"`
}

type AggregateCommit struct {
	Height               uint32    `json:"height" fieldNumber:"1"`
	AggregationBits      codec.Hex `json:"aggregationBits" fieldNumber:"2"`
	CertificateSignature codec.Hex `json:"certificateSignature" fieldNumber:"3"`
}

func (c *AggregateCommit) Empty() bool {
	return len(c.AggregationBits) == 0 && len(c.CertificateSignature) == 0
}

func (b *BlockHeader) signingBlockHeader() *signingBlockHeader {
	return &signingBlockHeader{
		Version:            b.Version,
		Timestamp:          b.Timestamp,
		Height:             b.Height,
		PreviousBlockID:    b.PreviousBlockID,
		GeneratorAddress:   b.GeneratorAddress,
		EventRoot:          b.EventRoot,
		AssetRoot:          b.AssetRoot,
		StateRoot:          b.StateRoot,
		MaxHeightPrevoted:  b.MaxHeightPrevoted,
		MaxHeightGenerated: b.MaxHeightGenerated,
		ValidatorsHash:     b.ValidatorsHash,
		TransactionRoot:    b.TransactionRoot,
		AggregateCommit:    b.AggregateCommit,
	}
}

// SigningBytes return signed part of the block header.
func (b *BlockHeader) SigningBytes() []byte {
	return b.signingBlockHeader().Encode()
}

func (b *BlockHeader) Sign(chainID, privateKey []byte) {
	signature := b.signingBlockHeader().Sign(chainID, privateKey)
	b.Signature = signature
	headerBytes := b.Encode()
	id := crypto.Hash(headerBytes)
	b.ID = id
}

func (b *BlockHeader) VerifySignature(chainID, publicKey []byte) bool {
	signingBytes := b.SigningBytes()
	return ValidateBlockSignature(publicKey, b.Signature, chainID, signingBytes)
}

// NewBlockHeader create blockheader instance from encoded value.
func NewBlockHeader(value []byte) (*BlockHeader, error) {
	header := &BlockHeader{}
	if err := header.Decode(value); err != nil {
		return nil, err
	}
	headerBytes := header.Encode()
	id := crypto.Hash(headerBytes)
	header.ID = id
	return header, nil
}

func (b *BlockHeader) Init() {
	headerBytes := b.Encode()
	id := crypto.Hash(headerBytes)
	b.ID = id
}

func NewBlockHeaderWithValues(
	version uint32,
	timestamp uint32,
	height uint32,
	previousBlockID []byte,
	assetRoot []byte,
	stateRoot []byte,
	maxHeightPrevoted uint32,
	maxHeightPreviouslyForged uint32,
	transactionRoot []byte,
	generatorPublicKey []byte,
	validatorHash []byte,
	aggregateCommit *AggregateCommit,
	signature []byte,
) (*BlockHeader, error) {
	header := &BlockHeader{
		Version:            version,
		Timestamp:          timestamp,
		Height:             height,
		PreviousBlockID:    previousBlockID,
		GeneratorAddress:   generatorPublicKey,
		TransactionRoot:    transactionRoot,
		AssetRoot:          assetRoot,
		StateRoot:          stateRoot,
		MaxHeightPrevoted:  maxHeightPrevoted,
		MaxHeightGenerated: maxHeightPreviouslyForged,
		ValidatorsHash:     validatorHash,
		AggregateCommit:    aggregateCommit,
		Signature:          signature,
	}
	headerBytes := header.Encode()
	id := crypto.Hash(headerBytes)
	header.ID = id
	return header, nil
}

// ValidateGenesis validates block header specific for genesis.
func (b *BlockHeader) ValidateGenesis() error {
	if b.Version != 0 {
		return errors.New("genesis block version must be 0")
	}
	if !bytes.Equal(b.TransactionRoot, emptyHash) {
		return errors.New("genesis block transaction root must be empty hash")
	}
	if !bytes.Equal(b.GeneratorAddress, bytes.Repeat([]byte{0}, AddressLength)) {
		return errors.New("genesis block generator address must be zero address")
	}
	if len(b.Signature) != 0 {
		return errors.New("genesis block signature must be empty")
	}
	return nil
}

// Validate block header property.
func (b *BlockHeader) Validate() error {
	if len(b.PreviousBlockID) != crypto.HashLengh {
		return fmt.Errorf("previous block id must be 32 bytes but received %v", b.PreviousBlockID)
	}
	if len(b.GeneratorAddress) != AddressLength {
		return fmt.Errorf("generator address must be 20 bytes but received %v", b.GeneratorAddress)
	}
	if len(b.Signature) != crypto.EdSignatureLength {
		return errors.New("block signature must not be empty")
	}
	return nil
}

func (b *BlockHeader) Readonly() SealedBlockHeader {
	return &readonlyBlockHeader{
		header: b,
	}
}

type SealedBlockHeader interface {
	ReadableBlockHeader
	ID() []byte
	Signature() []byte
	SigningBytes() []byte
	TransactionRoot() []byte
	AssetRoot() []byte
	StateRoot() []byte
	ValidatorsHash() []byte
}

type readonlyBlockHeader struct {
	header *BlockHeader
}

// ID is getter for id.
func (b *readonlyBlockHeader) ID() []byte                 { return b.header.ID }
func (b *readonlyBlockHeader) Version() uint32            { return b.header.Version }
func (b *readonlyBlockHeader) Timestamp() uint32          { return b.header.Timestamp }
func (b *readonlyBlockHeader) Height() uint32             { return b.header.Height }
func (b *readonlyBlockHeader) PreviousBlockID() codec.Hex { return b.header.PreviousBlockID }
func (b *readonlyBlockHeader) GeneratorAddress() codec.Lisk32 {
	return b.header.GeneratorAddress
}
func (b *readonlyBlockHeader) TransactionRoot() []byte   { return b.header.TransactionRoot }
func (b *readonlyBlockHeader) AssetRoot() []byte         { return b.header.AssetRoot }
func (b *readonlyBlockHeader) EventRoot() []byte         { return b.header.EventRoot }
func (b *readonlyBlockHeader) StateRoot() []byte         { return b.header.StateRoot }
func (b *readonlyBlockHeader) MaxHeightPrevoted() uint32 { return b.header.MaxHeightPrevoted }
func (b *readonlyBlockHeader) MaxHeightGenerated() uint32 {
	return b.header.MaxHeightGenerated
}
func (b *readonlyBlockHeader) ValidatorsHash() []byte            { return b.header.ValidatorsHash }
func (b *readonlyBlockHeader) AggregateCommit() *AggregateCommit { return b.header.AggregateCommit }

func (b *readonlyBlockHeader) Signature() []byte    { return b.header.Signature }
func (b *readonlyBlockHeader) SigningBytes() []byte { return b.header.SigningBytes() }

// signingBlockHeader holds signing part of block header.
type signingBlockHeader struct {
	Version            uint32           `fieldNumber:"1"`
	Timestamp          uint32           `fieldNumber:"2"`
	Height             uint32           `fieldNumber:"3"`
	PreviousBlockID    codec.Hex        `fieldNumber:"4"`
	GeneratorAddress   codec.Lisk32     `fieldNumber:"5"`
	TransactionRoot    codec.Hex        `fieldNumber:"6"`
	AssetRoot          codec.Hex        `fieldNumber:"7"`
	EventRoot          codec.Hex        `fieldNumber:"8"`
	StateRoot          codec.Hex        `fieldNumber:"9"`
	MaxHeightPrevoted  uint32           `fieldNumber:"10"`
	MaxHeightGenerated uint32           `fieldNumber:"11"`
	ValidatorsHash     codec.Hex        `fieldNumber:"12"`
	AggregateCommit    *AggregateCommit `fieldNumber:"13"`
}

func (s *signingBlockHeader) Sign(chainID, privateKey []byte) []byte {
	encoded := s.Encode()
	signature := crypto.Sign(privateKey, crypto.Hash(bytes.Join(TagBlockHeader, chainID, encoded)))
	return signature
}

type BlockAsset struct {
	Module string    `fieldNumber:"1" json:"module"`
	Data   codec.Hex `fieldNumber:"2" json:"data"`
}

func NewBlockAsset(val []byte) (*BlockAsset, error) {
	asset := &BlockAsset{}
	if err := asset.DecodeStrict(val); err != nil {
		return nil, err
	}
	return asset, nil
}

type BlockAssets []*BlockAsset

func (a BlockAssets) Valid() error {
	copied := make(BlockAssets, len(a))
	copy(copied, a)
	copied.Sort()

	moduleMap := map[string]bool{}
	for i, asset := range a {
		if copied[i].Module != asset.Module {
			return errors.New("assets must be sorted by module")
		}
		if _, exist := moduleMap[asset.Module]; exist {
			return errors.New("assets module must be unique")
		}
		moduleMap[asset.Module] = true
	}

	return nil
}

func (a BlockAssets) GetAsset(module string) ([]byte, bool) {
	for _, asset := range a {
		if asset.Module == module {
			return asset.Data, true
		}
	}
	return nil, false
}

func (a *BlockAssets) Sort() {
	original := *a
	sort.Slice(original, func(i, j int) bool {
		return original[i].Module < original[j].Module
	})
	*a = original
}

func (a BlockAssets) GetRoot() []byte {
	assets := make([][]byte, len(a))
	for i, asset := range a {
		encoded := asset.Encode()
		assets[i] = encoded
	}

	return rmt.CalculateRoot(assets)
}

func (a *BlockAssets) SetAsset(module string, data []byte) {
	original := *a
	for i, asset := range original {
		if asset.Module == module {
			original[i].Data = data
			*a = original
			return
		}
	}
	original = append(original, &BlockAsset{
		Module: module,
		Data:   data,
	})
	*a = original
}

type ReadableBlockHeader interface {
	Version() uint32
	Height() uint32
	Timestamp() uint32
	PreviousBlockID() codec.Hex
	GeneratorAddress() codec.Lisk32
	AggregateCommit() *AggregateCommit
	MaxHeightPrevoted() uint32
	MaxHeightGenerated() uint32
}

type ReadableBlockAssets interface {
	GetAsset(module string) ([]byte, bool)
}

type WritableBlockAssets interface {
	ReadableBlockAssets
	SetAsset(module string, data []byte)
}
