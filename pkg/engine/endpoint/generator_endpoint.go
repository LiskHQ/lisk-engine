package endpoint

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/ints"
	"github.com/LiskHQ/lisk-engine/pkg/consensus"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/db/diffdb"
	"github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/generator"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
	"github.com/LiskHQ/lisk-engine/pkg/router"
)

var (
	dayInMonth = 30
	monthInSec = uint32(60 * 60 * 24 * dayInMonth)
)

type generatorEndpoint struct {
	config        *config.Config
	chain         *blockchain.Chain
	consensusExec *consensus.Executer
	generator     *generator.Generator
	blockchainDB  *db.DB
	generatorDB   *db.DB
	abi           labi.ABI
}

func NewGeneratorEndpoint(
	config *config.Config,
	chain *blockchain.Chain,
	consensusExec *consensus.Executer,
	generator *generator.Generator,
	blockchainDB *db.DB,
	generatorDB *db.DB,
	abi labi.ABI,
) *generatorEndpoint {
	return &generatorEndpoint{
		config:        config,
		chain:         chain,
		consensusExec: consensusExec,
		generator:     generator,
		blockchainDB:  blockchainDB,
		generatorDB:   generatorDB,
		abi:           abi,
	}
}

func (a *generatorEndpoint) Endpoint() router.EndpointHandlers {
	return map[string]router.EndpointHandler{
		"updateStatus":       a.HandleUpdateStatus,
		"getStatus":          a.HandleGetStatus,
		"setStatus":          a.HandleSetStatus,
		"estimateSafeStatus": a.HandleEstimateSafeStatus,
		"getAllKeys":         a.HandleGetAllKeys,
		"setKeys":            a.HandleSetKeys,
		"hasKeys":            a.HandleHasKeys,
	}
}

type UpdateStatusRequest struct {
	GeneratorAddress   codec.Lisk32 `json:"generatorAddress" fieldNumber:"1"`
	Password           string       `json:"password" fieldNumber:"2"`
	Enable             bool         `json:"enable" fieldNumber:"3"`
	Height             uint32       `json:"height" fieldNumber:"4"`
	MaxHeightGenerated uint32       `json:"maxHeightGenerated" fieldNumber:"5"`
	MaxHeightPrevoted  uint32       `json:"maxHeightPrevoted" fieldNumber:"6"`
}
type UpdateStatusResponse struct {
	Address codec.Lisk32 `json:"generatorAddress" fieldNumber:"1"`
	Enabled bool         `json:"enabled" fieldNumber:"2"`
}

type GeneratorStatus struct {
	Address            codec.Lisk32 `json:"generatorAddress" fieldNumber:"1"`
	Enabled            bool         `json:"enabled" fieldNumber:"2"`
	Height             uint32       `json:"height" fieldNumber:"3"`
	MaxHeightGenerated uint32       `json:"maxHeightGenerated" fieldNumber:"4"`
	MaxHeightPrevoted  uint32       `json:"maxHeightPrevoted" fieldNumber:"5"`
}

type GetGeneratorsResponse struct {
	Status []*GeneratorStatus `json:"status" fieldNumber:"1"`
}

