package statemachine

import (
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
)

var (
	// ErrNotFound is returned when data do not exist in the underline database.
	ErrNotFound = db.ErrDataNotFound
)

type VerifyStatus int32
type ExecStatus int32

const (
	verifyStatusOK      VerifyStatus = VerifyStatus(labi.TxVerifyResultOk)
	verifyStatusError   VerifyStatus = VerifyStatus(labi.TxVerifyResultInvalid)
	verifyStatusPending VerifyStatus = VerifyStatus(labi.TxVerifyResultPending)
	execStatusInvalid   ExecStatus   = ExecStatus(labi.TxExecuteResultInvalid)
	execStatusFail      ExecStatus   = ExecStatus(labi.TxExecuteResultFail)
	execStatusOK        ExecStatus   = ExecStatus(labi.TxExecuteResultSuccess)
)

type StateStore interface {
	Snapshot() int
	RestoreSnapshot(id int) error
	WithPrefix([]byte) Store
}

type Store interface {
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
	Set(key, value []byte) error
	Del(key []byte) error
	Iterate(prefix []byte, limit int, reverse bool) ([]db.KeyValue, error)
	Range(start, end []byte, limit int, reverse bool) ([]db.KeyValue, error)
	Snapshot() int
	RestoreSnapshot(id int) error
}

type ImmutableStore interface {
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
	Range(start, end []byte, limit int, reverse bool) ([]db.KeyValue, error)
	Iterate(prefix []byte, limit int, reverse bool) ([]db.KeyValue, error)
}

type Command interface {
	ID() uint32
	Name() string

	Verify(ctx *TransactionVerifyContext) VerifyResult
	Execute(ctx *TransactionExecuteContext) error
}

type Module interface {
	Name() string
	// hooks
	InitGenesisState(ctx *GenesisBlockProcessingContext) error
	FinalizeGenesisState(ctx *GenesisBlockProcessingContext) error
	InsertAssets(ctx *InsertAssetsContext) error
	VerifyAssets(ctx *VerifyAssetsContext) error
	VerifyTransaction(ctx *TransactionVerifyContext) VerifyResult
	BeforeTransactionsExecute(ctx *BeforeTransactionsExecuteContext) error
	AfterTransactionsExecute(ctx *AfterTransactionsExecuteContext) error
	BeforeCommandExecute(ctx *TransactionExecuteContext) error
	AfterCommandExecute(ctx *TransactionExecuteContext) error
	GetCommand(name string) (Command, bool)
}

type VerifyResult interface {
	Err() error
	OK() bool
	Code() int32
}

type verifyResult struct {
	err    error
	status VerifyStatus
}

func (v *verifyResult) OK() bool {
	return v.err == nil && v.status == verifyStatusOK
}

func (v *verifyResult) Err() error {
	return v.err
}

func (v *verifyResult) Code() int32 {
	return int32(v.status)
}

func NewVerifyResultOK() VerifyResult {
	return &verifyResult{status: verifyStatusOK}
}

func NewVerifyResultError(err error) VerifyResult {
	return &verifyResult{err: err, status: verifyStatusError}
}

func NewVerifyResultPending(err error) VerifyResult {
	return &verifyResult{err: err, status: verifyStatusPending}
}

type ExecResult interface {
	Err() error
	Code() int32
}

type execResult struct {
	err    error
	status ExecStatus
}

func (r *execResult) Code() int32 {
	return int32(r.status)
}

func (r *execResult) Err() error {
	return r.err
}

func NewExecResultOK() ExecResult {
	return &execResult{
		err:    nil,
		status: execStatusOK,
	}
}

func NewExecResultFail(err error) ExecResult {
	return &execResult{
		err:    err,
		status: execStatusFail,
	}
}
func NewExecResultInvalid(err error) ExecResult {
	return &execResult{
		err:    err,
		status: execStatusInvalid,
	}
}
