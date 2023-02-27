package consensus

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/collection/ints"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/certificate"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/liskbft"
	"github.com/LiskHQ/lisk-engine/pkg/db/diffdb"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

func (c *Executer) singleCommitValidator(ctx context.Context, msg *p2p.Message) p2p.ValidationResult {
	postCommits := &EventPostSingleCommits{}
	if err := postCommits.DecodeStrict(msg.Data); err != nil {
		return p2p.ValidationReject
	}
	return p2p.ValidationAccept
}

func (c *Executer) onSingleCommitsReceived(event *p2p.Event) {
	postCommits := &EventPostSingleCommits{}
	if err := postCommits.DecodeStrict(event.Data()); err != nil {
		panic(err)
	}
	diffStore := diffdb.New(c.database, blockchain.DBPrefixToBytes(blockchain.DBPrefixState))
	for _, singleCommit := range postCommits.SingleCommits {
		// 1. if certificate already exist in pool, discard
		if c.certificatePool.Has(singleCommit) {
			continue
		}

		// 2. Discard m if m.height <= getMaxRemovalHeight()
		_, maxHeightPrecommited, _, err := c.liskBFT.API().GetBFTHeights(diffStore)
		if err != nil {
			c.logger.Errorf("Failed to get votes state while processing single commits with %v", err)
			return
		}
		finalizedBlockHeader, err := c.chain.DataAccess().GetBlockHeaderByHeight(maxHeightPrecommited)
		if err != nil {
			c.logger.Errorf("Failed to get votes state while processing single commits with %v", err)
			return
		}
		if singleCommit.Height() <= certificate.GetMaxRemovalHeight(finalizedBlockHeader) {
			continue
		}

		// 3. (height < finalizedHeight - 100  or height > 100) && height % (len(validatorAPI.getValidators())) != 0, discard
		paramExist, err := c.liskBFT.API().ExistBFTParameters(diffStore, singleCommit.Height())
		if err != nil {
			c.logger.Errorf("Failed to get ExistBFTParameters while processing single commits with %v", err)
			return
		}
		if (singleCommit.Height() < maxHeightPrecommited-certificate.CommitRangeStored || singleCommit.Height() > maxHeightPrecommited) &&
			!paramExist {
			continue
		}

		// 4. if m.blockID is not the ID of the block b in the current chain at height m.height
		blockHeader, err := c.chain.DataAccess().GetBlockHeaderByHeight(singleCommit.Height())
		if err != nil {
			c.logger.Errorf("Failed to get block with id %s while processing single commits with %v", singleCommit.BlockID(), err)
			return
		}
		if !bytes.Equal(blockHeader.ID, singleCommit.BlockID()) {
			continue
		}
		// 5. if validatorAddress is not active at the height, discard and ban (bftAPI.getBFTParams(height))
		params, err := c.liskBFT.API().GetBFTParameters(diffStore, singleCommit.Height())
		if err != nil {
			return
		}
		_, active := liskbft.BFTValidators(params.Validators()).Find(singleCommit.ValidatorAddress())
		if !active {
			c.conn.ApplyPenalty(event.PeerID(), p2p.MaxScore)
			c.logger.Errorf("Address %s was not active at height %d. Applying penalty to %s.", blockHeader.GeneratorAddress, singleCommit.Height(), event.PeerID())
			return
		}
		// 6. check signature of certificate obtained by the block at the height, if invalid discard and ban
		cert := certificate.NewCertificateFromBlock(blockHeader)
		validator, exist := liskbft.BFTValidators(params.Validators()).Find(singleCommit.ValidatorAddress())
		if !exist {
			c.logger.Errorf("Failed to get registered keys for %s while processing single commits with %v", singleCommit.ValidatorAddress(), err)
			return
		}
		validCert, err := cert.Verify(c.chain.ChainID(), singleCommit.CertificateSignature(), validator.BLSKey())
		if err != nil {
			c.logger.Errorf("Failed to verify signature while processing single commits with %v", err)
			return
		}
		if !validCert {
			c.conn.ApplyPenalty(event.PeerID(), p2p.MaxScore)
			c.logger.Errorf("Invalid commit received. Applying penalty to %s.", event.PeerID())
			return
		}
		// 7. add certificate
		c.certificatePool.Add(singleCommit)
	}
}

