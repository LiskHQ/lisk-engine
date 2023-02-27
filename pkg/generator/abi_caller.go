package generator

import (
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/liskbft"
	"github.com/LiskHQ/lisk-engine/pkg/db/diffdb"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
)

type stateExecuter struct {
	client       labi.ABI
	contextID    codec.Hex
	consensus    Consensus
	chainID      codec.Hex
	events       blockchain.Events
	abiConsensus *labi.Consensus
}

func newBlockExecuteABI(client labi.ABI, chainID codec.Hex, diffStore *diffdb.Database, consensus Consensus, header *blockchain.BlockHeader) (*stateExecuter, error) {
	res, err := client.InitStateMachine(&labi.InitStateMachineRequest{
		Header: header,
	})
	if err != nil {
		return nil, err
	}
	return &stateExecuter{
		client:    client,
		contextID: res.ContextID,
		chainID:   chainID,
		consensus: consensus,
		events:    []*blockchain.Event{},
	}, nil
}

func (c *stateExecuter) InsertAssets() (blockchain.BlockAssets, error) {
	res, err := c.client.InsertAssets(&labi.InsertAssetsRequest{
		ContextID: c.contextID,
	})
	if err != nil {
		return nil, err
	}
	return res.Assets, nil
}

func (c *stateExecuter) BeforeTransactionsExecute(diffStore *diffdb.Database, header blockchain.SealedBlockHeader, assets blockchain.BlockAssets) error {
	if err := c.consensus.BFTBeforeTransactionsExecute(header, diffStore); err != nil {
		return err
	}
	abiConsensus, err := getABIConsensus(c.consensus, diffStore, header)
	if err != nil {
		return err
	}
	c.abiConsensus = abiConsensus

	resp, err := c.client.BeforeTransactionsExecute(&labi.BeforeTransactionsExecuteRequest{
		ContextID: c.contextID,
		Assets:    assets,
		Consensus: c.abiConsensus,
	})
	if err != nil {
		return err
	}
	c.events = append(c.events, resp.Events...)
	return nil
}

func (c *stateExecuter) VerifyTransaction(tx *blockchain.Transaction) error {
	res, err := c.client.VerifyTransaction(&labi.VerifyTransactionRequest{
		ContextID:   c.contextID,
		Transaction: tx,
	})
	if err != nil {
		return err
	}
	if res.Result != labi.TxVerifyResultOk {
		return fmt.Errorf("failed to verify transaction %s", tx.ID)
	}
	return nil
}

func (c *stateExecuter) ExecuteTransaction(header *blockchain.BlockHeader, assets blockchain.BlockAssets, tx *blockchain.Transaction) error {
	res, err := c.client.ExecuteTransaction(&labi.ExecuteTransactionRequest{
		ContextID:   c.contextID,
		Transaction: tx,
		Header:      header,
		Assets:      assets,
		DryRun:      false,
	})
	if err != nil {
		return err
	}
	if res.Result == labi.TxExecuteResultInvalid {
		return fmt.Errorf("invalid transaction %s", tx.ID)
	}
	c.events = append(c.events, res.Events...)
	return nil
}

func (c *stateExecuter) AfterTransactionsExecute(assets blockchain.BlockAssets, transactions []*blockchain.Transaction) error {
	resp, err := c.client.AfterTransactionsExecute(&labi.AfterTransactionsExecuteRequest{
		ContextID:    c.contextID,
		Assets:       assets,
		Consensus:    c.abiConsensus,
		Transactions: transactions,
	})
	if err != nil {
		return err
	}
	// handle execute result
	c.events = append(c.events, resp.Events...)
	return nil
}

func (c *stateExecuter) Commit(prevStateRoot codec.Hex) (codec.Hex, error) {
	res, err := c.client.Commit(&labi.CommitRequest{
		ContextID: c.contextID,
		StateRoot: prevStateRoot,
		DryRun:    true,
	})
	if err != nil {
		return nil, err
	}

	return res.StateRoot, err
}

func (c *stateExecuter) Clear() error {
	_, err := c.client.Clear(&labi.ClearRequest{})
	return err
}

func (c *stateExecuter) Events() []*blockchain.Event {
	c.events.UpdateIndex()
	return c.events
}

func getABIConsensus(consensus Consensus, diffStore *diffdb.Database, header blockchain.ReadableBlockHeader) (*labi.Consensus, error) {
	params, err := consensus.GetBFTParameters(diffStore, header.Height())
	if err != nil {
		return nil, err
	}
	generators, err := consensus.GetGeneratorKeys(diffStore, header.Height())
	if err != nil {
		return nil, err
	}
	impliesMaxPrevote, err := consensus.ImpliesMaximalPrevotes(diffStore, header)
	if err != nil {
		return nil, err
	}
	_, _, maxHeightCertified, err := consensus.GetBFTHeights(diffStore)
	if err != nil {
		return nil, err
	}
	validators := liskbft.GetLabiValidators(params.Validators(), generators)
	return &labi.Consensus{
		CurrentValidators:    validators,
		ImplyMaxPrevote:      impliesMaxPrevote,
		MaxHeightCertified:   maxHeightCertified,
		CertificateThreshold: params.CertificateThreshold(),
	}, nil
}
