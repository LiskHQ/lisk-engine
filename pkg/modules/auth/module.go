// Package auth implements auth module following [LIP-0041].
//
// [LIP-0041]: https://github.com/LiskHQ/lips/blob/main/proposals/lip-0041.md
package auth

import (
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/framework/blueprint"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

const (
	MaxKeyCount = 64
)

type AuthModuleConfig struct {
}

type Module struct {
	blueprint.Module
	api      *API
	endpoint *Endpoint
}

func NewModule() *Module {
	return &Module{
		api:      &API{},
		endpoint: &Endpoint{},
	}
}

func (m *Module) ID() uint32 {
	return 3
}

func (m *Module) Name() string {
	return "auth"
}

func (m *Module) GetCommand(name string) (statemachine.Command, bool) {
	return nil, false
}

func (m *Module) API() *API {
	return m.api
}

func (m *Module) Endpoint() statemachine.Endpoint {
	return m.endpoint
}
func (m *Module) Init(cfg []byte) error {
	m.api.init(m.ID())
	m.endpoint.init(m.ID())
	return nil
}

func (m *Module) InitGenesisState(ctx *statemachine.GenesisBlockProcessingContext) error {
	assetBytes, exist := ctx.BlockAssets().GetAsset(m.Name())
	if !exist {
		return nil
	}
	asset := &GenesisAsset{}
	if err := asset.Decode(assetBytes); err != nil {
		return err
	}
	userStore := ctx.GetStore(m.ID(), storePrefixUserAccount)
	addresses := make([][]byte, len(asset.AuthDataSubstore))
	for i, acct := range asset.AuthDataSubstore {
		if len(acct.Address) != 20 {
			return fmt.Errorf("invalid address length for auth module")
		}
		addresses[i] = acct.Address
		mandatoryKeys := make([][]byte, len(acct.AuthAccount.MandatoryKeys))
		for i, key := range acct.AuthAccount.MandatoryKeys {
			mandatoryKeys[i] = key
		}
		if !bytes.IsSorted(mandatoryKeys) {
			return fmt.Errorf("mandatory keys for %s is not sorted", acct.Address)
		}
		if !bytes.IsUnique(mandatoryKeys) {
			return fmt.Errorf("mandatory keys for %s is not unique", acct.Address)
		}

		optionalKeys := make([][]byte, len(acct.AuthAccount.OptionalKeys))
		for i, key := range acct.AuthAccount.OptionalKeys {
			optionalKeys[i] = key
		}
		if !bytes.IsSorted(optionalKeys) {
			return fmt.Errorf("optional keys for %s is not sorted", acct.Address)
		}
		if !bytes.IsUnique(optionalKeys) {
			return fmt.Errorf("optional keys for %s is not unique", acct.Address)
		}

		if len(mandatoryKeys)+len(optionalKeys) > MaxKeyCount {
			return fmt.Errorf("number of mandatory and optional keys cannot exceed %d", MaxKeyCount)
		}
		totalKeys := bytes.JoinSlice(mandatoryKeys, optionalKeys)
		if !bytes.IsUnique(totalKeys) {
			return fmt.Errorf("mandatory and optional keys for %s is not unique", acct.Address)
		}
		if len(mandatoryKeys)+len(optionalKeys) < int(acct.AuthAccount.NumberOfSignatures) {
			return fmt.Errorf("numberOfSignatures %d is bigger than total of mandatory and optional keys", acct.AuthAccount.NumberOfSignatures)
		}
		if len(mandatoryKeys) > int(acct.AuthAccount.NumberOfSignatures) {
			return fmt.Errorf("numberOfSignatures %d is smaller than number of mandatory keys", acct.AuthAccount.NumberOfSignatures)
		}

		initAccount := &UserAccount{
			Nonce:              acct.AuthAccount.Nonce,
			NumberOfSignatures: acct.AuthAccount.NumberOfSignatures,
			MandatoryKeys:      mandatoryKeys,
			OptionalKeys:       optionalKeys,
		}
		if err := statemachine.SetEncodable(userStore, acct.Address, initAccount); err != nil {
			return err
		}
	}
	if !bytes.IsUnique(addresses) {
		return fmt.Errorf("duplicate address in genesis assets")
	}
	return nil
}

func (m *Module) VerifyTransaction(ctx *statemachine.TransactionVerifyContext) statemachine.VerifyResult {
	userStore := ctx.GetStore(m.ID(), storePrefixUserAccount)
	userAccount := &UserAccount{}
	if err := statemachine.GetDecodableOrDefault(userStore, ctx.Transaction().SenderAddress(), userAccount); err != nil {
		return statemachine.NewVerifyResultError(err)
	}
	if err := verifySignatures(userAccount, ctx.ChainID(), ctx.Transaction()); err != nil {
		return statemachine.NewVerifyResultError(err)
	}
	if ctx.Transaction().Nonce() < userAccount.Nonce {
		return statemachine.NewVerifyResultError(fmt.Errorf(
			"address %s has higher nonce %d than transaction nonce %d",
			ctx.Transaction().SenderAddress(),
			userAccount.Nonce, ctx.Transaction().Nonce(),
		))
	}
	if ctx.Transaction().Nonce() > userAccount.Nonce {
		return statemachine.NewVerifyResultPending(fmt.Errorf(
			"address %s has lower nonce %d than transaction nonce %d",
			ctx.Transaction().SenderAddress(),
			userAccount.Nonce, ctx.Transaction().Nonce(),
		))
	}
	return statemachine.NewVerifyResultOK()
}

func (m *Module) BeforeCommandExecute(ctx *statemachine.TransactionExecuteContext) error {
	userAccount := &UserAccount{}
	userStore := ctx.GetStore(m.ID(), storePrefixUserAccount)
	if err := statemachine.GetDecodableOrDefault(userStore, ctx.Transaction().SenderAddress(), userAccount); err != nil {
		return err
	}
	if ctx.Transaction().Nonce() != userAccount.Nonce {
		return fmt.Errorf(
			"address %s has nonce %d but transaction nonce is %d",
			ctx.Transaction().SenderAddress(),
			userAccount.Nonce, ctx.Transaction().Nonce(),
		)
	}
	userAccount.Nonce += 1
	if err := statemachine.SetEncodable(userStore, ctx.Transaction().SenderAddress(), userAccount); err != nil {
		return err
	}
	return nil
}
