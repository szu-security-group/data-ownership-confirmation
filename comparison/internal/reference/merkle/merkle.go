package merkle

import (
	"bytes"
	"data-ownership-system/pkg/types"
	"data-ownership-system/utils"
	"fmt"
)

type MerkleNode struct {
	Hash   []byte
	Left   *MerkleNode
	Right  *MerkleNode
	IsLeaf bool
	Parent *MerkleNode
}

type MerkleTree struct {
	Root   *MerkleNode
	Leaves []*MerkleNode
	Data   []*types.HashTableEntry
}

type ProofPath struct {
	Index    int
	Siblings [][]byte
	Root     []byte
}

func NewMerkleTree(entries []*types.HashTableEntry) *MerkleTree {
	if len(entries) == 0 {
		return &MerkleTree{}
	}

	leaves := make([]*MerkleNode, len(entries))
	for i, entry := range entries {
		entryData := []byte(entry.URL)
		entryData = append(entryData, byte(entry.State))
		entryData = append(entryData, entry.Hash...)

		leafHash := utils.Keccak256(entryData)
		leaves[i] = &MerkleNode{
			Hash:   leafHash,
			IsLeaf: true,
		}
	}

	root := buildTree(leaves)

	return &MerkleTree{
		Root:   root,
		Leaves: leaves,
		Data:   entries,
	}
}

func buildTree(nodes []*MerkleNode) *MerkleNode {
	if len(nodes) == 1 {
		return nodes[0]
	}

	var nextLevel []*MerkleNode

	for i := 0; i < len(nodes); i += 2 {
		left := nodes[i]
		var right *MerkleNode

		if i+1 < len(nodes) {
			right = nodes[i+1]
		} else {
			right = &MerkleNode{
				Hash:   make([]byte, len(left.Hash)),
				IsLeaf: false,
			}
			copy(right.Hash, left.Hash)
		}

		combinedHash := append(left.Hash, right.Hash...)
		parentHash := utils.Keccak256(combinedHash)

		parent := &MerkleNode{
			Hash:   parentHash,
			Left:   left,
			Right:  right,
			IsLeaf: false,
		}
		left.Parent = parent
		right.Parent = parent

		nextLevel = append(nextLevel, parent)
	}

	return buildTree(nextLevel)
}

func (mt *MerkleTree) GetRoot() []byte {
	if mt.Root == nil {
		return nil
	}
	return mt.Root.Hash
}

func (mt *MerkleTree) GenerateProofByIndex(index int) (*ProofPath, error) {
	if index < 0 || index >= len(mt.Data) {
		return nil, fmt.Errorf("index out of range: %d", index)
	}

	siblings := mt.getSiblings(index)

	return &ProofPath{
		Index:    index,
		Siblings: siblings,
		Root:     mt.GetRoot(),
	}, nil
}

func (mt *MerkleTree) getSiblings(leafIndex int) [][]byte {
	var siblings [][]byte

	currentNode := mt.Leaves[leafIndex]
	for parent := currentNode.Parent; parent != nil; parent = parent.Parent {
		if parent.Left == currentNode {
			siblings = append(siblings, parent.Right.Hash)
		} else {
			siblings = append(siblings, parent.Left.Hash)
		}
		currentNode = parent
	}

	return siblings
}

func VerifyProof(proof *ProofPath, targetEntry *types.HashTableEntry) bool {
	if proof == nil || targetEntry == nil {
		return false
	}

	entryData := []byte(targetEntry.URL)
	entryData = append(entryData, byte(targetEntry.State))
	entryData = append(entryData, targetEntry.Hash...)
	currentHash := utils.Keccak256(entryData)

	currentIndex := proof.Index
	for _, siblingHash := range proof.Siblings {
		if currentIndex%2 == 0 {
			combinedHash := append(currentHash, siblingHash...)
			currentHash = utils.Keccak256(combinedHash)
		} else {
			combinedHash := append(siblingHash, currentHash...)
			currentHash = utils.Keccak256(combinedHash)
		}
		currentIndex = currentIndex / 2
	}

	return bytes.Equal(currentHash, proof.Root)
}

func (mt *MerkleTree) UpdateLeaf(index int, newEntry *types.HashTableEntry) error {
	if index < 0 || index >= len(mt.Data) {
		return fmt.Errorf("index out of range: %d", index)
	}

	mt.Data[index] = newEntry

	entryData := []byte(newEntry.URL)
	entryData = append(entryData, byte(newEntry.State))
	entryData = append(entryData, newEntry.Hash...)
	mt.Leaves[index].Hash = utils.Keccak256(entryData)

	mt.updatePathToRoot(index)

	return nil
}

func (mt *MerkleTree) updatePathToRoot(leafIndex int) {
	if mt.Root == nil || len(mt.Leaves) <= 1 {
		if len(mt.Leaves) == 1 {
			mt.Root = mt.Leaves[0]
		}
		return
	}

	for node := mt.Leaves[leafIndex]; node != nil && node.Parent != nil; node = node.Parent {
		parent := node.Parent
		leftHash := parent.Left.Hash
		rightHash := parent.Right.Hash
		combinedHash := append(leftHash, rightHash...)
		parent.Hash = utils.Keccak256(combinedHash)
	}
}

func (mt *MerkleTree) AddLeaf(newEntry *types.HashTableEntry, slotIndex int) error {
	entryData := []byte(newEntry.URL)
	entryData = append(entryData, byte(newEntry.State))
	entryData = append(entryData, newEntry.Hash...)
	leafHash := utils.Keccak256(entryData)

	if slotIndex < 0 || slotIndex >= len(mt.Leaves) {
		return fmt.Errorf("slot index out of range: %d", slotIndex)
	}

	mt.Data[slotIndex] = newEntry
	mt.Leaves[slotIndex].Hash = leafHash

	mt.updatePathToRoot(slotIndex)

	return nil
}

func (mt *MerkleTree) RemoveLeaf(slotIndex int) error {
	if slotIndex < 0 || slotIndex >= len(mt.Leaves) {
		return fmt.Errorf("slot index out of range: %d", slotIndex)
	}

	deletedEntry := &types.HashTableEntry{
		URL:   types.DeletedSlotURL,
		State: 1,
		Hash:  mt.Data[slotIndex].Hash,
	}
	mt.Data[slotIndex] = deletedEntry

	entryData := []byte(deletedEntry.URL)
	entryData = append(entryData, byte(deletedEntry.State))
	entryData = append(entryData, deletedEntry.Hash...)
	mt.Leaves[slotIndex].Hash = utils.Keccak256(entryData)

	mt.updatePathToRoot(slotIndex)

	return nil
}

func (mt *MerkleTree) String() string {
	if mt.Root == nil {
		return "MerkleTree{empty}"
	}
	return fmt.Sprintf("MerkleTree{root: %x, leaves: %d}", mt.Root.Hash[:8], len(mt.Leaves))
}
