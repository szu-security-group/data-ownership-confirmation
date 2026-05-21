package types

import (
	"data-ownership-system/utils"
	"fmt"
)

type DataItem struct {
	URL     string
	Content []byte
	Hash    []byte
}

type HashTableEntry struct {
	URL   string
	State int
	Hash  []byte
}

const EmptySlotURL = "__EMPTY_SLOT__"

const DeletedSlotURL = "__DELETED_SLOT__"

func NewDataItem(url string, content []byte) *DataItem {
	hash := utils.Keccak256(content)
	return &DataItem{
		URL:     url,
		Content: content,
		Hash:    hash,
	}
}

func NewHashTableEntry(url string, state int, hash []byte) *HashTableEntry {
	return &HashTableEntry{
		URL:   url,
		State: state,
		Hash:  hash,
	}
}

func NewEmptyEntry() *HashTableEntry {
	return &HashTableEntry{
		URL:   EmptySlotURL,
		State: 0,
		Hash:  make([]byte, 32),
	}
}

func (e *HashTableEntry) IsEmpty() bool {
	return e.URL == EmptySlotURL
}

func (e *HashTableEntry) IsDeleted() bool {
	return e.State == 1
}

func (e *HashTableEntry) IsAvailable() bool {
	return e.IsEmpty() || e.IsDeleted()
}

func (e *HashTableEntry) String() string {
	return fmt.Sprintf("Entry{URL: %s, State: %d, Hash: %x}", e.URL, e.State, e.Hash[:8])
}

func (e *HashTableEntry) Equal(other *HashTableEntry) bool {
	if e == nil || other == nil {
		return e == other
	}
	return e.URL == other.URL && e.State == other.State &&
		string(e.Hash) == string(other.Hash)
}
