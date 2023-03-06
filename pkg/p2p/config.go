package p2p

import "github.com/LiskHQ/lisk-engine/pkg/codec"

type Config struct {
	Version                  string
	Addresses                []string
	AdvertiseAddresses       bool
	ConnectionSecurity       string
	AllowIncomingConnections bool
	EnableNATService         bool
	EnableUsingRelayService  bool
	EnableRelayService       bool
	EnableHolePunching       bool
	SeedPeers                []string
	FixedPeers               []string
	BlacklistedIPs           []string
	MinNumOfConnections      int
	MaxNumOfConnections      int
	// GossipSub configuration
	IsSeedPeer bool
	ChainID    codec.Hex
}

func (c *Config) insertDefault() error {
	if c.Version == "" {
		c.Version = "1.0"
	}
	if c.Addresses == nil {
		c.Addresses = []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}
	}
	if !c.AdvertiseAddresses {
		c.AdvertiseAddresses = true
	}
	if c.ConnectionSecurity == "" {
		c.ConnectionSecurity = "tls"
	}
	if c.SeedPeers == nil {
		c.SeedPeers = []string{}
	}
	if c.FixedPeers == nil {
		c.FixedPeers = []string{}
	}
	if c.BlacklistedIPs == nil {
		c.BlacklistedIPs = []string{}
	}
	if c.MinNumOfConnections == 0 {
		c.MinNumOfConnections = 20
	}
	if c.MaxNumOfConnections == 0 {
		c.MaxNumOfConnections = 100
	}
	return nil
}
