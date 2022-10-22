package validators

import (
	"bytes"
	"errors"
	"fmt"
	"math"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

type API struct {
	moduleID uint32
}

func (a *API) init(moduleID uint32, cfg *ModuleConfig) {
	a.moduleID = moduleID
}

func (a *API) RegisterValidatorKeys(context statemachine.APIContext, address codec.Lisk32, blsKey, proofOfpossession, generationKey codec.Hex) error {
	validatorStore := context.GetStore(a.moduleID, storePrefixValidatorsData)
	_, validatorStoreErr := validatorStore.Get(address)
	if validatorStoreErr != nil && !errors.Is(validatorStoreErr, statemachine.ErrNotFound) {
		return validatorStoreErr
	}
	if validatorStoreErr == nil {
		return fmt.Errorf("address %s is already registered as validator", address.String())
	}

	blsStore := context.GetStore(a.moduleID, storePrefixBLSKeys)
	_, blsStoreErr := blsStore.Get(blsKey)
	if blsStoreErr != nil && !errors.Is(blsStoreErr, statemachine.ErrNotFound) {
		return blsStoreErr
	}
	if blsStoreErr == nil {
		return fmt.Errorf("address %s is already registered as validator", address.String())
	}

	if !crypto.BLSPopVerify(blsKey, proofOfpossession) {
		return fmt.Errorf("invalid proof of possession %s for %s", proofOfpossession, blsKey)
	}

	validator := &ValidatorAccount{
		GenerationKey: generationKey,
		BLSKey:        blsKey,
	}
	encodedValidator, err := validator.Encode()
	if err != nil {
		return err
	}
	if err := validatorStore.Set(address, encodedValidator); err != nil {
		return err
	}
	// Store BLS key index
	validatorAddress := &ValidatorAddress{
		Address: address,
	}
	encodedValidatorAddress, err := validatorAddress.Encode()
	if err != nil {
		return err
	}
	if err := blsStore.Set(blsKey, encodedValidatorAddress); err != nil {
		return err
	}
	return nil
}

func (a *API) GetValidatorAccount(context statemachine.ImmutableAPIContext, address codec.Lisk32) (codec.Hex, codec.Hex, error) {
	validatorStore := context.GetStore(a.moduleID, storePrefixValidatorsData)
	validator := &ValidatorAccount{}
	if err := statemachine.GetDecodable(validatorStore, address, validator); err != nil {
		return nil, nil, err
	}
	return validator.GenerationKey, validator.BLSKey, nil
}

func (a *API) GetGeneratorList(context statemachine.ImmutableAPIContext) ([]codec.Lisk32, error) {
	generatorStore := context.GetStore(a.moduleID, storePrefixGeneratorList)
	list := &GeneratorList{}
	if err := statemachine.GetDecodable(generatorStore, emptyKey, list); err != nil {
		return nil, err
	}
	return list.Addresses, nil
}

// SetGeneratorList updates validator set which will be used from next height.
func (a *API) SetGeneratorList(context statemachine.APIContext, validators []codec.Lisk32) error {
	validatorStore := context.GetStore(a.moduleID, storePrefixGeneratorList)
	list := &GeneratorList{
		Addresses: validators,
	}
	if err := statemachine.SetEncodable(validatorStore, emptyKey, list); err != nil {
		return err
	}
	return nil
}

func (a *API) GetValidatorGenerationKey(context statemachine.ImmutableAPIContext, address codec.Lisk32) (codec.Hex, error) {
	validatorStore := context.GetStore(a.moduleID, storePrefixValidatorsData)
	encodedValidator, err := validatorStore.Get(address)
	if err != nil {
		return nil, err
	}
	validator := &ValidatorAccount{}
	if err := validator.Decode(encodedValidator); err != nil {
		return nil, err
	}
	return validator.GenerationKey, nil
}

func (a *API) SetValidatorBLSKey(context statemachine.APIContext, address codec.Lisk32, blsKey, proofOfpossession codec.Hex) error {
	blsStore := context.GetStore(a.moduleID, storePrefixBLSKeys)
	_, blsStoreErr := blsStore.Get(blsKey)
	if blsStoreErr != nil && !errors.Is(blsStoreErr, statemachine.ErrNotFound) {
		return blsStoreErr
	}
	if blsStoreErr == nil {
		return fmt.Errorf("address %s is already registered as validator", address)
	}

	validatorStore := context.GetStore(a.moduleID, storePrefixValidatorsData)
	validatorAccount := &ValidatorAccount{}
	if err := statemachine.GetDecodable(validatorStore, address, validatorAccount); err != nil {
		return err
	}

	invalidBLSKey := make([]byte, 48)
	if bytes.Equal(validatorAccount.BLSKey, invalidBLSKey) {
		return fmt.Errorf("address %s is has already valid BLS key", address)
	}

	if !crypto.BLSPopVerify(blsKey, proofOfpossession) {
		return fmt.Errorf("invalid proof of possession %s for %s", proofOfpossession, blsKey)
	}

	validatorAccount.BLSKey = blsKey
	if err := statemachine.SetEncodable(validatorStore, address, validatorAccount); err != nil {
		return err
	}

	validatorAddress := &ValidatorAddress{
		Address: address,
	}
	if err := statemachine.SetEncodable(blsStore, blsKey, validatorAddress); err != nil {
		return err
	}
	return nil
}

func (a *API) SetValidatorGeneratorKey(context statemachine.APIContext, address codec.Lisk32, generatorKey codec.Hex) error {
	validatorStore := context.GetStore(a.moduleID, storePrefixValidatorsData)
	validatorAccount := &ValidatorAccount{}
	err := statemachine.GetDecodable(validatorStore, address, validatorAccount)
	if err != nil {
		if errors.Is(err, statemachine.ErrNotFound) {
			return fmt.Errorf("address %s is not registered as a validator", address)
		}
		return err
	}
	validatorAccount.GenerationKey = generatorKey
	if err := statemachine.SetEncodable(validatorStore, address, validatorAccount); err != nil {
		return err
	}
	return nil
}

func (a *API) IsKeyRegistered(context statemachine.ImmutableAPIContext, blsKey codec.Hex) (bool, error) {
	blsStore := context.GetStore(a.moduleID, storePrefixBLSKeys)
	_, err := blsStore.Get(blsKey)
	if err != nil {
		if errors.Is(err, statemachine.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// FIXME: remove this usage
// GetGeneratorsBetweenTimestamps returns a map [address string]missedBlockCount between previous and current timestamp.
func (a *API) GetGeneratorsBetweenTimestamps(context statemachine.ImmutableAPIContext, startTimestamp, endTimestamp uint32, validators labi.Validators) (map[string]uint32, error) {
	result := map[string]uint32{}
	if startTimestamp > endTimestamp {
		return nil, fmt.Errorf("received invalid previous timestamp %d greater than current timestamp %d", startTimestamp, endTimestamp)
	}
	genesisStore := context.GetStore(a.moduleID, storePrefixGenesisData)
	genesisState := &GenesisState{}
	if err := statemachine.GetDecodable(genesisStore, emptyKey, genesisState); err != nil {
		return nil, err
	}
	startSlotNumber := SlotNumber(genesisState.GenesisTimestamp, 10, int64(startTimestamp))
	endSlotNumber := SlotNumber(genesisState.GenesisTimestamp, 10, int64(endTimestamp))
	totalSlots := endSlotNumber - startSlotNumber + 1
	baseSlots := int(math.Floor(float64(totalSlots) / float64(len(validators))))
	if baseSlots > 0 {
		totalSlots -= baseSlots * len(validators)
		for _, validator := range validators {
			result[string(validator.Address)] = uint32(baseSlots)
		}
	}
	for slotNumber := startSlotNumber; slotNumber < startSlotNumber+totalSlots; slotNumber++ {
		index := slotNumber % len(validators)
		validator := validators[index]
		if _, exist := result[string(validator.Address)]; exist {
			result[string(validator.Address)] += 1
		} else {
			result[string(validator.Address)] = 1
		}
	}

	return result, nil
}
