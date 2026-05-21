package auth

import (
	"bytes"
	"data-ownership-system/hashtable"
	"data-ownership-system/merkle"
	"data-ownership-system/pkg/types"
	"data-ownership-system/utils"
	"fmt"
	"strings"
	"sync"
)

type AuthenticatedDataStructure struct {
	HashTables  map[string]*hashtable.HashTable
	MerkleTrees map[string]*merkle.MerkleTree
	SubRoots    map[string][]byte
	TopRoot     []byte
	LoadFactor  float64
	DataTypes   []string
}

func NewAuthenticatedDataStructure(loadFactor float64) *AuthenticatedDataStructure {
	return &AuthenticatedDataStructure{
		HashTables:  make(map[string]*hashtable.HashTable),
		MerkleTrees: make(map[string]*merkle.MerkleTree),
		SubRoots:    make(map[string][]byte),
		LoadFactor:  loadFactor,
		DataTypes:   make([]string, 0),
	}
}

func (ads *AuthenticatedDataStructure) AddDataType(dataType string, initialSize int) {
	if _, exists := ads.HashTables[dataType]; !exists {
		ads.HashTables[dataType] = hashtable.NewHashTable(initialSize, ads.LoadFactor)
		ads.DataTypes = append(ads.DataTypes, dataType)
	}
}

func (ads *AuthenticatedDataStructure) BuildFromDataItems(dataItems []*types.DataItem, dataTypes []string, typeCounts map[string]int) error {
	ads.DataTypes = make([]string, len(dataTypes))
	copy(ads.DataTypes, dataTypes)

	typeGroups := make(map[string][]*types.DataItem)

	if typeCounts != nil {
		for _, dataType := range dataTypes {
			typeGroups[dataType] = make([]*types.DataItem, 0, typeCounts[dataType])
		}
	}

	for _, item := range dataItems {
		dataType := ads.extractDataType(item.URL)
		if typeGroups[dataType] == nil {
			typeGroups[dataType] = make([]*types.DataItem, 0)
		}
		typeGroups[dataType] = append(typeGroups[dataType], item)
	}

	fmt.Println("Data type distribution:")
	totalItems := len(dataItems)
	for _, dataType := range dataTypes {
		count := len(typeGroups[dataType])
		percentage := float64(count) / float64(totalItems) * 100
		fmt.Printf("  %s: %d items (%.1f%%)\n", dataType, count, percentage)
	}

	return ads.buildFromTypeGroups(typeGroups, dataTypes)
}

func (ads *AuthenticatedDataStructure) buildFromTypeGroups(typeGroups map[string][]*types.DataItem, dataTypes []string) error {

	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, len(dataTypes))

	for _, dataType := range dataTypes {
		wg.Add(1)
		go func(dt string) {
			defer wg.Done()

			items := typeGroups[dt]
			if items == nil {
				items = make([]*types.DataItem, 0)
			}

			initialSize := int(float64(len(items)) / ads.LoadFactor)
			if initialSize < 16 {
				initialSize = 16
			}

			ht := hashtable.NewHashTable(initialSize, ads.LoadFactor)

			for _, item := range items {
				_, err := ht.Insert(item.URL, item.Hash)
				if err != nil {
					errChan <- fmt.Errorf("failed to insert data item into hash table: %v", err)
					return
				}
			}

			entries := ht.GetAllEntries()
			mt := merkle.NewMerkleTree(entries)

			mu.Lock()
			if ads.HashTables == nil {
				ads.HashTables = make(map[string]*hashtable.HashTable)
			}
			if ads.MerkleTrees == nil {
				ads.MerkleTrees = make(map[string]*merkle.MerkleTree)
			}
			if ads.SubRoots == nil {
				ads.SubRoots = make(map[string][]byte)
			}

			ads.HashTables[dt] = ht
			ads.MerkleTrees[dt] = mt
			ads.SubRoots[dt] = mt.GetRoot()
			mu.Unlock()

		}(dataType)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return <-errChan
	}

	ads.calculateTopRoot()

	return nil
}

func (ads *AuthenticatedDataStructure) extractDataType(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return "default"
}

func (ads *AuthenticatedDataStructure) calculateTopRoot() {
	if len(ads.DataTypes) == 0 {
		ads.TopRoot = nil
		return
	}

	var combinedHash []byte
	for _, dataType := range ads.DataTypes {
		if root := ads.SubRoots[dataType]; root != nil {
			combinedHash = append(combinedHash, root...)
		}
	}

	if len(combinedHash) > 0 {
		ads.TopRoot = utils.Keccak256(combinedHash)
	}
}

func (ads *AuthenticatedDataStructure) GenerateProof(url string, hash []byte) (*CombinedProof, error) {
	dataType := ads.extractDataType(url)

	ht, exists := ads.HashTables[dataType]
	if !exists {
		return nil, fmt.Errorf("data type %s does not exist", dataType)
	}

	entry, slotIndex, found := ht.GetWithIndex(url)
	if !found {
		return nil, fmt.Errorf("data item with URL %s not found", url)
	}

	if !bytes.Equal(entry.Hash, hash) {
		return nil, fmt.Errorf("hash mismatch")
	}

	mt := ads.MerkleTrees[dataType]
	merkleProof, err := mt.GenerateProofByIndex(slotIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Merkle proof: %v", err)
	}

	otherRoots := make(map[string][]byte)
	for _, dt := range ads.DataTypes {
		if dt != dataType {
			otherRoots[dt] = ads.SubRoots[dt]
		}
	}

	return &CombinedProof{
		URL:         url,
		Hash:        hash,
		DataType:    dataType,
		MerkleProof: merkleProof,
		OtherRoots:  otherRoots,
		TopRoot:     ads.TopRoot,
	}, nil
}

