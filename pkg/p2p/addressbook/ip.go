package addressbook

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Network identifies type.
type Network uint8

const (
	networkIPV4 Network = iota
	networkPrivate
	networkLocal
	networkOthers
)

func ipToBytes(ip string) ([]byte, error) {
	splitIP := strings.Split(ip, ".")
	if len(splitIP) != 4 {
		return nil, fmt.Errorf("invalid IP format: %s", ip)
	}
	result := make([]byte, len(splitIP))
	for i, numStr := range splitIP {
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return nil, err
		}
		if num > 255 {
			return nil, fmt.Errorf("invalid IP format: %s", ip)
		}
		result[i] = uint8(num)
	}
	return result, nil
}

func IsIPV4(ip string) bool {
	if net.ParseIP(ip) == nil {
		return false
	}
	return strings.Contains(ip, ".")
}

func isPrivateNetwork(ipBytes []byte) bool {
	if ipBytes[0] == 10 {
		return true
	}
	if ipBytes[0] == 172 && (ipBytes[1] >= 16 && ipBytes[1] <= 31) {
		return true
	}
	return false
}

// getNetwork returns network category.
func GetNetwork(ip string) (Network, error) {
	ipBytes, err := ipToBytes(ip)
	if err != nil {
		return 0, err
	}
	if ipBytes[0] == 127 || ip == "0.0.0.0" {
		return networkLocal, nil
	}
	if isPrivateNetwork(ipBytes) {
		return networkPrivate, nil
	}
	if IsIPV4(ip) {
		return networkIPV4, nil
	}
	return networkOthers, fmt.Errorf("unsupported network with IP %s", ip)
}
