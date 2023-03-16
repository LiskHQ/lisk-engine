package liskbft

import (
	"errors"
	"fmt"
	"math"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/contradiction"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/validator"
	"github.com/LiskHQ/lisk-engine/pkg/db/diffdb"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

type API struct {
	moduleID  uint32
	batchSize int
}

func (a *API) init(moduleID uint32, batchSize int) {
	a.moduleID = moduleID
	a.batchSize = batchSize
}

func (a *API) AreHeadersContradicting(header1, header2 blockchain.SealedBlockHeader) (bool, error) {
	if bytes.Equal(header1.ID(), header2.ID()) {
		return false, nil
	}
	bftHeader1 := contradiction.NewBFTBlockHeader(header1)
	bftHeader2 := contradiction.NewBFTBlockHeader(header2)
	return contradiction.AreDistinctHeadersContradicting(bftHeader1, bftHeader2), nil
}

func (a *API) IsHeaderContradictingChain(context *diffdb.Database, header blockchain.SealedBlockHeader) (bool, error) {
	bftVotesStore := context.WithPrefix(dbPrefix(storePrefixBFTVotes))
	bftVotes := &BFTVotes{}
	if err := diffdb.GetDecodable(bftVotesStore, emptyKey, bftVotes); err != nil {
		return true, err
	}
	return bftVotes.contradicting(contradiction.NewBFTBlockHeader(header)), nil
}

func (a *API) ExistBFTParameters(diffStore *diffdb.Database, height uint32) (bool, error) {
	paramsStore := diffStore.WithPrefix(dbPrefix(storePrefixBFTParams))
	_, exist := paramsStore.Get(bytes.FromUint32(height))
	if !exist {
		return false, nil
	}
	return true, nil
}

func (a *API) GetBFTParameters(diffStore *diffdb.Database, height uint32) (*BFTParams, error) {
	paramsStore := diffStore.WithPrefix(dbPrefix(storePrefixBFTParams))
	params, err := getBFTParams(paramsStore, height)
	if err != nil {
		return nil, err
	}
	return params, nil
}

func (a *API) GetBFTHeights(diffStore *diffdb.Database) (uint32, uint32, uint32, error) {
	bftVotesStore := diffStore.WithPrefix(dbPrefix(storePrefixBFTVotes))
	bftVotes := &BFTVotes{}
	if err := diffdb.GetDecodable(bftVotesStore, emptyKey, bftVotes); err != nil {
		return 0, 0, 0, err
	}
	return bftVotes.maxHeightPrevoted, bftVotes.maxHeightPrecommited, bftVotes.maxHeightCertified, nil
}

func (a *API) ImpliesMaximalPrevotes(context *diffdb.Database, blockHeader blockchain.ReadableBlockHeader) (bool, error) {
	bftVotes := &BFTVotes{}
	bftVotesStore := context.WithPrefix(dbPrefix(storePrefixBFTVotes))
	if err := diffdb.GetDecodable(bftVotesStore, emptyKey, bftVotes); err != nil {
		return false, err
	}
	currentHeight := BFTBlockHeaders(bftVotes.blockBFTInfos).latest().height
	if blockHeader.Height() != currentHeight {
		return false, fmt.Errorf("reward can be only checked for current height %d, but received %d", currentHeight, blockHeader.Height())
	}
	previousHeight := blockHeader.MaxHeightGenerated()
	if previousHeight >= blockHeader.Height() {
		return false, nil
	}
	if currentHeight < previousHeight {
		return false, fmt.Errorf("invalid heights. currentHeight: %d previousHeight %d", currentHeight, previousHeight)
	}
	offset := currentHeight - previousHeight - 1
	if int(offset) >= len(bftVotes.blockBFTInfos) {
		return true, nil
	}
	bftInfo := bftVotes.blockBFTInfos[offset]
	if !bytes.Equal(bftInfo.generatorAddress, blockHeader.GeneratorAddress()) {
		return false, nil
	}
	return true, nil
}

func (a *API) NextHeightBFTParameters(context *diffdb.Database, height uint32) (uint32, error) {
	paramsStore := context.WithPrefix(dbPrefix(storePrefixBFTParams))
	start := bytes.FromUint32(height + 1)
	end := bytes.FromUint32(math.MaxUint32)
	kv := paramsStore.Range(start, end, 1, false)

	if len(kv) != 1 {
		return 0, statemachine.ErrNotFound
	}
	return bytes.ToUint32(kv[0].Key()), nil
}

func (a *API) SetBFTParameters(
	diffStore *diffdb.Database,
	precommitThreshold, certificateThreshold uint64,
	validators BFTValidators,
) error {
	if len(validators) > a.batchSize {
		return fmt.Errorf("invalid validators size. The number of validators can be at most the batch size %d", a.batchSize)
	}

	aggregateBFTWeight := uint64(0)
	for _, validator := range validators {
		if validator.bftWeight <= 0 {
			return fmt.Errorf("invalid BFT weight for %s. BFTWeight must be a positive integer", validator.address)
		}
		aggregateBFTWeight += validator.bftWeight
	}
	if aggregateBFTWeight/3+1 > precommitThreshold || precommitThreshold > aggregateBFTWeight {
		return fmt.Errorf("invalid precommit threshold %d against aggregate BFT weight %d", precommitThreshold, aggregateBFTWeight)
	}
	if aggregateBFTWeight/3+1 > certificateThreshold || certificateThreshold > aggregateBFTWeight {
		return fmt.Errorf("invalid precommit threshold %d against aggregate BFT weight %d", precommitThreshold, aggregateBFTWeight)
	}

	validators.Sort()

	// get current vote
	bftVotesStore := diffStore.WithPrefix(dbPrefix(storePrefixBFTVotes))
	bftVotes := &BFTVotes{}
	if err := diffdb.GetDecodable(bftVotesStore, emptyKey, bftVotes); err != nil {
		return err
	}
	currentHeight := bftVotes.maxHeightPrevoted
	if len(bftVotes.blockBFTInfos) > 0 {
		currentHeight = bftVotes.blockBFTInfos[0].height
	}

	paramsStore := diffStore.WithPrefix(dbPrefix(storePrefixBFTParams))
	currentParams, err := getBFTParams(paramsStore, currentHeight)
	if err != nil && !errors.Is(err, ErrBFTParamsNotFound) {
		return err
	}
	// If there is no change in BFT params no update should be made
	if currentParams != nil &&
		BFTValidators(currentParams.validators).Equal(validators) &&
		currentParams.precommitThreshold == precommitThreshold &&
		currentParams.certificateThreshold == certificateThreshold {
		return nil
	}

	nextHeight := currentHeight + 1
	hashValidators := make(validator.HashValidators, len(validators))
	for i, validator := range validators {
		hashValidators[i] = validator
	}
	validatorsHash, err := validator.ComputeValidatorsHash(hashValidators, certificateThreshold)
	if err != nil {
		return err
	}

	params := &BFTParams{
		prevoteThreshold:     aggregateBFTWeight*2/3 + 1,
		precommitThreshold:   precommitThreshold,
		certificateThreshold: certificateThreshold,
		validators:           validators,
		validatorsHash:       validatorsHash,
	}

	nextHeightBytes := bytes.FromUint32(nextHeight)
	diffdb.SetEncodable(paramsStore, nextHeightBytes, params)

	currentActiveValidators := ActiveValidators(bftVotes.activeValidatorsVoteInfo)
	newActiveValidators := make(ActiveValidators, len(validators))
	for i, validator := range validators {
		if validator.bftWeight == 0 {
			continue
		}
		if originalConsenter, exist := currentActiveValidators.get(validator.address); exist {
			newActiveValidators[i] = originalConsenter
		} else {
			newActiveValidators[i] = &ActiveValidator{
				address:                validator.address,
				minActiveHeight:        nextHeight,
				largestHeightPrecommit: nextHeight - 1,
			}
		}
	}
	newActiveValidators.sort()
	bftVotes.activeValidatorsVoteInfo = newActiveValidators
	diffdb.SetEncodable(bftVotesStore, emptyKey, bftVotes)
	return nil
}

func (a *API) SetGeneratorKeys(diffStore *diffdb.Database, generators Generators) error {
	bftVotesStore := diffStore.WithPrefix(dbPrefix(storePrefixBFTVotes))
	bftVotes := &BFTVotes{}
	if err := diffdb.GetDecodable(bftVotesStore, emptyKey, bftVotes); err != nil {
		return err
	}
	nextHeight := bftVotes.maxHeightPrevoted + 1
	if len(bftVotes.blockBFTInfos) > 0 {
		nextHeight = bftVotes.blockBFTInfos[0].height + 1
	}
	keysStore := diffStore.WithPrefix(dbPrefix(storePrefixGeneratorKeys))
	diffdb.SetEncodable(keysStore, bytes.FromUint32(nextHeight), &GeneratorKeys{Generators: generators})
	return nil
}

func (a *API) GetGeneratorKeys(diffStore *diffdb.Database, height uint32) (Generators, error) {
	keysStore := diffStore.WithPrefix(dbPrefix(storePrefixGeneratorKeys))
	keys, err := getGeneratorKeys(keysStore, height)
	if err != nil {
		return nil, err
	}
	return keys.Generators, nil
}

func (a *API) HeaderHasPriority(diffStore *diffdb.Database, header blockchain.SealedBlockHeader, height, maxHeightPrevoted, maxHeightPreviouslyForged uint32) (bool, error) {
	if header.Version() == 0 {
		return height <= header.Height() && maxHeightPrevoted <= header.Height(), nil
	}
	return (maxHeightPrevoted < header.MaxHeightPrevoted() || (maxHeightPrevoted == header.MaxHeightPrevoted() && height < header.Height())), nil
}

func (a *API) GetValidator(context *diffdb.Database, address codec.Lisk32, height uint32) (*BFTValidator, error) {
	paramsStore := context.WithPrefix(dbPrefix(storePrefixBFTParams))
	params, err := getBFTParams(paramsStore, height)
	if err != nil {
		return nil, err
	}
	for _, validator := range params.validators {
		if bytes.Equal(validator.address, address) {
			return validator, nil
		}
	}
	return nil, fmt.Errorf("address %s is not validator at height %d", address, height)
}

func (a *API) GetCurrentValidators(context *diffdb.Database) (BFTValidators, error) {
	bftVotesStore := context.WithPrefix(dbPrefix(storePrefixBFTVotes))
	bftVotes := &BFTVotes{}
	if err := diffdb.GetDecodable(bftVotesStore, emptyKey, bftVotes); err != nil {
		return nil, err
	}
	if len(bftVotes.blockBFTInfos) == 0 {
		return nil, ErrBFTParamsNotFound
	}
	currentHeight := BFTBlockHeaders(bftVotes.blockBFTInfos).latest().height
	paramsStore := context.WithPrefix(dbPrefix(storePrefixBFTParams))
	params, err := getBFTParams(paramsStore, currentHeight)
	if err != nil {
		return nil, err
	}
	return params.validators, nil
}
