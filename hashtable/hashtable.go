package hashtable

import (
	"data-ownership-system/pkg/types"
	"data-ownership-system/utils"
	"fmt"
)

type HashTable struct {
	entries    []*types.HashTableEntry
	size       int
	count      int
	loadFactor float64
	version    int
}

func NewHashTable(initialSize int, loadFactor float64) *HashTable {
	actualSize := utils.NextPowerOfTwo(initialSize)

	entries := make([]*types.HashTableEntry, actualSize)
	for i := range entries {
		entries[i] = types.NewEmptyEntry()
	}

	return &HashTable{
		entries:    entries,
		size:       actualSize,
		count:      0,
		loadFactor: loadFactor,
		version:    0,
	}
}

func (ht *HashTable) Insert(url string, hash []byte) (int, error) {
	if url == "" {
		return -1, fmt.Errorf("URL cannot be empty")
	}
	if len(hash) == 0 {
		return -1, fmt.Errorf("hash cannot be empty")
	}
	if len(hash) != 32 {
		return -1, fmt.Errorf("hash length must be 32 bytes, got %d bytes", len(hash))
	}

	currentLoadFactor := float64(ht.count) / float64(ht.size)
	if currentLoadFactor >= ht.loadFactor {
		err := ht.resize()
		if err != nil {
			return -1, fmt.Errorf("resize failed: %v", err)
		}
	}

	index, err := ht.findSlot(url, true)
	if err != nil {
		return -1, fmt.Errorf("failed to find insertion slot: %v", err)
	}

	entry := ht.entries[index]
	wasEmpty := entry.IsAvailable()

	entry.URL = url
	entry.State = 0
	entry.Hash = hash

	if wasEmpty {
		ht.count++
	}

	return index, nil
}

func (ht *HashTable) Get(url string) (*types.HashTableEntry, bool) {
	index, err := ht.findSlot(url, false)
	if err != nil {
		return nil, false
	}

	entry := ht.entries[index]
	if entry.IsEmpty() || entry.URL != url {
		return nil, false
	}

	if entry.IsDeleted() {
		return nil, false
	}

	return entry, true
}

func (ht *HashTable) GetWithIndex(url string) (*types.HashTableEntry, int, bool) {
	index, err := ht.findSlot(url, false)
	if err != nil {
		return nil, -1, false
	}

	entry := ht.entries[index]
	if entry.IsEmpty() || entry.URL != url {
		return nil, -1, false
	}

	if entry.IsDeleted() {
		return nil, -1, false
	}

	return entry, index, true
}

func (ht *HashTable) Delete(url string) (int, bool) {
	index, err := ht.findSlot(url, false)
	if err != nil {
		return -1, false
	}

	entry := ht.entries[index]
	if entry.IsEmpty() || entry.URL != url || entry.IsDeleted() {
		return -1, false
	}

	entry.State = 1
	entry.URL = types.DeletedSlotURL
	ht.count--

	return index, true
}

func (ht *HashTable) Update(url string, newHash []byte) (int, bool) {
	index, err := ht.findSlot(url, false)
	if err != nil {
		return -1, false
	}

	entry := ht.entries[index]
	if entry.IsEmpty() || entry.URL != url || entry.IsDeleted() {
		return -1, false
	}

	entry.Hash = newHash
	return index, true
}

func (ht *HashTable) findSlot(url string, forInsert bool) (int, error) {
	h1 := utils.Hash1(url, ht.size)
	h2 := utils.Hash2(url, ht.size)

	mask := ht.size - 1
	for i := 0; i < ht.size; i++ {
		index := (h1 + i*h2) & mask
		if ht.checkSlot(index, url, forInsert) {
			return index, nil
		}
	}

	return -1, fmt.Errorf("could not find a suitable slot")
}

func (ht *HashTable) checkSlot(index int, url string, forInsert bool) bool {
	entry := ht.entries[index]

	if forInsert {
		return entry.IsAvailable() || entry.URL == url
	} else {
		if entry.URL == url {
			return true
		}
		if entry.IsEmpty() && !entry.IsDeleted() {
			return true
		}
		return false
	}
}

func (ht *HashTable) resize() error {
	oldEntries := ht.entries
	oldSize := ht.size

	ht.size = oldSize * 2
	ht.entries = make([]*types.HashTableEntry, ht.size)
	ht.count = 0
	ht.version++

	for i := range ht.entries {
		ht.entries[i] = types.NewEmptyEntry()
	}

	for _, entry := range oldEntries {
		if !entry.IsEmpty() && !entry.IsDeleted() {
			_, err := ht.Insert(entry.URL, entry.Hash)
			if err != nil {
				return fmt.Errorf("failed to reinsert data: %v", err)
			}
		}
	}

	return nil
}

func (ht *HashTable) GetAllEntries() []*types.HashTableEntry {
	return ht.entries
}

func (ht *HashTable) Size() int {
	return ht.size
}

func (ht *HashTable) Count() int {
	return ht.count
}

func (ht *HashTable) LoadFactor() float64 {
	return float64(ht.count) / float64(ht.size)
}

func (ht *HashTable) String() string {
	return fmt.Sprintf("HashTable{size: %d, count: %d, loadFactor: %.2f}",
		ht.size, ht.count, ht.LoadFactor())
}

func (ht *HashTable) GetEntry(index int) *types.HashTableEntry {
	if index < 0 || index >= ht.size {
		return nil
	}
	return ht.entries[index]
}

func (ht *HashTable) Version() int {
	return ht.version
}