func (a *generatorEndpoint) HandleUpdateStatus(w router.EndpointResponseWriter, r *router.EndpointRequest) {
	req := &UpdateStatusRequest{}
	if err := json.Unmarshal(r.Params(), req); err != nil {
		w.Error(err)
		return
	}
	keysStore := diffdb.New(a.generatorDB, generator.GeneratorDBPrefixKeys)
	encodedKeys, exist := keysStore.Get(req.GeneratorAddress)
	if !exist {
		w.Error(fmt.Errorf("generator %s is not stored", req.GeneratorAddress))
		return
	}
	keys := &generator.Keys{}
	if err := keys.Decode(encodedKeys); err != nil {
		w.Error(err)
		return
	}
	plainKeys := &generator.PlainKeys{}
	if keys.IsPlain() {
		if err := plainKeys.Decode(keys.Data); err != nil {
			w.Error(err)
			return
		}
	} else {
		encryptedMsg := &crypto.EncryptedMessage{}
		if err := encryptedMsg.Decode(keys.Data); err != nil {
			w.Error(err)
			return
		}
		encodedPlainKeys, err := crypto.DecryptMessageWithPassword(encryptedMsg, req.Password)
		if err != nil {
			w.Error(err)
			return
		}
		if err := plainKeys.Decode(encodedPlainKeys); err != nil {
			w.Error(err)
			return
		}
	}

	if !req.Enable {
		a.generator.DisableGeneration(req.GeneratorAddress)
		w.Write(&UpdateStatusResponse{
			Address: req.GeneratorAddress,
			Enabled: false,
		})
		return
	}
	lastBlock := a.chain.LastBlock()
	synced, err := a.consensusExec.HeaderHasPriority(diffdb.New(a.blockchainDB, blockchain.DBPrefixToBytes(blockchain.DBPrefixState)), lastBlock.Header.Readonly(), req.Height, req.MaxHeightPrevoted, req.MaxHeightGenerated)
	if err != nil {
		w.Error(err)
		return
	}
	if !synced {
		w.Error(fmt.Errorf("failed to enable block generation as the node is not synced to the network"))
		return
	}
	geneInfoStore := diffdb.New(a.generatorDB, generator.GeneratorDBPrefixGeneratedInfo)
	if err := a.verifyAndUpdateGeneratorInfo(geneInfoStore, req.GeneratorAddress, req.Height, req.MaxHeightPrevoted, req.MaxHeightGenerated); err != nil {
		w.Error(fmt.Errorf("failed to enable block generation with %w", err))
		return
	}
	batch := a.generatorDB.NewBatch()
	geneInfoStore.Commit(batch)
	a.generatorDB.Write(batch)

	a.generator.EnableGeneration(req.GeneratorAddress, plainKeys)
	w.Write(&UpdateStatusResponse{
		Address: req.GeneratorAddress,
		Enabled: true,
	})
}

func (a *generatorEndpoint) HandleGetStatus(w router.EndpointResponseWriter, r *router.EndpointRequest) {
	generatorInfoStore := diffdb.New(a.generatorDB, generator.GeneratorDBPrefixGeneratedInfo)
	encodedGeneratorInfoList := generatorInfoStore.Iterate([]byte{}, -1, false)

	status := make([]*GeneratorStatus, len(encodedGeneratorInfoList))
	for i, encodedGeneratorInfo := range encodedGeneratorInfoList {
		generatorInfo := &generator.GeneratorInfo{}
		if err := generatorInfo.Decode(encodedGeneratorInfo.Value()); err != nil {
			w.Error(err)
			return
		}
		status[i] = &GeneratorStatus{
			Address:            codec.Lisk32(encodedGeneratorInfo.Key()),
			Enabled:            a.generator.IsGenerationEnabled(encodedGeneratorInfo.Key()),
			Height:             generatorInfo.Height,
			MaxHeightGenerated: generatorInfo.MaxHeightGenerated,
			MaxHeightPrevoted:  generatorInfo.MaxHeightPrevoted,
		}
	}
	w.Write(&GetGeneratorsResponse{
		Status: status,
	})
}

func (a *generatorEndpoint) verifyAndUpdateGeneratorInfo(generatorInfoStore *diffdb.Database, generatorAddress []byte, height, maxHeightPrevoted, maxHeightGenerated uint32) error {
	inputGeneratorInfo := &generator.GeneratorInfo{
		Height:             height,
		MaxHeightPrevoted:  maxHeightPrevoted,
		MaxHeightGenerated: maxHeightGenerated,
	}
	encodedInfo, _ := generatorInfoStore.Get(generatorAddress)

	if len(encodedInfo) != 0 {
		previousGeneratorInfo := &generator.GeneratorInfo{}
		if err := previousGeneratorInfo.Decode(encodedInfo); err != nil {
			return err
		}
		if !inputGeneratorInfo.Equal(previousGeneratorInfo) {
			return errors.New("failed to enable block generation due to contradicting block generation info")
		}
	} else if !inputGeneratorInfo.IsZero() {
		return errors.New("failed to enable block generation. there is no previous generator info")
	}
	encodedGeneratorInfo := inputGeneratorInfo.Encode()
	generatorInfoStore.Set(generatorAddress, encodedGeneratorInfo)
	return nil
}

