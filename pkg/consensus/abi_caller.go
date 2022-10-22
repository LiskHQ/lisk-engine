package consensus

import (
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/liskbft"
	"github.com/LiskHQ/lisk-engine/pkg/db/diffdb"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
)

type stateExecuter struct {
	client    labi.ABI
	contextID codec.Hex
	bft       *liskbft.Module
	consensus *labi.Consensus
	events    blockchain.Events
}

func newBlockExecuteABI(client labi.ABI, bft *liskbft.Module, diffStore *diffdb.Database, header *blockchain.BlockHeader) (*stateExecuter, error) {
	res, err := client.InitStateMachine(&labi.InitStateMachineRequest{
		Header: header,
	})
	if err != nil {
		return nil, err
	}
	return &stateExecuter{
		client:    client,
		contextID: res.ContextID,
		bft:       bft,
		events:    []*blockchain.Event{},
	}, nil
}

func (c *stateExecuter) Verify(block *blockchain.Block) error {
	_, err := c.client.VerifyAssets(&labi.VerifyAssetsRequest{
		ContextID: c.contextID,
		Assets:    block.Assets,
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *stateExecuter) Execute(diffStore *diffdb.Database, block *blockchain.Block) (*NextValidatorParams, error) {
	if err := c.bft.BeforeTransactionsExecute(block.Header.Readonly(), diffStore); err != nil {
		return nil, err
	}
	consensus, err := getABIConsensus(c.bft, diffStore, block.Header.Readonly())
	if err != nil {
		return nil, err
	}
	c.consensus = consensus
	// Execute BFT
	beforeTxsResp, err := c.client.BeforeTransactionsExecute(&labi.BeforeTransactionsExecuteRequest{
		ContextID: c.contextID,
		Assets:    block.Assets,
		Consensus: c.consensus,
	})
	if err != nil {
		return nil, err
	}
	c.events = append(c.events, beforeTxsResp.Events...)
	for _, tx := range block.Transactions {
		txVerifyRes, err := c.client.VerifyTransaction(&labi.VerifyTransactionRequest{
			ContextID:   c.contextID,
			Transaction: tx,
		})
		if err != nil {
			return nil, err
		}
		if txVerifyRes.Result != labi.TxExecuteResultSuccess {
			return nil, fmt.Errorf("failed to execute transaction %s", tx.ID)
		}
		txResp, err := c.client.ExecuteTransaction(&labi.ExecuteTransactionRequest{
			ContextID:   c.contextID,
			Assets:      block.Assets,
			Header:      block.Header,
			Transaction: tx,
			DryRun:      false,
		})
		if err != nil {
			return nil, err
		}
		// Add transaction result event
		c.events = append(c.events, txResp.Events...)
	}
	resp, err := c.client.AfterTransactionsExecute(&labi.AfterTransactionsExecuteRequest{
		ContextID:    c.contextID,
		Assets:       block.Assets,
		Consensus:    c.consensus,
		Transactions: block.Transactions,
	})
	if err != nil {
		return nil, err
	}
	c.events = append(c.events, resp.Events...)
	c.events.UpdateIndex()

	if resp.PreCommitThreshold != 0 || resp.CertificateThreshold != 0 || len(resp.NextValidators) != 0 {
		bftValidators, generators := liskbft.GetBFTValidatorAndGenerators(resp.NextValidators)
		if err := c.bft.API().SetBFTParameters(diffStore, resp.PreCommitThreshold, resp.CertificateThreshold, bftValidators); err != nil {
			return nil, err
		}
		if err := c.bft.API().SetGeneratorKeys(diffStore, generators); err != nil {
			return nil, err
		}
		return &NextValidatorParams{
			NextValidators:       resp.NextValidators,
			PrecommitThreshold:   resp.PreCommitThreshold,
			CertificateThreshold: resp.CertificateThreshold,
		}, nil
	}

	return nil, nil
}

func (c *stateExecuter) Commit(currentStateRoot, expectedStateRoot codec.Hex) error {
	_, err := c.client.Commit(&labi.CommitRequest{
		ContextID:         c.contextID,
		StateRoot:         currentStateRoot,
		ExpectedStateRoot: expectedStateRoot,
		DryRun:            false,
	})
	return err
}

func (c *stateExecuter) Clear() error {
	_, err := c.client.Clear(&labi.ClearRequest{})
	return err
}

func (c *stateExecuter) Events() []*blockchain.Event {
	return c.events
}

type genesisStateExecuter struct {
	client    labi.ABI
	contextID codec.Hex
	bft       *liskbft.Module
	events    blockchain.Events
}

func newGenesisExecuteABI(client labi.ABI, bft *liskbft.Module, header *blockchain.BlockHeader) (*genesisStateExecuter, error) {
	res, err := client.InitStateMachine(&labi.InitStateMachineRequest{
		Header: header,
	})
	if err != nil {
		return nil, err
	}
	return &genesisStateExecuter{
		client:    client,
		contextID: res.ContextID,
		bft:       bft,
		events:    []*blockchain.Event{},
	}, nil
}

func (c *genesisStateExecuter) ExecuteGenesis(diffStore *diffdb.Database, block *blockchain.Block) error {
	if err := c.bft.InitGenesisState(block.Header.Readonly(), diffStore); err != nil {
		return err
	}
	resp, err := c.client.InitGenesisState(&labi.InitGenesisStateRequest{
		ContextID: c.contextID,
	})
	if err != nil {
		return err
	}
	validators, generators := liskbft.GetBFTValidatorAndGenerators(resp.NextValidators)
	if err := c.bft.API().SetBFTParameters(diffStore, resp.PreCommitThreshold, resp.CertificateThreshold, validators); err != nil {
		return err
	}
	if err := c.bft.API().SetGeneratorKeys(diffStore, generators); err != nil {
		return err
	}
	c.events = append(c.events, resp.Events...)
	c.events.UpdateIndex()
	return nil
}

func (c *genesisStateExecuter) Commit(expectedStateRoot codec.Hex) error {
	_, err := c.client.Commit(&labi.CommitRequest{
		ContextID:         c.contextID,
		StateRoot:         []byte{},
		ExpectedStateRoot: expectedStateRoot,
		DryRun:            false,
	})
	return err
}

func (c *genesisStateExecuter) Events() []*blockchain.Event {
	return c.events
}

func (c *genesisStateExecuter) Clear() error {
	_, err := c.client.Clear(&labi.ClearRequest{})
	return err
}

func getABIConsensus(bft *liskbft.Module, diffStore *diffdb.Database, header blockchain.ReadableBlockHeader) (*labi.Consensus, error) {
	params, err := bft.API().GetBFTParameters(diffStore, header.Height())
	if err != nil {
		return nil, err
	}
	generators, err := bft.API().GetGeneratorKeys(diffStore, header.Height())
	if err != nil {
		return nil, err
	}
	impliesMaxPrevote, err := bft.API().ImpliesMaximalPrevotes(diffStore, header)
	if err != nil {
		return nil, err
	}
	_, _, maxHeightCertified, err := bft.API().GetBFTHeights(diffStore)
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

type stateReverter struct {
	client    labi.ABI
	contextID codec.Hex
	header    *blockchain.BlockHeader
}

func newBlockRevertABI(client labi.ABI, diffStore *diffdb.Database, header *blockchain.BlockHeader) (*stateReverter, error) {
	res, err := client.InitStateMachine(&labi.InitStateMachineRequest{
		Header: header,
	})
	if err != nil {
		return nil, err
	}
	return &stateReverter{
		client:    client,
		header:    header,
		contextID: res.ContextID,
	}, nil
}

func (c *stateReverter) Revert(expectedStateRoot codec.Hex) error {
	_, err := c.client.Revert(&labi.RevertRequest{
		ContextID:         c.contextID,
		StateRoot:         c.header.StateRoot,
		ExpectedStateRoot: expectedStateRoot,
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *stateReverter) Clear() error {
	_, err := c.client.Clear(&labi.ClearRequest{})
	return err
}
