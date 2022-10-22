// Package validators implements validators module following [LIP-0044].
//
// [LIP-0044]: https://github.com/LiskHQ/lips/blob/main/proposals/lip-0044.md
package validators

import (
	"encoding/json"

	"github.com/LiskHQ/lisk-engine/pkg/framework/blueprint"
	"github.com/LiskHQ/lisk-engine/pkg/statemachine"
)

var (
	storePrefixValidatorsData uint16 = 0x0000
	storePrefixGeneratorList  uint16 = 0x4000
	storePrefixBLSKeys        uint16 = 0x8000
	storePrefixGenesisData    uint16 = 0xc000
	emptyKey                         = []byte{}
)

type Module struct {
	blueprint.Module
	api      *API
	endpoint *Endpoint
}

type ModuleConfig struct {
}

func NewModule() *Module {
	return &Module{
		endpoint: &Endpoint{},
		api:      &API{},
	}
}

func (m *Module) Init(cfg []byte) error {
	config := &ModuleConfig{}
	if err := json.Unmarshal(cfg, config); err != nil {
		return err
	}
	m.endpoint.init(m.ID())
	m.api.init(m.ID(), config)
	return nil
}

// Properties to satisfy state machine module.
func (m *Module) ID() uint32 {
	return 10
}
func (m *Module) Name() string {
	return "validators"
}

func (m *Module) API() *API {
	return m.api
}

func (m *Module) Endpoint() statemachine.Endpoint {
	return m.endpoint
}

// AfterGenesisBlockApply initialize the validator and finality.
func (m *Module) InitGenesisState(ctx *statemachine.GenesisBlockProcessingContext) error {
	genesisStore := ctx.GetStore(m.ID(), storePrefixGenesisData)
	if err := statemachine.SetEncodable(genesisStore, emptyKey, &GenesisState{
		GenesisTimestamp: ctx.BlockHeader().Timestamp(),
	}); err != nil {
		return err
	}
	return nil
}