type SetStatusRequest struct {
	Address            codec.Lisk32 `json:"address"`
	Height             uint32       `json:"height"`
	MaxHeightGenerated uint32       `json:"maxHeightPreviouslyForged"`
	MaxHeightPrevoted  uint32       `json:"maxHeightPrevoted"`
}
type SetStatusResponse struct {
}

func (a *generatorEndpoint) HandleSetStatus(w router.EndpointResponseWriter, r *router.EndpointRequest) {
	req := &SetStatusRequest{}
	if err := json.Unmarshal(r.Params(), req); err != nil {
		w.Error(err)
		return
	}
	inputGeneratorInfo := &generator.GeneratorInfo{
		Height:             req.Height,
		MaxHeightPrevoted:  req.MaxHeightPrevoted,
		MaxHeightGenerated: req.MaxHeightGenerated,
	}
	generatorInfoStore := diffdb.New(a.generatorDB, generator.GeneratorDBPrefixGeneratedInfo)
	encodedGeneratorInfo := inputGeneratorInfo.Encode()
	generatorInfoStore.Set(req.Address, encodedGeneratorInfo)
	batch := a.generatorDB.NewBatch()
	generatorInfoStore.Commit(batch)
	a.generatorDB.Write(batch)
	w.Write(&SetStatusResponse{})
}

type EstimateSafeStatusRequest struct {
	TimeShutdown uint32 `json:"timeShutdown"`
}

type EstimateSafeStatusResponse struct {
	Height             uint32 `json:"height"`
	MaxHeightPrevoted  uint32 `json:"maxHeightPrevoted"`
	MaxHeightGenerated uint32 `json:"maxHeightGenerated"`
}

func (a *generatorEndpoint) HandleEstimateSafeStatus(w router.EndpointResponseWriter, r *router.EndpointRequest) {
	req := &EstimateSafeStatusRequest{}
	if err := json.Unmarshal(r.Params(), req); err != nil {
		w.Error(err)
		return
	}
	finalizedHeight, err := a.chain.DataAccess().GetFinalizedHeight()
	if err != nil {
		w.Error(err)
		return
	}
	finalizedBlock, err := a.chain.DataAccess().GetBlockHeaderByHeight(finalizedHeight)
	if err != nil {
		w.Error(err)
		return
	}
	if finalizedBlock.Timestamp < req.TimeShutdown {
		w.Error(fmt.Errorf("a block at the time shutdown %d must be finalized", req.TimeShutdown))
		return
	}
	// 	// assume there is 30 days per month
	numberOfBlocksPerMonth := monthInSec / a.config.Genesis.BlockTime
	heightOneMonthAgo := ints.Max(numberOfBlocksPerMonth, 0)

	blockHeaderLastMonth, err := a.chain.DataAccess().GetBlockHeaderByHeight(heightOneMonthAgo)
	if err != nil {
		w.Error(err)
		return
	}
	missedBlocks := ((finalizedBlock.Timestamp - blockHeaderLastMonth.Timestamp) / a.config.Genesis.BlockTime) - (finalizedBlock.Height - blockHeaderLastMonth.Height)
	safeGeneratedHeight := finalizedHeight + missedBlocks
	w.Write(&EstimateSafeStatusResponse{
		Height:             safeGeneratedHeight,
		MaxHeightPrevoted:  safeGeneratedHeight,
		MaxHeightGenerated: safeGeneratedHeight,
	})
}