type CombinedProof struct {
	URL         string
	Hash        []byte
	DataType    string
	MerkleProof *merkle.ProofPath
	OtherRoots  map[string][]byte
	TopRoot     []byte
}

func (ads *AuthenticatedDataStructure) VerifyProof(proof *CombinedProof) bool {
	if proof == nil {
		return false
	}

	entry := &types.HashTableEntry{
		URL:   proof.URL,
		State: 0,
		Hash:  proof.Hash,
	}

	if !merkle.VerifyProof(proof.MerkleProof, entry) {
		return false
	}

	var combinedHash []byte
	for _, dataType := range ads.DataTypes {
		if dataType == proof.DataType {
			combinedHash = append(combinedHash, proof.MerkleProof.Root...)
		} else {
			if root, exists := proof.OtherRoots[dataType]; exists {
				combinedHash = append(combinedHash, root...)
			}
		}
	}

	calculatedTopRoot := utils.Keccak256(combinedHash)
	return bytes.Equal(calculatedTopRoot, proof.TopRoot)
}

func (ads *AuthenticatedDataStructure) InsertDataItem(item *types.DataItem) error {
	dataType := ads.extractDataType(item.URL)

	if _, exists := ads.HashTables[dataType]; !exists {
		ads.AddDataType(dataType, 16)
	}

	ht := ads.HashTables[dataType]
	beforeVersion := ht.Version()
	slotIndex, err := ht.Insert(item.URL, item.Hash)
	if err != nil {
		return fmt.Errorf("failed to insert data item: %v", err)
	}

	mt := ads.MerkleTrees[dataType]
	if mt == nil {
		entries := ht.GetAllEntries()
		mt = merkle.NewMerkleTree(entries)
		ads.MerkleTrees[dataType] = mt
	} else {
		afterVersion := ht.Version()
		if afterVersion != beforeVersion {
			entries := ht.GetAllEntries()
			mt = merkle.NewMerkleTree(entries)
			ads.MerkleTrees[dataType] = mt
		} else {
			newEntry := &types.HashTableEntry{
				URL:   item.URL,
				State: 0,
				Hash:  item.Hash,
			}
			err = mt.AddLeaf(newEntry, slotIndex)
			if err != nil {
				return fmt.Errorf("failed to incrementally add Merkle leaf: %v", err)
			}
		}
	}
	ads.SubRoots[dataType] = mt.GetRoot()

	ads.calculateTopRoot()

	return nil
}

func (ads *AuthenticatedDataStructure) UpdateDataItem(url string, newHash []byte) error {
	dataType := ads.extractDataType(url)

	ht, exists := ads.HashTables[dataType]
	if !exists {
		return fmt.Errorf("data type %s does not exist", dataType)
	}

	slotIndex, success := ht.Update(url, newHash)
	if !success {
		return fmt.Errorf("failed to update data item, URL: %s", url)
	}

	mt := ads.MerkleTrees[dataType]
	if mt == nil {
		return fmt.Errorf("Merkle tree does not exist, data type: %s", dataType)
	}

	updatedEntry := &types.HashTableEntry{
		URL:   url,
		State: 0,
		Hash:  newHash,
	}
	err := mt.UpdateLeaf(slotIndex, updatedEntry)
	if err != nil {
		return fmt.Errorf("failed to incrementally update Merkle tree: %v", err)
	}

	ads.SubRoots[dataType] = mt.GetRoot()

	ads.calculateTopRoot()

	return nil
}

func (ads *AuthenticatedDataStructure) DeleteDataItem(url string) error {
	dataType := ads.extractDataType(url)

	ht, exists := ads.HashTables[dataType]
	if !exists {
		return fmt.Errorf("data type %s does not exist", dataType)
	}

	slotIndex, success := ht.Delete(url)
	if !success {
		return fmt.Errorf("failed to delete data item, URL: %s", url)
	}

	mt := ads.MerkleTrees[dataType]
	if mt == nil {
		return fmt.Errorf("Merkle tree does not exist, data type: %s", dataType)
	}

	err := mt.RemoveLeaf(slotIndex)
	if err != nil {
		return fmt.Errorf("failed to incrementally delete Merkle leaf: %v", err)
	}

	ads.SubRoots[dataType] = mt.GetRoot()

	ads.calculateTopRoot()

	return nil
}

func (ads *AuthenticatedDataStructure) GetTopRoot() []byte {
	return ads.TopRoot
}

func (ads *AuthenticatedDataStructure) GetStatistics() map[string]interface{} {
	stats := make(map[string]interface{})

	totalItems := 0
	totalCapacity := 0

	for dataType, ht := range ads.HashTables {
		totalItems += ht.Count()
		totalCapacity += ht.Size()

		stats[fmt.Sprintf("%s_count", dataType)] = ht.Count()
		stats[fmt.Sprintf("%s_capacity", dataType)] = ht.Size()
		stats[fmt.Sprintf("%s_load_factor", dataType)] = ht.LoadFactor()
	}

	stats["total_items"] = totalItems
	stats["total_capacity"] = totalCapacity
	stats["data_types"] = len(ads.DataTypes)
	stats["top_root"] = fmt.Sprintf("%x", ads.TopRoot)

	return stats
}

func (ads *AuthenticatedDataStructure) String() string {
	return fmt.Sprintf("AuthDataStructure{types: %d, items: %d, root: %x}",
		len(ads.DataTypes),
		func() int {
			total := 0
			for _, ht := range ads.HashTables {
				total += ht.Count()
			}
			return total
		}(),
		ads.TopRoot[:8])
}
