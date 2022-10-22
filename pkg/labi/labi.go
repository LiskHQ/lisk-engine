// Package labi defines ABI interface to communicate with Lisk Application.
package labi

import (
	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/trie/smt"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

type ABI interface {
	Init(req *InitRequest) (*InitResponse, error)
	InitStateMachine(req *InitStateMachineRequest) (*InitStateMachineResponse, error)
	InitGenesisState(req *InitGenesisStateRequest) (*InitGenesisStateResponse, error)
	InsertAssets(req *InsertAssetsRequest) (*InsertAssetsResponse, error)
	VerifyAssets(req *VerifyAssetsRequest) (*VerifyAssetsResponse, error)
	BeforeTransactionsExecute(req *BeforeTransactionsExecuteRequest) (*BeforeTransactionsExecuteResponse, error)
	AfterTransactionsExecute(req *AfterTransactionsExecuteRequest) (*AfterTransactionsExecuteResponse, error)
	VerifyTransaction(req *VerifyTransactionRequest) (*VerifyTransactionResponse, error)
	ExecuteTransaction(req *ExecuteTransactionRequest) (*ExecuteTransactionResponse, error)
	Commit(req *CommitRequest) (*CommitResponse, error)
	Revert(req *RevertRequest) (*RevertResponse, error)
	Clear(req *ClearRequest) (*ClearResponse, error)
	Finalize(req *FinalizeRequest) (*FinalizeResponse, error)
	GetMetadata(req *MetadataRequest) (*MetadataResponse, error)
	Query(req *QueryRequest) (*QueryResponse, error)
	Prove(req *ProveRequest) (*ProveResponse, error)
}

const (
	TxVeirfyResultInvalid  int32 = -1
	TxVeirfyResultPending  int32 = 0
	TxVeirfyResultOk       int32 = 1
	TxExecuteResultInvalid int32 = -1
	TxExecuteResultFail    int32 = 0
	TxExecuteResultSuccess int32 = 1
)

type InitRequest struct {
	ChainID         codec.Hex `json:"chainID" fieldNumber:"1"`
	LastBlockHeight uint32    `json:"lastBlockHeight" fieldNumber:"2"`
	LastStateRoot   []byte    `json:"lastStateRoot" fieldNumber:"3"`
}
type InitResponse struct {
}

type InitStateMachineRequest struct {
	Header *blockchain.BlockHeader `json:"header" fieldNumber:"1"`
}
type InitStateMachineResponse struct {
	ContextID codec.Hex `json:"contextID" fieldNumber:"1"`
}

type InitGenesisStateRequest struct {
	ContextID codec.Hex `json:"contextID" fieldNumber:"1"`
}

type InitGenesisStateResponse struct {
	Events               []*blockchain.Event `json:"events" fieldNumber:"1"`
	PreCommitThreshold   uint64              `json:"preCommitThreshold" fieldNumber:"2"`
	CertificateThreshold uint64              `json:"certificateThreshold" fieldNumber:"3"`
	NextValidators       []*Validator        `json:"nextValidators" fieldNumber:"4"`
}

type InsertAssetsRequest struct {
	ContextID       codec.Hex `json:"contextID" fieldNumber:"1"`
	FinalizedHeight uint32    `json:"finalizedHeight" fieldNumber:"2"`
}

type InsertAssetsResponse struct {
	Assets []*blockchain.BlockAsset `json:"assets" fieldNumber:"1"`
}

type VerifyAssetsRequest struct {
	ContextID codec.Hex                `json:"contextID" fieldNumber:"1"`
	Assets    []*blockchain.BlockAsset `json:"assets" fieldNumber:"2"`
}

type VerifyAssetsResponse struct {
}

type Validators []*Validator

type Validator struct {
	Address      codec.Lisk32 `json:"address" fieldNumber:"1"`
	BFTWeight    uint64       `json:"bftWeight" fieldNumber:"2"`
	GeneratorKey codec.Hex    `json:"generatorKey" fieldNumber:"3"`
	BLSKey       codec.Hex    `json:"blsKey" fieldNumber:"4"`
}

type Consensus struct {
	CurrentValidators    []*Validator `json:"currentValidators" fieldNumber:"1"`
	ImplyMaxPrevote      bool         `json:"implyMaxPrevote" fieldNumber:"2"`
	MaxHeightCertified   uint32       `json:"maxHeightCertified" fieldNumber:"3"`
	CertificateThreshold uint64       `json:"certificateThreshold" fieldNumber:"4"`
}

type BeforeTransactionsExecuteRequest struct {
	ContextID codec.Hex                `json:"contextID" fieldNumber:"1"`
	Assets    []*blockchain.BlockAsset `json:"assets" fieldNumber:"2"`
	Consensus *Consensus               `json:"consensus" fieldNumber:"3"`
}

type BeforeTransactionsExecuteResponse struct {
	Events []*blockchain.Event `json:"events" fieldNumber:"1"`
}

type AfterTransactionsExecuteRequest struct {
	ContextID    codec.Hex                 `json:"contextID" fieldNumber:"1"`
	Assets       []*blockchain.BlockAsset  `json:"assets" fieldNumber:"2"`
	Consensus    *Consensus                `json:"consensus" fieldNumber:"3"`
	Transactions []*blockchain.Transaction `json:"transactions" fieldNumber:"4"`
}

type AfterTransactionsExecuteResponse struct {
	Events               []*blockchain.Event `json:"events" fieldNumber:"1"`
	PreCommitThreshold   uint64              `json:"preCommitThreshold" fieldNumber:"2"`
	CertificateThreshold uint64              `json:"certificateThreshold" fieldNumber:"3"`
	NextValidators       []*Validator        `json:"nextValidators" fieldNumber:"4"`
}

type VerifyTransactionRequest struct {
	ContextID   codec.Hex               `json:"contextID" fieldNumber:"1"`
	Transaction *blockchain.Transaction `json:"transaction" fieldNumber:"2"`
}

type VerifyTransactionResponse struct {
	Result int32 `json:"result" fieldNumber:"1"`
}

type ExecuteTransactionRequest struct {
	ContextID   codec.Hex                `json:"contextID" fieldNumber:"1"`
	Transaction *blockchain.Transaction  `json:"transaction" fieldNumber:"2"`
	Assets      []*blockchain.BlockAsset `json:"assets" fieldNumber:"3"`
	DryRun      bool                     `json:"dryRun" fieldNumber:"4"`
	Header      *blockchain.BlockHeader  `json:"header" fieldNumber:"5"`
	Consensus   *Consensus               `json:"consensus" fieldNumber:"6"`
}

type ExecuteTransactionResponse struct {
	Events []*blockchain.Event `json:"events" fieldNumber:"1"`
	Result int32               `json:"result" fieldNumber:"2"`
}

type CommitRequest struct {
	ContextID         codec.Hex `json:"contextID" fieldNumber:"1"`
	StateRoot         codec.Hex `json:"stateRoot" fieldNumber:"2"`
	ExpectedStateRoot codec.Hex `json:"expectedStateRoot" fieldNumber:"3"`
	DryRun            bool      `json:"dryRun" fieldNumber:"4"`
}

type CommitResponse struct {
	StateRoot codec.Hex `json:"stateRoot" fieldNumber:"1"`
}

type RevertRequest struct {
	ContextID         codec.Hex `json:"contextID" fieldNumber:"1"`
	StateRoot         codec.Hex `json:"stateRoot" fieldNumber:"2"`
	ExpectedStateRoot codec.Hex `json:"expectedStateRoot" fieldNumber:"3"`
}

type RevertResponse struct {
	StateRoot codec.Hex `json:"stateRoot" fieldNumber:"1"`
}

type FinalizeRequest struct {
	FinalizedHeight uint32 `json:"finalizedHeight" fieldNumber:"1"`
}

type FinalizeResponse struct {
}

type ClearRequest struct {
}

type ClearResponse struct {
}

type MetadataRequest struct {
}

type MetadataResponse struct {
	Data []byte `json:"data" fieldNumber:"1"`
}

type QueryRequest struct {
	Method string                  `json:"method" fieldNumber:"1"`
	Params []byte                  `json:"params" fieldNumber:"2"`
	Header *blockchain.BlockHeader `json:"header" fieldNumber:"3"`
}

type QueryResponse struct {
	Data []byte `json:"data" fieldNumber:"1"`
}

type ProveRequest struct {
	StateRoot codec.Hex   `json:"stateRoot" fieldNumber:"1"`
	Keys      []codec.Hex `json:"keys" fieldNumber:"2"`
}

type ProveResponse struct {
	Result *smt.Proof `json:"proof" fieldNumber:"1"`
}