func (c *Executer) Certify(from, to uint32, address codec.Lisk32, blsPrivateKey []byte) error {
	if from > to {
		return fmt.Errorf("invalid range to certify from %d to %d", from, to)
	}
	diffStore := diffdb.New(c.database, blockchain.DBPrefixToBytes(blockchain.DBPrefixState))
	// h is the height of a block authenticating a change of BFT parameters
	eg, _ := errgroup.WithContext(c.ctx)
	for i := from + 1; i <= to; i++ {
		i := i
		eg.Go(func() error {
			exist, err := c.liskBFT.API().ExistBFTParameters(diffStore, i)
			if err != nil {
				return err
			}
			if !exist {
				return nil
			}
			params, err := c.liskBFT.API().GetBFTParameters(diffStore, i)
			if err != nil {
				return err
			}
			_, active := liskbft.BFTValidators(params.Validators()).Find(address)
			if !active {
				return nil
			}
			header, err := c.chain.DataAccess().GetBlockHeaderByHeight(i)
			if err != nil {
				return err
			}
			c.logger.Debugf("certifying block at hegiht %d", i)
			singleCommit, err := certificate.NewSingleCommit(header, address, c.chain.ChainID(), blsPrivateKey)
			if err != nil {
				return err
			}
			c.certificatePool.Add(singleCommit)
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	//  h2 is not the height of a block authenticating a change of BFT parameters
	exist, err := c.liskBFT.API().ExistBFTParameters(diffStore, to+1)
	if err != nil {
		return err
	}
	if exist {
		return nil
	}
	params, err := c.liskBFT.API().GetBFTParameters(diffStore, to)
	if err != nil {
		return err
	}
	_, active := liskbft.BFTValidators(params.Validators()).Find(address)
	if !active {
		return nil
	}
	header, err := c.chain.DataAccess().GetBlockHeaderByHeight(to)
	if err != nil {
		return err
	}
	c.logger.Debugf("certifying block at hegiht %d", to)
	singleCommit, err := certificate.NewSingleCommit(header, address, c.chain.ChainID(), blsPrivateKey)
	if err != nil {
		return err
	}
	c.certificatePool.Add(singleCommit)
	return nil
}

func (c *Executer) broadcastCertificate() error {
	diffStore := diffdb.New(c.database, blockchain.DBPrefixToBytes(blockchain.DBPrefixState))
	_, maxHeightPrecommited, _, err := c.liskBFT.API().GetBFTHeights(diffStore)
	if err != nil {
		return err
	}
	finalizedBlockHeader, err := c.chain.DataAccess().GetBlockHeaderByHeight(maxHeightPrecommited)
	if err != nil {
		c.logger.Errorf("Failed to get votes state while processing single commits with %v", err)
		return err
	}
	// 1. cleanup pool (finalizedHeight, len(validatorAPI.getValidators()))
	removeHeight := certificate.GetMaxRemovalHeight(finalizedBlockHeader)
	cache := newCommitDiscard(c.liskBFT.API())
	c.certificatePool.Cleanup(func(h uint32) bool {
		if h <= removeHeight {
			return false
		}
		exist, err := cache.exist(diffStore, h+1)
		if err != nil {
			c.logger.Errorf("Failed to call ExistBFTParameters while processing single commits with %v", err)
			return false
		}
		if !(h >= maxHeightPrecommited-certificate.CommitRangeStored && h < maxHeightPrecommited) && !exist {
			return false
		}
		return true
	})
	if c.certificatePool.Size() == 0 {
		return nil
	}
	// 2. select from pool(finalizedHeight, len(liskBFT.getValidators()))
	currentHeight := c.chain.LastBlock().Header.Height
	params, err := c.liskBFT.API().GetBFTParameters(diffStore, currentHeight)
	if err != nil {
		c.logger.Errorf("Failed to call GetBFTParameters while processing single commits with %v", err)
		return err
	}
	numActiveValidators := len(params.Validators())
	selectedCommites := c.certificatePool.Select(maxHeightPrecommited, numActiveValidators)
	if len(selectedCommites) == 0 {
		return nil
	}
	// 3. send to the network
	postCertificateData := &EventPostSingleCommits{
		SingleCommits: selectedCommites,
	}
	data, err := postCertificateData.Encode()
	if err != nil {
		return err
	}
	err = c.conn.Publish(c.ctx, P2PEventPostSingleCommits, data)
	if err != nil {
		return err
	}
	c.certificatePool.Upgrade(selectedCommites)
	return nil
}

func (c *Executer) verifyAggregateCommit(diffStore *diffdb.Database, aggregteCommit *blockchain.AggregateCommit) error {
	_, maxHeightPrecommited, maxHeightCertified, err := c.liskBFT.API().GetBFTHeights(diffStore)
	if err != nil {
		return err
	}
	if aggregteCommit.Empty() && aggregteCommit.Height == maxHeightCertified {
		return nil
	}
	if len(aggregteCommit.AggregationBits) == 0 || len(aggregteCommit.CertificateSignature) == 0 {
		return fmt.Errorf("aggregate commit aggregation bits or signature is empty")
	}
	if aggregteCommit.Height <= maxHeightCertified {
		return fmt.Errorf("aggregate commit height must be strictly increasing. MaxHeightCertified %d actual %d", maxHeightCertified, aggregteCommit.Height)
	}
	if aggregteCommit.Height > maxHeightPrecommited {
		return fmt.Errorf("aggregate commit has height %d higher than maxHeightPrecommited %d", aggregteCommit.Height, maxHeightPrecommited)
	}
	heightNextBFTParams, err := c.liskBFT.API().NextHeightBFTParameters(diffStore, maxHeightCertified+1)
	if err != nil && !errors.Is(err, statemachine.ErrNotFound) {
		return err
	}
	if err != nil && aggregteCommit.Height > heightNextBFTParams-1 {
		return fmt.Errorf("aggregate commit has height %d higher than next BFT params %d. It must include new aggregate commit", aggregteCommit.Height, heightNextBFTParams)
	}
	header, err := c.chain.DataAccess().GetBlockHeaderByHeight(aggregteCommit.Height)
	if err != nil {
		return err
	}
	certificateFromBlock := certificate.NewCertificateFromBlock(header)
	certificateFromBlock.AggregationBits = aggregteCommit.AggregationBits
	certificateFromBlock.Signature = aggregteCommit.CertificateSignature
	params, err := c.liskBFT.API().GetBFTParameters(diffStore, aggregteCommit.Height)
	if err != nil {
		return err
	}
	validatorBLSKeys := make(ValidatorsWithBLSKey, len(params.Validators()))
	for i, val := range params.Validators() {
		validatorBLSKeys[i] = &ValidatorWithBLSKey{
			Address:   val.Address(),
			BFTWeight: val.BFTWeight(),
			BLSKey:    val.BLSKey(),
		}
	}
	validatorBLSKeys.sort()
	keys := make([][]byte, len(validatorBLSKeys))
	weights := make([]uint64, len(validatorBLSKeys))
	for i, val := range validatorBLSKeys {
		keys[i] = val.BLSKey
		weights[i] = val.BFTWeight
	}
	valid, err := certificateFromBlock.VerifyAggregateCertificateSignature(keys, weights, params.CertificateThreshold(), c.chain.ChainID())
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("invalid certificate received")
	}
	return nil
}

func (c *Executer) GetAggregateCommit() (*blockchain.AggregateCommit, error) {
	diffStore := diffdb.New(c.database, blockchain.DBPrefixToBytes(blockchain.DBPrefixState))
	_, maxHeightPrecommited, maxHeightCertified, err := c.liskBFT.API().GetBFTHeights(diffStore)
	if err != nil {
		return nil, err
	}
	heightNextBFTParams, err := c.liskBFT.API().NextHeightBFTParameters(diffStore, maxHeightCertified+1)
	if err != nil {
		if !errors.Is(err, statemachine.ErrNotFound) {
			return nil, err
		}
	}
	var nextHeight uint32
	if err == nil {
		nextHeight = ints.Min(heightNextBFTParams-1, maxHeightPrecommited)
	} else {
		nextHeight = maxHeightPrecommited
	}

	for nextHeight > maxHeightCertified {
		commits := c.certificatePool.Get(nextHeight)
		if len(commits) == 0 {
			nextHeight -= 1
			continue
		}
		params, err := c.liskBFT.API().GetBFTParameters(diffStore, nextHeight)
		if err != nil {
			return nil, err
		}
		aggregateBFTWeight := uint64(0)
		for _, commit := range commits {
			commitValidator, exist := liskbft.BFTValidators(params.Validators()).Find(commit.ValidatorAddress())
			if !exist {
				panic("Validator address must exist in params")
			}
			aggregateBFTWeight += commitValidator.BFTWeight()
		}
		if aggregateBFTWeight < params.CertificateThreshold() {
			nextHeight -= 1
			continue
		}
		keypairs := make(certificate.AddressKeyPairs, len(params.Validators()))

		for i, validator := range params.Validators() {
			keypairs[i] = &certificate.AddressKeyPair{
				Address: validator.Address(),
				BLSKey:  validator.BLSKey(),
			}
		}
		return commits.Aggregate(keypairs)
	}
	return &blockchain.AggregateCommit{
		Height:               maxHeightCertified,
		AggregationBits:      codec.Hex{},
		CertificateSignature: codec.Hex{},
	}, nil
}

type commitDiscard struct {
	existHeight map[uint32]bool
	bftAPI      *liskbft.API
	mutex       *sync.RWMutex
}

func (c *commitDiscard) exist(diffStore *diffdb.Database, h uint32) (bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	val, exist := c.existHeight[h]
	if exist {
		return val, nil
	}
	exist, err := c.bftAPI.ExistBFTParameters(diffStore, h)
	if err != nil {
		return false, err
	}
	c.existHeight[h] = exist
	return exist, nil
}

func newCommitDiscard(bftAPI *liskbft.API) *commitDiscard {
	return &commitDiscard{
		existHeight: map[uint32]bool{},
		mutex:       new(sync.RWMutex),
		bftAPI:      bftAPI,
	}
}
