package addressbook

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/collection/ints"
)

const (
	maxReturnNumber          = 1000
	newAddressesBucketSize   = 128
	triedAddressesBucketSize = 64
)

// Book keep track of peer addresses.
type Book struct {
	mutex                *sync.Mutex
	newAddresses         *bucket
	triedAddresses       *bucket
	seedAddresses        []*Address
	fixedAddresses       []*Address
	whitelistedAddresses []*Address
	bannedPeers          map[string]*Address
}

// NewBook return new book with intiial addresses.
func NewBook(secret []byte, initialAddresses []*Address, seedAddresses []*Address, fixedAddresses []*Address, whitelistedAddresses []*Address) *Book {
	book := &Book{
		mutex:                new(sync.Mutex),
		newAddresses:         newBucket(newAddressesBucketSize, secret),
		triedAddresses:       newBucket(triedAddressesBucketSize, secret),
		seedAddresses:        seedAddresses,
		fixedAddresses:       fixedAddresses,
		whitelistedAddresses: whitelistedAddresses,
		bannedPeers:          map[string]*Address{},
	}
	for _, addr := range initialAddresses {
		if err := book.AddTriedAddress(addr); err != nil {
			panic(err)
		}
	}
	return book
}

// GetAddressByIP returns address if exist. return bool if exist or not.
func (b *Book) GetAddressByIP(ip string) (*Address, bool) {
	addr, exist := b.newAddresses.get(ip)
	if exist {
		return addr, true
	}
	addr, exist = b.triedAddresses.get(ip)
	if exist {
		return addr, true
	}
	return nil, false
}

// GetAddress returns an address.
func (b *Book) GetAddress(num int, excludeIPs []string) []*Address {
	if num <= 0 {
		return nil
	}
	b.mutex.Lock()
	defer b.mutex.Unlock()
	currentSize := b.newAddresses.size() + b.triedAddresses.size()
	if currentSize == 0 {
		return nil
	}
	if b.triedAddresses.size() < 100 {
		allList := []*Address{}
		allList = append(allList, b.triedAddresses.getAll(excludeIPs)...)
		allList = append(allList, b.newAddresses.getAll(excludeIPs)...)
		rand.Shuffle(len(allList), func(i, j int) {
			allList[i], allList[j] = allList[j], allList[i]
		})
		if len(allList) <= num {
			return allList
		}
		return allList[:num]
	}
	threshold := math.Max(float64(b.TriedAddressSize())/float64(currentSize), 0.5)
	val := rand.Float64()
	var list []*Address
	if val < threshold {
		// return from tried peer
		list = b.triedAddresses.getAll(excludeIPs)
	} else {
		list = b.newAddresses.getAll(excludeIPs)
	}
	rand.Shuffle(len(list), func(i, j int) {
		list[i], list[j] = list[j], list[i]
	})
	if len(list) <= num {
		return list
	}
	return list[:num]
}

// GetAddresses returns known addresses.
func (b *Book) GetAddresses(max int) []*Address {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	currentSize := b.newAddresses.size() + b.triedAddresses.size()
	if currentSize == 0 {
		return []*Address{}
	}
	rangeLow := ints.Min(maxReturnNumber, currentSize/4)
	rangeHigh := ints.Min(maxReturnNumber, currentSize/2)
	n := rand.Intn(rangeHigh-rangeLow+1) + rangeLow
	amount := ints.Max(n, ints.Min(100, currentSize))

	// get all list
	allList := append(b.triedAddresses.getAll(nil), b.newAddresses.getAll(nil)...)
	rand.Shuffle(len(allList), func(i, j int) {
		allList[i], allList[j] = allList[j], allList[i]
	})
	return allList[:ints.Min(amount, len(allList))]
}

// All returns all known addresses.
func (b *Book) All() []*Address {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return append(b.triedAddresses.getAll(nil), b.newAddresses.getAll(nil)...)
}

// AddNewAddress inserts new address to new bucket.
func (b *Book) AddNewAddress(addr *Address) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	exist, err := b.triedAddresses.exist(addr)
	if err != nil {
		return err
	}
	if exist {
		return nil
	}
	_, err = b.newAddresses.insert(addr)
	if err != nil {
		return err
	}
	return nil
}

// AddTriedAddress inserts new address to new bucket.
func (b *Book) AddTriedAddress(addr *Address) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	exist, err := b.newAddresses.exist(addr)
	if err != nil {
		return err
	}
	if exist {
		return nil
	}
	_, err = b.triedAddresses.insert(addr)
	if err != nil {
		return err
	}
	return nil
}

// MoveNewAddressToTried move new address to tried.
func (b *Book) MoveNewAddressToTried(addr *Address) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	err := b.newAddresses.remove(addr)
	if err != nil {
		return err
	}
	_, err = b.triedAddresses.insert(addr)
	if err != nil {
		return err
	}
	return nil
}

// MoveTriedAddressToNew move tried to new address.
func (b *Book) MoveTriedAddressToNew(addr *Address) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	err := b.triedAddresses.remove(addr)
	if err != nil {
		return err
	}
	_, err = b.newAddresses.insert(addr)
	if err != nil {
		return err
	}
	return nil
}

// Size returns current size.
func (b *Book) Size() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.triedAddresses.size() + b.newAddresses.size()
}

// TriedAddressSize returns current size.
func (b *Book) TriedAddressSize() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.triedAddresses.size()
}

// NewAddressSize returns current size.
func (b *Book) NewAddressSize() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.newAddresses.size()
}

// SeedAddresses returns seed addresses.
func (b *Book) SeedAddresses() []*Address {
	return b.seedAddresses
}

// FixedAddresses returns fixed addresses.
func (b *Book) FixedAddresses() []*Address {
	return b.fixedAddresses
}

// WhitelistedAddresses returns whitelisted addresses.
func (b *Book) WhitelistedAddresses() []*Address {
	return b.whitelistedAddresses
}

// BanAddress bans the address for 24hrs.
func (b *Book) BanAddress(addr *Address) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	// remove from new and tried peer
	b.newAddresses.remove(addr)   //nolint: errcheck // failing error is ignored since it cannot recover and would not harm
	b.triedAddresses.remove(addr) //nolint: errcheck // failing error is ignored since it cannot recover and would not harm
	now := time.Now()
	addr.bannedAt = &now
	b.bannedPeers[addr.IP()] = addr
}

// IsBanned checks if the address is banned or not.
func (b *Book) IsBanned(addr *Address) bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	bannedPeer, exist := b.bannedPeers[addr.IP()]
	if exist {
		if bannedPeer.bannedAt.Add(24 * time.Hour).Before(time.Now()) {
			return true
		}
		// No longger banned
		delete(b.bannedPeers, addr.IP())
	}
	return false
}

// CheckBanning checks all address banning status.
func (b *Book) CheckBanning() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for k, v := range b.bannedPeers {
		if v.bannedAt == nil || v.bannedAt.Add(24*time.Hour).After(time.Now()) {
			delete(b.bannedPeers, k)
		}
	}
}