type GetAllKeysResponseKey struct {
	Address codec.Lisk32 `json:"address"`
	Type    string       `json:"type"`
	Data    interface{}  `json:"data"`
}

type GetAllKeysResponse struct {
	Keys []GetAllKeysResponseKey `json:"keys"`
}

func (a *generatorEndpoint) HandleGetAllKeys(w router.EndpointResponseWriter, r *router.EndpointRequest) {
	keysStore := diffdb.New(a.generatorDB, generator.GeneratorDBPrefixKeys)
	encodedKeysList := keysStore.Iterate([]byte{}, -1, false)

	resp := &GetAllKeysResponse{
		Keys: []GetAllKeysResponseKey{},
	}
	for _, encodedKeys := range encodedKeysList {
		keys := &generator.Keys{}
		if err := keys.Decode(encodedKeys.Value()); err != nil {
			w.Error(err)
			return
		}
		if keys.Type == generator.KeyTypePlain {
			plainKeys := &generator.PlainKeys{}
			if err := plainKeys.Decode(keys.Data); err != nil {
				w.Error(err)
				return
			}
			resp.Keys = append(resp.Keys, GetAllKeysResponseKey{
				Address: encodedKeys.Key(),
				Type:    keys.Type,
				Data:    plainKeys,
			})
			continue
		}
		encryptedMsg := &crypto.EncryptedMessage{}
		if err := encryptedMsg.Decode(keys.Data); err != nil {
			w.Error(err)
			return
		}
		resp.Keys = append(resp.Keys, GetAllKeysResponseKey{
			Address: encodedKeys.Key(),
			Type:    keys.Type,
			Data:    encryptedMsg,
		})
	}
	w.Write(resp)
}

type HasKeysRequest struct {
	Address codec.Lisk32 `json:"address"`
}
type HasKeysResponse struct {
	HasKeys bool `json:"hasKeys"`
}

func (a *generatorEndpoint) HandleHasKeys(w router.EndpointResponseWriter, r *router.EndpointRequest) {
	req := &HasKeysRequest{}
	if err := json.Unmarshal(r.Params(), req); err != nil {
		w.Error(err)
		return
	}
	keysStore := diffdb.New(a.generatorDB, generator.GeneratorDBPrefixKeys)
	_, exist := keysStore.Get(req.Address)
	if !exist {
		w.Write(&HasKeysResponse{HasKeys: false})
		return
	}

	w.Write(&HasKeysResponse{HasKeys: true})
}

type SetKeysRequest struct {
	Address codec.Lisk32    `json:"address"`
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
}
type SetKeysResponse struct {
}

func (r SetKeysRequest) Validate() error {
	if r.Type != generator.KeyTypeEncrypted && r.Type != generator.KeyTypePlain {
		return fmt.Errorf("invalid keys type %s", r.Type)
	}
	return nil
}

func (a *generatorEndpoint) HandleSetKeys(w router.EndpointResponseWriter, r *router.EndpointRequest) {
	req := &SetKeysRequest{}
	if err := json.Unmarshal(r.Params(), req); err != nil {
		w.Error(err)
		return
	}
	if err := req.Validate(); err != nil {
		w.Error(err)
		return
	}
	var data []byte
	if req.Type == generator.KeyTypePlain {
		keysData := &generator.PlainKeys{}
		if err := json.Unmarshal(req.Data, keysData); err != nil {
			w.Error(err)
			return
		}
		if err := keysData.Validate(); err != nil {
			w.Error(err)
			return
		}
		data = keysData.Encode()
	}
	keysStore := diffdb.New(a.generatorDB, generator.GeneratorDBPrefixKeys)
	keys := &generator.Keys{
		Address: req.Address,
		Type:    req.Type,
		Data:    data,
	}
	encodedKeys := keys.Encode()

	keysStore.Set(req.Address, encodedKeys)
	batch := a.generatorDB.NewBatch()
	keysStore.Commit(batch)
	a.generatorDB.Write(batch)
	w.Write(&SetKeysResponse{})
}
