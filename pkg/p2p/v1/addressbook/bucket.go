package addressbook

import (
	"encoding/binary"
	"math/rand"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/collection/strings"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

type bucketMod int

const (
	bucketModNewPeer   bucketMod = 16
	bucketModTriedPeer bucketMod = 4
	maxCellSize        int       = 32
)

type bucket struct {
	secret        []byte
	cells         map[int]bucketCell
	addresses     map[string]*Address
	numberOfCells int
}

func newBucket(numberOfCells int, secret []byte) *bucket {
	data := map[int]bucketCell{}
	for i := 0; i < numberOfCells; i++ {
		data[i] = bucketCell{}
	}
	return &bucket{
		secret:        secret,
		cells:         data,
		numberOfCells: numberOfCells,
		addresses:     make(map[string]*Address),
	}
}
func (b *bucket) getAll(excludeIPs []string) []*Address {
	res := []*Address{}
	for _, val := range b.addresses {
		if val.advertise && !strings.Contain(excludeIPs, val.IP()) {
			res = append(res, val)
		}
	}
	return res
}

func (b *bucket) size() int {
	size := 0
	for _, cell := range b.cells {
		size += len(cell)
	}
	return size
}

func (b *bucket) get(ip string) (*Address, bool) {
	return nil, false
}

func (b *bucket) insert(addr *Address) (bool, error) {
	_, exist := b.addresses[addr.IP()]
	if exist {
		return false, nil
	}
	sourceBytes, err := getSourceBytes(addr)
	if err != nil {
		return false, err
	}
	id, err := GetSubStoreID(addr.IP(), b.secret, b.numberOfCells, sourceBytes)
	if err != nil {
		return false, err
	}
	cell := b.cells[id]
	evicted := cell.insert(addr)
	if evicted != nil {
		delete(b.addresses, evicted.IP())
	}
	b.cells[id] = cell
	b.addresses[addr.IP()] = addr
	return evicted != nil, nil
}

func (b *bucket) exist(addr *Address) (bool, error) {
	sourceBytes, err := getSourceBytes(addr)
	if err != nil {
		return false, err
	}
	id, err := GetSubStoreID(addr.IP(), b.secret, b.numberOfCells, sourceBytes)
	if err != nil {
		return false, err
	}
	cell := b.cells[id]
	return cell.exist(addr), nil
}

func (b *bucket) remove(addr *Address) error {
	sourceBytes, err := getSourceBytes(addr)
	if err != nil {
		return err
	}
	id, err := GetSubStoreID(addr.IP(), b.secret, b.numberOfCells, sourceBytes)
	if err != nil {
		return err
	}
	cell := b.cells[id]
	cell.remove(addr)
	delete(b.addresses, addr.IP())
	return nil
}

type bucketCell []*Address

func (c *bucketCell) insert(addr *Address) *Address {
	original := *c
	var evicted *Address
	if !original.hasSpace() {
		evicted = original.evict()
	}
	original = append(original, addr)
	*c = original
	return evicted
}

func (c bucketCell) exist(addr *Address) bool {
	for _, containedAddr := range c {
		if containedAddr.IP() == addr.IP() {
			return true
		}
	}
	return false
}

func (c bucketCell) hasSpace() bool {
	return len(c) < maxCellSize
}

func (c *bucketCell) remove(addr *Address) {
	newCell := []*Address{}
	for _, existing := range *c {
		if existing.IP() != addr.IP() {
			newCell = append(newCell, existing)
		}
	}
	*c = newCell
}

func (c *bucketCell) evict() *Address {
	now := time.Now()
	foundIndex := -1
	// remove peer older than 30 days
	for i, addr := range *c {
		if addr.createdAt.Before(now.AddDate(0, -1, 0)) {
			foundIndex = i
			break
		}
	}
	if foundIndex < 0 {
		// if not raondomly choose one
		foundIndex = rand.Intn(len(*c))
	}
	evicted := (*c)[foundIndex]
	newCell := []*Address{}
	newCell = append(newCell, (*c)[:foundIndex]...)
	newCell = append(newCell, (*c)[foundIndex+1:]...)
	*c = newCell
	return evicted
}

func GetSubStoreID(ip string, secret []byte, bucketNum int, sourceBytes []byte) (int, error) {
	network, err := GetNetwork(ip)
	if err != nil {
		return 0, err
	}
	// If network is not IPV4, put in different pucket
	if network != networkIPV4 {
		input := append(secret, uint8(network)) //nolint: gocritic // intending to combine secret and network as input
		hashedNum := binary.BigEndian.Uint32(crypto.Hash(input))
		return int(hashedNum) % bucketNum, nil
	}
	ipBytes, err := ipToBytes(ip)
	if err != nil {
		return 0, nil
	}
	// Calculate k
	// k = Hash(random_secret, IP) % 4
	// k = Hash(random_secret, source_group, group) % 16
	networkBytes := [1]byte{uint8(network)}
	targetBytes := bytes.Join(secret, []byte{uint8(network)})
	if sourceBytes != nil {
		targetBytes = append(targetBytes, sourceBytes[0:2]...)
		targetBytes = append(targetBytes, ipBytes[0:2]...)
	} else {
		targetBytes = append(targetBytes, ipBytes...)
	}
	hashedTarget := binary.BigEndian.Uint32(crypto.Hash(targetBytes))
	mod := bucketModTriedPeer
	if sourceBytes != nil {
		mod = bucketModNewPeer
	}
	k := hashedTarget % uint32(mod)
	kBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(kBytes, k)
	bucketBytes := bytes.Join(secret, networkBytes[0:1])
	if sourceBytes != nil {
		bucketBytes = append(bucketBytes, sourceBytes[0:2]...)
	} else {
		bucketBytes = append(bucketBytes, targetBytes[0:2]...)
	}
	bucketBytes = append(bucketBytes, kBytes...)
	bucket := binary.BigEndian.Uint32(crypto.Hash(bucketBytes))
	return int(bucket) % bucketNum, nil
}

func getSourceBytes(addr *Address) ([]byte, error) {
	if addr.source == nil {
		return nil, nil
	}
	return ipToBytes(*addr.source)
}
