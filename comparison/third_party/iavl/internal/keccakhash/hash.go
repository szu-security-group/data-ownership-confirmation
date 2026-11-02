// Package keccakhash supplies the experiment's 256-bit hash substitution.
// Tree operations, node encoding and balancing remain those of IAVL v1.3.6.
package keccakhash

import (
	"golang.org/x/crypto/sha3"
	"hash"
)

const Size = 32

func New() hash.Hash                 { return sha3.NewLegacyKeccak256() }
func Sum256(b []byte) (out [32]byte) { h := New(); h.Write(b); h.Sum(out[:0]); return }
