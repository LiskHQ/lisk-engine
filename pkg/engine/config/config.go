// Package config provides config structure for engine.
package config

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/strings"
	"github.com/LiskHQ/lisk-engine/pkg/p2p/v2"
)

var (
	logLevels                 = []string{"trace", "debug", "info", "warn", "error", "fatal"}
	defaultTimeoutHour        = 6
	minInHour                 = 60
	secInMin                  = 60
	defaultBlockCache         = 515
	defaultKeepEventHeight    = 309
	defaultMaxTransactionSize = 15 * 1024
)

type Config struct {
	System          *SystemConfig          `json:"system"`
	RPC             *RPCConfig             `json:"rpc"`
	TransactionPool *TransactionPoolConfig `json:"transactionPool"`
	Network         *NetworkConfig         `json:"network"`
	Genesis         *GenesisConfig         `json:"genesis"`
	Generator       *GeneratorConfig       `json:"generator"`
}

func intPtr(v int) *int {
	return &v
}

func (c *Config) InsertDefault() error {
	if c.System == nil {
		c.System = &SystemConfig{}
	}
	if err := c.System.InsertDefault(); err != nil {
		return err
	}
	if c.TransactionPool == nil {
		c.TransactionPool = &TransactionPoolConfig{}
	}
	if err := c.TransactionPool.InsertDefault(); err != nil {
		return err
	}
	if c.RPC == nil {
		c.RPC = &RPCConfig{}
	}
	if err := c.RPC.InsertDefault(); err != nil {
		return err
	}
	if c.Network == nil {
		c.Network = &NetworkConfig{}
	}
	if err := c.Network.InsertDefault(); err != nil {
		return err
	}
	if c.Genesis == nil {
		c.Genesis = &GenesisConfig{}
	}
	if err := c.Genesis.InsertDefault(); err != nil {
		return err
	}
	if c.Generator == nil {
		c.Generator = &GeneratorConfig{
			Keys: &KeysConfig{},
		}
	}
	return nil
}

func (c *Config) Merge(config *Config) {
	c.System.Merge(config.System)
	if config.RPC != nil {
		if c.RPC == nil {
			c.RPC = config.RPC
		} else {
			c.RPC.Merge(config.RPC)
		}
	}
}

func (c *Config) Validate() error {
	if err := c.RPC.Validate(); err != nil {
		return err
	}
	return nil
}

type GeneratorConfig struct {
	Keys *KeysConfig `json:"keys"`
}

type KeysConfig struct {
	FromFile string `json:"fromFile"`
}

type TransactionPoolConfig struct {
	MaxTransactions             int    `json:"maxTransactions"`
	MaxTransactionsPerAccount   int    `json:"maxTransactionsPerAccount"`
	TransactionExpiryTime       int    `json:"transactionExpiryTime"`
	MinEntranceFeePriority      uint64 `json:"minEntranceFeePriority,string"`
	MinReplacementFeeDifference uint64 `json:"minReplacementFeeDifference,string"`
}

func (c *TransactionPoolConfig) InsertDefault() error {
	if c.MaxTransactions == 0 {
		c.MaxTransactions = 4096
	}
	if c.MaxTransactionsPerAccount == 0 {
		c.MaxTransactionsPerAccount = 64
	}
	if c.TransactionExpiryTime == 0 {
		c.TransactionExpiryTime = defaultTimeoutHour * minInHour * secInMin
	}
	return nil
}

type RPCConfig struct {
	Modes []string `json:"modes"`
	Port  int      `json:"port"`
	Host  string   `json:"host"`
}

func (c *RPCConfig) InsertDefault() error {
	if c.Port == 0 {
		c.Port = 7887
	}
	if c.Host == "" {
		c.Host = "0.0.0.0"
	}
	return nil
}
func (c *RPCConfig) Merge(config *RPCConfig) {
	if config.Host != "" {
		c.Host = config.Host
	}
	if config.Port != 0 {
		c.Port = config.Port
	}
}

func (c *RPCConfig) Validate() error {
	if c.Port < 1024 || c.Port > 65535 {
		return fmt.Errorf("invalid port %d for RPC is specified", c.Port)
	}
	return nil
}

