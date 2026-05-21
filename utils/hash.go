package utils

import (
	"golang.org/x/crypto/sha3"
)

func Hash1(key string, tableSize int) int {
	hash := uint32(0)
	for i := 0; i < len(key); i++ {
		hash = hash*31 + uint32(key[i])
	}
	return int(hash) & (tableSize - 1)
}

func Hash2(key string, tableSize int) int {
	hash := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= 31
	}

	result := int(hash) & (tableSize - 1)
	if result%2 == 0 {
		result++
	}
	if result == 0 {
		result = 1
	}
	return result
}

func Keccak256(data []byte) []byte {
	hash := sha3.NewLegacyKeccak256()
	hash.Write(data)
	return hash.Sum(nil)
}

func NextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}

	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++

	return n
}

func IsPowerOfTwo(n int) bool {
	return n > 0 && (n&(n-1)) == 0
}
