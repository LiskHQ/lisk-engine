package blueprint

import "github.com/LiskHQ/lisk-engine/pkg/statemachine"

type Command struct{}

func (b *Command) Verify(ctx *statemachine.TransactionVerifyContext) statemachine.VerifyResult {
	return statemachine.NewVerifyResultOK()
}
