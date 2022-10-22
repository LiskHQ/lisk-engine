// Package addressbook implements bucket list following [LIP-0004].
//
// [LIP-0004]: https://github.com/LiskHQ/lips/blob/main/proposals/lip-0094.md
package addressbook

import (
	"fmt"
	"net"
	"strings"
	"time"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

// Addresses holds multiple address.
type Addresses struct {
	Addresses []*Address `fieldNumber:"1"`
}

// AddressList holds multiple addresses.
type AddressList []*Address

// Include checks if address list includes the ip and port.
func (al AddressList) Include(ip string, port int) bool {
	for _, addr := range al {
		if addr.ip == ip && addr.port == uint32(port) {
			return true
		}
	}
	return false
}

// Address defines a content of the book.
type Address struct {
	ip        string `fieldNumber:"1"`
	port      uint32 `fieldNumber:"2"`
	advertise bool
	createdAt time.Time
	bannedAt  *time.Time
	source    *string
}

func (a Address) String() string {
	return fmt.Sprintf("%s:%d", a.ip, a.port)
}

// IP is getter for ip.
func (a *Address) IP() string {
	return a.ip
}

// Port is getter for port.
func (a *Address) Port() int {
	return int(a.port)
}

// SetAdvertise is setter for advertise.
func (a *Address) SetAdvertise(advertise bool) {
	a.advertise = advertise
}

// SetSource is setter for advertise.
func (a *Address) SetSource(source string) {
	a.source = &source
}

// NewAddressFromBytes creates new address set.
func NewAddressFromBytes(encoded []byte) (*Address, error) {
	address := &Address{
		advertise: true,
		createdAt: time.Now(),
	}
	if err := address.Decode(encoded); err != nil {
		return address, err
	}
	return address, nil
}

// NewAddress creates new address set.
func NewAddress(ip string, port int) *Address {
	address := &Address{
		ip:        ip,
		port:      uint32(port),
		advertise: true,
		createdAt: time.Now(),
	}
	return address
}

// NewAddressWithoutIP creates new address set.
func NewAddressWithoutIP(advertise bool, source *string) *Address {
	address := &Address{
		advertise: advertise,
		createdAt: time.Now(),
		source:    source,
	}
	return address
}

// NewAddressFromNetAddr creates new addr from net.
func NewAddressFromNetAddr(addr net.Addr, port int) (*Address, error) {
	if !IsIPV4(addr.String()) {
		return nil, fmt.Errorf("address %s is not in IPV4 format", addr.String())
	}
	addrStr := strings.Split(addr.String(), ":")
	return NewAddress(addrStr[0], port), nil
}
