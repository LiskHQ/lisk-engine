package liskbft

import (
	"github.com/LiskHQ/lisk-engine/pkg/labi"
)

func GetBFTValidatorAndGenerators(validators labi.Validators) (BFTValidators, Generators) {
	bftValidators := BFTValidators{}
	generators := Generators{}
	for _, validator := range validators {
		if validator.BFTWeight > 0 {
			bftValidators = append(bftValidators, NewValidator(
				validator.Address,
				validator.BFTWeight,
				validator.BLSKey,
			))
		}
		generators = append(generators, NewGenerator(validator.Address, validator.GeneratorKey))
	}
	return bftValidators, generators
}

func GetLabiValidators(validators BFTValidators, generators Generators) labi.Validators {
	labiValidators := make(labi.Validators, len(generators))
	for i, generator := range generators {
		labiValidators[i] = &labi.Validator{
			Address:      generator.Address(),
			GeneratorKey: generator.GeneratorKey(),
			BFTWeight:    0,
			BLSKey:       []byte{},
		}
		if validator, ok := validators.Find(generator.Address()); ok {
			labiValidators[i].BFTWeight = validator.BFTWeight()
			labiValidators[i].BLSKey = validator.BLSKey()
		}
	}
	return labiValidators
}
