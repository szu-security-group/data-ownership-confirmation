// Package comparison implements current-snapshot authenticated dictionaries.
package comparison

import (
	"encoding/binary"
	"errors"
	"fmt"
	"golang.org/x/crypto/sha3"
	"hash"
	"sync"
)

type Hash [32]byte
type Item struct {
	URL      string
	Digest   Hash
	Category int
}
type Metrics struct {
	HashCalls     uint64 `json:"hash_calls"`
	HashBytes     uint64 `json:"hash_bytes"`
	Rebuilds      uint64 `json:"rebuilds"`
	RehashedItems uint64 `json:"rehashed_items"`
	Rotations     uint64 `json:"rotations"`
	Probes        uint64 `json:"probes"`
}

func (a Metrics) Add(b Metrics) Metrics {
	return Metrics{a.HashCalls + b.HashCalls, a.HashBytes + b.HashBytes, a.Rebuilds + b.Rebuilds, a.RehashedItems + b.RehashedItems, a.Rotations + b.Rotations, a.Probes + b.Probes}
}
func (a Metrics) Sub(b Metrics) Metrics {
	return Metrics{a.HashCalls - b.HashCalls, a.HashBytes - b.HashBytes, a.Rebuilds - b.Rebuilds, a.RehashedItems - b.RehashedItems, a.Rotations - b.Rotations, a.Probes - b.Probes}
}

var pool = sync.Pool{New: func() any { return sha3.NewLegacyKeccak256() }}

func Sum(parts ...[]byte) (out Hash) {
	h := pool.Get().(hash.Hash)
	h.Reset()
	for _, p := range parts {
		h.Write(p)
	}
	h.Sum(out[:0])
	pool.Put(h)
	return
}

type hasher struct {
	h     hash.Hash
	stats *Metrics
	size  uint64
}

func newHasher(m *Metrics) *hasher            { return &hasher{h: sha3.NewLegacyKeccak256(), stats: m} }
func (h *hasher) Write(b []byte) (int, error) { h.size += uint64(len(b)); return h.h.Write(b) }
func (h *hasher) Sum(b []byte) []byte {
	h.stats.HashCalls++
	h.stats.HashBytes += h.size
	return h.h.Sum(b)
}
func (h *hasher) Reset()         { h.h.Reset(); h.size = 0 }
func (h *hasher) Size() int      { return 32 }
func (h *hasher) BlockSize() int { return h.h.BlockSize() }
func (h *hasher) digest(p ...[]byte) (v Hash) {
	h.Reset()
	for _, b := range p {
		h.Write(b)
	}
	h.Sum(v[:0])
	return
}

var Empty = Sum([]byte{2})

var ErrMissing = errors.New("key not found")
var ErrExists = errors.New("key already exists")

type ADS interface {
	Build([]Item) error
	Insert(Item) error
	Modify(Item) error
	Delete(string) error
	Root() Hash
	Prove(string) ([]byte, error)
	GenerateProof(string) (NativeProof, error)
	Stats() Metrics
}

var Kinds = []string{"ht", "merkle", "iavl", "smt", "mpt"}

func New(kind string, load float64) (ADS, error) {
	switch kind {
	case "ht":
		return newDense(kind, load), nil
	case "merkle":
		return newMerkle(), nil
	case "iavl":
		return newIAVL(), nil
	case "smt":
		return newSMT(), nil
	case "mpt":
		return newMPT(), nil
	}
	return nil, fmt.Errorf("unknown ADS %q", kind)
}
func power2(n int) int {
	r := 1
	for r < n {
		r *= 2
	}
	return r
}
func binaryRoot(i Item, p []byte) (Hash, error) {
	if len(p) < 2 {
		return Hash{}, errors.New("short proof")
	}
	n := int(binary.BigEndian.Uint16(p))
	if n > 256 || len(p) != 2+33*n {
		return Hash{}, errors.New("invalid path length")
	}
	m := Metrics{}
	h := newHasher(&m)
	r := h.digest([]byte(i.URL), []byte{0}, i.Digest[:])
	for j := 0; j < n; j++ {
		d := p[2+33*j]
		var s Hash
		copy(s[:], p[3+33*j:35+33*j])
		if d == 0 {
			r = h.digest(r[:], s[:])
		} else if d == 1 {
			r = h.digest(s[:], r[:])
		} else {
			return Hash{}, errors.New("invalid direction")
		}
	}
	return r, nil
}
func VerifySub(kind string, root Hash, item Item, p []byte) bool {
	switch kind {
	case "ht":
		r, e := binaryRoot(item, p)
		return e == nil && r == root
	case "merkle", "iavl":
		r, e := subRoot(kind, item, p)
		return e == nil && r == root
	case "smt":
		return verifySMT(root, item, p)
	case "mpt":
		return verifyMPT(root, item, p)
	}
	return false
}
