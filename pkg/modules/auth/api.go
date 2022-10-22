package auth

import (
	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/framework/blueprint"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

type API struct {
	blueprint.API
	moduleID uint32
}

func (a *API) init(moduleID uint32) {
	a.moduleID = moduleID
}

func (a *API) GetNonce(context statemachine.APIContext, address codec.Lisk32) (uint64, error) {
	userStore := context.GetStore(a.moduleID, storePrefixUserAccount)
	data := &UserAccount{}
	if err := statemachine.GetDecodableOrDefault(userStore, address, data); err != nil {
		return 0, err
	}
	return data.Nonce, nil
}

func (a *API) VerifySignatures(context statemachine.ImmutableAPIContext, chainID []byte, transaction blockchain.FrozenTransaction) error {
	userStore := context.GetStore(a.moduleID, storePrefixUserAccount)
	userAccount := &UserAccount{}
	if err := statemachine.GetDecodableOrDefault(userStore, transaction.SenderAddress(), userAccount); err != nil {
		return err
	}
	return verifySignatures(userAccount, chainID, transaction)
}