type SystemConfig struct {
	Version              string `json:"version"`
	DataPath             string `json:"dataPath"`
	KeepEventsForHeights *int   `json:"keepEventsForHeights"`
	LogLevel             string `json:"logLevel"`
	MaxBlockCache        *int   `json:"maxBlockCache"`
}

func (c SystemConfig) GetKeepEventsForHeights() int {
	if c.KeepEventsForHeights != nil {
		return *c.KeepEventsForHeights
	}
	return -1
}

func (c SystemConfig) GetMaxBlokckCache() int {
	if c.MaxBlockCache != nil {
		return *c.MaxBlockCache
	}
	return defaultBlockCache
}

func (c *SystemConfig) InsertDefault() error {
	if c.Version == "" {
		c.Version = "0.1.0"
	}
	if c.MaxBlockCache == nil {
		c.MaxBlockCache = intPtr(defaultBlockCache)
	}
	if c.KeepEventsForHeights == nil {
		c.KeepEventsForHeights = intPtr(defaultKeepEventHeight)
	}
	if c.DataPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		c.DataPath = path.Join(home, ".lisk", "engine")
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	return nil
}

func (c *SystemConfig) Merge(config *SystemConfig) {
	if config.Version != "" {
		c.Version = config.Version
	}
	if config.DataPath != "" {
		c.DataPath = config.DataPath
	}
	if config.MaxBlockCache != nil {
		c.MaxBlockCache = config.MaxBlockCache
	}
	if config.LogLevel != "" {
		c.LogLevel = config.LogLevel
	}
}

func (c SystemConfig) Validate() error {
	if !strings.Contain(logLevels, c.LogLevel) {
		return fmt.Errorf("log level %s is not allowed", c.LogLevel)
	}
	if c.DataPath == "" {
		return errors.New("dataPath cannot be empty")
	}
	return nil
}

type GenesisBlockConfig struct {
	FromFile string    `json:"fromFile"`
	Blob     codec.Hex `json:"blob"`
}

type GenesisConfig struct {
	Block               *GenesisBlockConfig `json:"block"`
	ChainID             codec.Hex           `json:"chainID"`
	BlockTime           uint32              `json:"blockTime"`
	MaxTransactionsSize uint32              `json:"maxTransactionsSize"`
	BFTBatchSize        uint32              `json:"bftBatchSize"`
}

func (c *GenesisConfig) InsertDefault() error {
	if c.BlockTime == 0 {
		c.BlockTime = 10
	}
	if c.MaxTransactionsSize == 0 {
		c.MaxTransactionsSize = uint32(defaultMaxTransactionSize)
	}
	if c.BFTBatchSize == 0 {
		c.BFTBatchSize = 101
	}

	return nil
}

type NetworkConfig struct {
	Version                string     `json:"version"`
	Port                   int        `json:"port"`
	AdvertiseAddress       bool       `json:"advertiseAddress"`
	SeedPeers              NetworkIPs `json:"seedPeers"`
	WhiteListedPeers       NetworkIPs `json:"whitelistedPeers"`
	FixedPeers             NetworkIPs `json:"fixedPeers"`
	BlackListedIPs         []string   `json:"blackListedIPs"`
	MaxInboundConnections  int        `json:"maxInboundConnections"`
	MaxOutboundConnections int        `json:"maxOutboundConnections"`
}

func (c *NetworkConfig) InsertDefault() error {
	if c.Version == "" {
		c.Version = "1.0"
	}
	if c.Port == 0 {
		c.Port = 4000
	}
	if c.MaxInboundConnections == 0 {
		c.MaxInboundConnections = 100
	}
	if c.MaxOutboundConnections == 0 {
		c.MaxOutboundConnections = 20
	}
	if c.SeedPeers == nil {
		c.SeedPeers = NetworkIPs{}
	}
	if c.WhiteListedPeers == nil {
		c.WhiteListedPeers = NetworkIPs{}
	}
	if c.FixedPeers == nil {
		c.FixedPeers = NetworkIPs{}
	}
	if c.BlackListedIPs == nil {
		c.BlackListedIPs = []string{}
	}
	return nil
}

type NetworkIPs []NetworkIP

func (n NetworkIPs) GetP2PAddress() []*p2p.Address {
	addresses := make([]*p2p.Address, len(n))
	for i, addr := range n {
		addresses[i] = p2p.NewAddress(addr.IP, addr.Port)
	}
	return addresses
}

type NetworkIP struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}
