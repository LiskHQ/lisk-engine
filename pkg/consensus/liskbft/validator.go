package liskbft

import (
	"fmt"
	"sort"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/collection/ints"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/contradiction"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

// ActiveValidator holds information for a particular concenter.
type ActiveValidator struct {
	address                codec.Lisk32 `fieldNumber:"1"`
	minActiveHeight        uint32       `fieldNumber:"2"`
	largestHeightPrecommit uint32       `fieldNumber:"3"`
}

type ActiveValidators []*ActiveValidator

func (c *ActiveValidators) sort() {
	original := *c
	sort.Slice(original, func(i, j int) bool {
		return bytes.Compare(original[i].address, original[j].address) > 0
	})
	*c = original
}

func (c ActiveValidators) get(address []byte) (*ActiveValidator, bool) {
	for _, validator := range c {
		if bytes.Equal(validator.address, address) {
			return validator, true
		}
	}
	return nil, false
}

func NewValidator(address codec.Lisk32, bftWeight uint64, blsKey codec.Hex) *BFTValidator {
	return &BFTValidator{
		address:   address,
		bftWeight: bftWeight,
		blsKey:    blsKey,
	}
}

type BFTValidator struct {
	address   codec.Lisk32 `fieldNumber:"1"`
	bftWeight uint64       `fieldNumber:"2"`
	blsKey    codec.Hex    `fieldNumber:"3"`
}

func (v *BFTValidator) Address() codec.Lisk32 { return v.address }
func (v *BFTValidator) BFTWeight() uint64     { return v.bftWeight }
func (v *BFTValidator) BLSKey() codec.Hex     { return v.blsKey }

type BFTValidators []*BFTValidator

func (vs BFTValidators) Get(address []byte) (*BFTValidator, bool) {
	for _, state := range vs {
		if bytes.Equal(state.address, address) {
			return state, true
		}
	}
	return nil, false
}

func (vs BFTValidators) Find(address []byte) (*BFTValidator, bool) {
	for _, v := range vs {
		if bytes.Equal(v.Address(), address) {
			return v, true
		}
	}
	return nil, false
}

func (vs BFTValidators) Equal(validators BFTValidators) bool {
	if len(vs) != len(validators) {
		return false
	}
	for i := 0; i < len(vs); i++ {
		if !bytes.Equal(vs[i].Address(), validators[i].Address()) {
			return false
		}
		if vs[i].BFTWeight() != validators[i].BFTWeight() {
			return false
		}
	}
	return true
}

func (vs *BFTValidators) Sort() {
	original := *vs
	sort.Slice(original, func(i, j int) bool {
		return bytes.Compare(original[i].address, original[j].address) > 0
	})
	*vs = original
}

type BFTParams struct {
	prevoteThreshold     uint64          `fieldNumber:"1"`
	precommitThreshold   uint64          `fieldNumber:"2"`
	certificateThreshold uint64          `fieldNumber:"3"`
	validators           []*BFTValidator `fieldNumber:"4"`
	validatorsHash       codec.Hex       `fieldNumber:"5"`
}

func (c BFTParams) String() string {
	return fmt.Sprintf("Thresholds prevote: %d, precommit %d, certificate %d, validatorsHash %s", c.prevoteThreshold, c.precommitThreshold, c.certificateThreshold, c.validatorsHash)
}

func (c *BFTParams) PrevoteThreshold() uint64     { return c.prevoteThreshold }
func (c *BFTParams) PrecommitThreshold() uint64   { return c.precommitThreshold }
func (c *BFTParams) CertificateThreshold() uint64 { return c.certificateThreshold }
func (c *BFTParams) ValidatorsHash() codec.Hex    { return c.validatorsHash }
func (c *BFTParams) Validators() []*BFTValidator {
	return c.validators
}

type BFTBlockHeader struct {
	height             uint32       `fieldNumber:"1"`
	generatorAddress   codec.Lisk32 `fieldNumber:"2"`
	maxHeightGenerated uint32       `fieldNumber:"3"`
	maxHeightPrevoted  uint32       `fieldNumber:"4"`
	prevoteWeight      uint64       `fieldNumber:"5"`
	precommitWeight    uint64       `fieldNumber:"6"`
}

func (h *BFTBlockHeader) Height() uint32             { return h.height }
func (h *BFTBlockHeader) GeneratorAddress() []byte   { return h.generatorAddress }
func (h *BFTBlockHeader) MaxHeightGenerated() uint32 { return h.maxHeightGenerated }
func (h *BFTBlockHeader) MaxHeightPrevoted() uint32  { return h.maxHeightPrevoted }

type BFTBlockHeaders []*BFTBlockHeader

func (b BFTBlockHeaders) latest() *BFTBlockHeader {
	return b[0]
}

type BFTVotes struct {
	maxHeightPrevoted        uint32             `fieldNumber:"1"`
	maxHeightPrecommited     uint32             `fieldNumber:"2"`
	maxHeightCertified       uint32             `fieldNumber:"3"`
	blockBFTInfos            []*BFTBlockHeader  `fieldNumber:"5"`
	activeValidatorsVoteInfo []*ActiveValidator `fieldNumber:"6"`
}

func (v BFTVotes) String() string {
	oldest := v.blockBFTInfos[len(v.blockBFTInfos)-1]
	return fmt.Sprintf("maxPrevote %d maxPrecommited %d maxCertified %d. Oldest bft info: Prevote %d Precommit %d", v.maxHeightPrevoted, v.maxHeightPrecommited, v.maxHeightCertified, oldest.prevoteWeight, oldest.precommitWeight)
}

func (v *BFTVotes) insertBlockBFTInfo(header blockchain.ReadableBlockHeader, maxLength int) error {
	headers := make(BFTBlockHeaders, ints.Min(len(v.blockBFTInfos)+1, maxLength))
	headers[0] = &BFTBlockHeader{
		generatorAddress:   header.GeneratorAddress(),
		height:             header.Height(),
		maxHeightGenerated: header.MaxHeightGenerated(),
		maxHeightPrevoted:  header.MaxHeightPrevoted(),
		prevoteWeight:      0,
		precommitWeight:    0,
	}
	for i, bftInfo := range v.blockBFTInfos {
		if i+1 == maxLength {
			break
		}
		headers[i+1] = bftInfo
	}
	v.blockBFTInfos = headers
	return nil
}

func (v *BFTVotes) updatePrevotesPrecommits(paramsCache *bftParamsCache) error {
	if len(v.blockBFTInfos) == 0 {
		return nil
	}
	newBlockBFTInfo := v.blockBFTInfos[0]
	// a block header only implies votes if maxHeightGenerated < height
	if newBlockBFTInfo.maxHeightGenerated >= newBlockBFTInfo.height {
		return nil
	}
	validatorInfo, exist := ActiveValidators(v.activeValidatorsVoteInfo).get(newBlockBFTInfo.generatorAddress)
	if !exist {
		return nil
	}
	heightNotPrevoted := v.getHeightNotPrevoted()

	minPrecomimtHeight := ints.Max(
		validatorInfo.minActiveHeight,
		heightNotPrevoted+1,
		validatorInfo.largestHeightPrecommit+1,
	)
	hasPrecommitted := false
	for _, blockBFTInfo := range v.blockBFTInfos {
		if blockBFTInfo.height < minPrecomimtHeight {
			break
		}
		params, err := paramsCache.GetParameters(blockBFTInfo.height)
		if err != nil {
			return err
		}
		if blockBFTInfo.prevoteWeight >= params.prevoteThreshold {
			bftValidator, exist := BFTValidators(params.validators).Get(newBlockBFTInfo.generatorAddress)
			if !exist {
				return fmt.Errorf("invalid state. Validator %s does not exist in BFT parameters at height %d", blockBFTInfo.generatorAddress, blockBFTInfo.height)
			}
			blockBFTInfo.precommitWeight += bftValidator.bftWeight
			if !hasPrecommitted {
				for _, activeValidator := range v.activeValidatorsVoteInfo {
					if bytes.Equal(activeValidator.address, newBlockBFTInfo.generatorAddress) {
						activeValidator.largestHeightPrecommit = blockBFTInfo.height
						break
					}
				}
				hasPrecommitted = true
			}
		}
	}

	minPrevoteHeight := ints.Max(
		newBlockBFTInfo.maxHeightGenerated+1,
		validatorInfo.minActiveHeight,
	)

	for _, blockBFTInfo := range v.blockBFTInfos {
		if blockBFTInfo.height < minPrevoteHeight {
			break
		}
		params, err := paramsCache.GetParameters(blockBFTInfo.height)
		if err != nil {
			return err
		}
		bftValidator, exist := BFTValidators(params.validators).Get(newBlockBFTInfo.generatorAddress)
		if !exist {
			return fmt.Errorf("invalid state. Validator %s does not exist in BFT parameters at height %d", blockBFTInfo.generatorAddress, blockBFTInfo.height)
		}
		blockBFTInfo.prevoteWeight += bftValidator.bftWeight
	}
	return nil
}

func (v *BFTVotes) updateMaxHeightPrevoted(paramsCache *bftParamsCache) error {
	for _, blockBFTInfo := range v.blockBFTInfos {
		params, err := paramsCache.GetParameters(blockBFTInfo.height)
		if err != nil {
			return err
		}
		if blockBFTInfo.prevoteWeight >= params.prevoteThreshold {
			v.maxHeightPrevoted = blockBFTInfo.height
			return nil
		}
	}
	return nil
}

func (v *BFTVotes) updateMaxHeightPrecommitted(paramsCache *bftParamsCache) error {
	for _, blockBFTInfo := range v.blockBFTInfos {
		params, err := paramsCache.GetParameters(blockBFTInfo.height)
		if err != nil {
			return err
		}
		if blockBFTInfo.precommitWeight >= params.precommitThreshold {
			v.maxHeightPrecommited = blockBFTInfo.height
			return nil
		}
	}
	return nil
}

func (v *BFTVotes) updateMaxHeightCertified(header blockchain.ReadableBlockHeader) error {
	if len(header.AggregateCommit().AggregationBits) == 0 && len(header.AggregateCommit().CertificateSignature) == 0 {
		return nil
	}
	v.maxHeightCertified = header.AggregateCommit().Height
	return nil
}

func (v *BFTVotes) getHeightNotPrevoted() uint32 {
	if len(v.blockBFTInfos) == 0 {
		panic("at least one block header must exist to prevote or precommit")
	}
	newBlockBFTInfo := v.blockBFTInfos[0]
	currentHeight := newBlockBFTInfo.height
	heightPreviousBlock := newBlockBFTInfo.maxHeightGenerated
	for int(currentHeight)-int(heightPreviousBlock) < len(v.blockBFTInfos) {
		blockBFTInfo := v.blockBFTInfos[int(currentHeight-heightPreviousBlock)]
		if !bytes.Equal(blockBFTInfo.generatorAddress, newBlockBFTInfo.generatorAddress) || blockBFTInfo.maxHeightGenerated >= heightPreviousBlock {
			return heightPreviousBlock
		}
		heightPreviousBlock = blockBFTInfo.maxHeightGenerated
	}
	oldest := v.blockBFTInfos[len(v.blockBFTInfos)-1]
	return oldest.height - 1
}

func (v BFTVotes) contradicting(header contradiction.BFTPartialBlockHeader) bool {
	// bft headers are sorted height asc, loop from the tail
	if len(v.blockBFTInfos) == 0 {
		return false
	}
	for _, bftBlock := range v.blockBFTInfos {
		if bytes.Equal(bftBlock.generatorAddress, header.GeneratorAddress()) {
			bftHeader1 := contradiction.BFTPartialBlockHeader(bftBlock)
			return contradiction.AreDistinctHeadersContradicting(bftHeader1, header)
		}
	}
	return false
}

type HashValidator struct {
	BLSKey    []byte `fieldNumber:"1"`
	BFTWeight uint64 `fieldNumber:"2"`
}
type HashValidators []*HashValidator

type HashingValidators struct {
	ActiveValidators     []*HashValidator `fieldNumber:"1"`
	CertificateThreshold uint64           `fieldNumber:"2"`
}

func newBFTParamsCache(paramsStore statemachine.ImmutableStore) *bftParamsCache {
	return &bftParamsCache{
		paramsStore: paramsStore,
		data:        make(map[uint32]*BFTParams),
	}
}

type bftParamsCache struct {
	paramsStore statemachine.ImmutableStore
	data        map[uint32]*BFTParams
}

type KV struct {
	key   []byte
	value []byte
}

func (k *KV) Key() []byte   { return k.key }
func (k *KV) Value() []byte { return k.value }

func (c *bftParamsCache) cache(from, to uint32) error {
	start := bytes.FromUint32(from)
	end := bytes.FromUint32(to)
	kv := c.paramsStore.Range(start, end, -1, true)

	if from > 0 {
		nextLowest, err := getBFTParams(c.paramsStore, from)
		if err != nil {
			return err
		}
		encodedNextLowest, err := nextLowest.Encode()
		if err != nil {
			return err
		}
		kv = append(kv, &KV{
			key:   bytes.FromUint32(from),
			value: encodedNextLowest,
		})
	}
	for height := from; height <= to; height++ {
		var value []byte
		for _, element := range kv {
			keyHeight := bytes.ToUint32(element.Key())
			if keyHeight <= height {
				value = element.Value()
				break
			}
		}
		if value == nil {
			return fmt.Errorf("invalid state. BFT parameters should always exist")
		}
		params := &BFTParams{}
		if err := params.Decode(value); err != nil {
			return err
		}
		c.data[height] = params
	}
	return nil
}

func (c *bftParamsCache) GetParameters(height uint32) (*BFTParams, error) {
	cached, exist := c.data[height]
	if exist {
		return cached, nil
	}
	params, err := getBFTParams(c.paramsStore, height)
	if err != nil {
		return nil, err
	}
	c.data[height] = params
	return params, nil
}

type Generator struct {
	address      codec.Lisk32 `fieldNumber:"1"`
	generatorKey codec.Hex    `fieldNumber:"2"`
}

type Generators []*Generator

func NewGenerator(address codec.Lisk32, generatorKey codec.Hex) *Generator {
	return &Generator{
		address:      address,
		generatorKey: generatorKey,
	}
}

func (g *Generator) Address() codec.Lisk32   { return g.address }
func (g *Generator) GeneratorKey() codec.Hex { return g.generatorKey }

type SlotNumberGetter interface {
	GetSlotNumber(unixTime uint32) int
}

func (vs Generators) AtTimestamp(slot SlotNumberGetter, timestamp uint32) (*Generator, error) {
	currentSlot := slot.GetSlotNumber(timestamp)
	expectedValidator := vs[currentSlot%len(vs)]
	return expectedValidator, nil
}

type GeneratorKeys struct {
	Generators []*Generator `fieldNumber:"1"`
}
