package auth

import (
	"errors"
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/framework/blueprint"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

type RegisterMultisignatureGroup struct {
	blueprint.Command
}

func (a *RegisterMultisignatureGroup) ID() uint32 {
	return 0
}

func (a *RegisterMultisignatureGroup) Name() string {
	return "registerMultisignatureGroup"
}

func (a *RegisterMultisignatureGroup) Verify(ctx *statemachine.TransactionVerifyContext) error {
	params := &RegisterMultisignatureParams{}
	if err := params.Decode(ctx.Transaction().Params()); err != nil {
		return err
	}
	if err := params.Validate(); err != nil {
		return err
	}
	if len(ctx.Transaction().Signatures()) != len(params.MandatoryKeys)+len(params.OptionalKeys)+1 {
		return fmt.Errorf("number of keys do not match with required number of signatures %d", len(params.MandatoryKeys)+len(params.OptionalKeys)+1)
	}
	// Validate all signatures
	for _, signature := range ctx.Transaction().Signatures() {
		if len(signature) != 64 {
			return fmt.Errorf("invalid signature %v is included", signature)
		}
	}
	signingBytes, err := ctx.Transaction().SigningBytes()
	if err != nil {
		return err
	}
	networkSigningBytes := crypto.Hash(bytes.Join(ctx.ChainID(), signingBytes))
	if err := crypto.VerifySignature(ctx.Transaction().SenderPublicKey(), ctx.Transaction().Signatures()[0], networkSigningBytes); err != nil {
		return errors.New("invalid sender signature")
	}
	for i, key := range params.MandatoryKeys {
		if err := crypto.VerifySignature(key, ctx.Transaction().Signatures()[i+1], networkSigningBytes); err != nil {
			return fmt.Errorf("invalid signature from mandatory key is included at %d", i)
		}
	}
	for i, key := range params.OptionalKeys {
		if err := crypto.VerifySignature(key, ctx.Transaction().Signatures()[i+1+len(params.MandatoryKeys)], networkSigningBytes); err != nil {
			return fmt.Errorf("invalid signature from mandatory key is included at %d", i)
		}
	}
	keysAccount := &UserAccount{}
	keyStore := ctx.GetStore(a.ID(), storePrefixUserAccount)
	if err := statemachine.GetDecodableOrDefault(keyStore, ctx.Transaction().SenderAddress(), keysAccount); err != nil {
		return err
	}
	if keysAccount.IsMultisignatureAccount() {
		return fmt.Errorf("address %s is already a multi-signature account", ctx.Transaction().SenderAddress())
	}
	return nil
}

func (a *RegisterMultisignatureGroup) Execute(ctx *statemachine.TransactionExecuteContext) error {
	keysAccount := &UserAccount{}
	keyStore := ctx.GetStore(a.ID(), storePrefixUserAccount)
	if err := statemachine.GetDecodableOrDefault(keyStore, ctx.Transaction().SenderAddress(), keysAccount); err != nil {
		return err
	}
	if keysAccount.NumberOfSignatures != 0 {
		return errors.New("register multisignature only allowed once per account")
	}
	asset := &RegisterMultisignatureParams{}
	if err := asset.Decode(ctx.Transaction().Params()); err != nil {
		return err
	}
	keysAccount.MandatoryKeys = asset.MandatoryKeys
	keysAccount.OptionalKeys = asset.OptionalKeys
	keysAccount.NumberOfSignatures = asset.NumberOfSignatures
	if err := statemachine.SetEncodable(keyStore, ctx.Transaction().SenderAddress(), keysAccount); err != nil {
		return err
	}
	return nil
}
