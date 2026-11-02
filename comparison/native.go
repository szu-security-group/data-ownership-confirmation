package comparison

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/celestiaorg/smt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie"
	"golang.org/x/crypto/sha3"
)

type SMT struct {
	t     *smt.SparseMerkleTree
	stats Metrics
}

func newSMT() *SMT {
	s := &SMT{}
	s.t = smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), newHasher(&s.stats))
	return s
}
func (s *SMT) Build(xs []Item) error {
	for _, i := range xs {
		if _, e := s.t.Update([]byte(i.URL), i.Digest[:]); e != nil {
			return e
		}
	}
	return nil
}
func (s *SMT) Insert(i Item) error {
	ok, e := s.t.Has([]byte(i.URL))
	if e != nil {
		return e
	}
	if ok {
		return ErrExists
	}
	_, e = s.t.Update([]byte(i.URL), i.Digest[:])
	return e
}
func (s *SMT) Modify(i Item) error {
	ok, e := s.t.Has([]byte(i.URL))
	if e != nil {
		return e
	}
	if !ok {
		return ErrMissing
	}
	_, e = s.t.Update([]byte(i.URL), i.Digest[:])
	return e
}
func (s *SMT) Delete(u string) error {
	ok, e := s.t.Has([]byte(u))
	if e != nil {
		return e
	}
	if !ok {
		return ErrMissing
	}
	_, e = s.t.Delete([]byte(u))
	return e
}
func (s *SMT) Root() (h Hash) { copy(h[:], s.t.Root()); return }
func (s *SMT) Stats() Metrics { return s.stats }
func (s *SMT) Prove(u string) ([]byte, error) {
	p, e := s.GenerateProof(u)
	if e != nil {
		return nil, e
	}
	return p.MarshalBinary()
}
func (s *SMT) GenerateProof(u string) (NativeProof, error) {
	ok, e := s.t.Has([]byte(u))
	if e != nil {
		return nil, e
	}
	if !ok {
		return nil, ErrMissing
	}
	p, e := s.t.ProveCompact([]byte(u))
	if e != nil {
		return nil, e
	}
	return &smtProof{p}, nil
}

func decodeSMT(p []byte) (smt.SparseCompactMerkleProof, error) {
	if len(p) < 2 {
		return smt.SparseCompactMerkleProof{}, errors.New("short SMT proof")
	}
	n := int(binary.BigEndian.Uint16(p))
	k := (n + 7) / 8
	if n > 256 || len(p) < 2+k || (len(p)-2-k)%32 != 0 {
		return smt.SparseCompactMerkleProof{}, errors.New("bad SMT proof")
	}
	q := smt.SparseCompactMerkleProof{NumSideNodes: n, BitMask: p[2 : 2+k]}
	for j := 2 + k; j < len(p); j += 32 {
		q.SideNodes = append(q.SideNodes, p[j:j+32])
	}
	return q, nil
}
func verifySMT(root Hash, i Item, p []byte) bool {
	q, e := decodeSMT(p)
	return e == nil && smt.VerifyCompactProof(q, root[:], []byte(i.URL), i.Digest[:], sha3.NewLegacyKeccak256())
}
func smtRoot(i Item, p []byte) (Hash, error) {
	q, e := decodeSMT(p)
	if e != nil {
		return Hash{}, e
	}
	d, e := smt.DecompactProof(q, sha3.NewLegacyKeccak256())
	if e != nil {
		return Hash{}, e
	}
	key := Sum([]byte(i.URL))
	v := Sum(i.Digest[:])
	r := Sum([]byte{0}, key[:], v[:])
	for j, s := range d.SideNodes {
		bit := len(d.SideNodes) - 1 - j
		if key[bit/8]>>(7-bit%8)&1 == 0 {
			r = Sum([]byte{1}, r[:], s)
		} else {
			r = Sum([]byte{1}, s, r[:])
		}
	}
	return r, nil
}

type MPT struct{ t *trie.Trie }

func newMPT() *MPT { return &MPT{trie.NewEmpty(nil)} }
func (m *MPT) Build(xs []Item) error {
	for _, i := range xs {
		k := Sum([]byte(i.URL))
		if e := m.t.Update(k[:], i.Digest[:]); e != nil {
			return e
		}
	}
	m.t.Hash()
	return nil
}
func (m *MPT) Insert(i Item) error {
	k := Sum([]byte(i.URL))
	v, e := m.t.Get(k[:])
	if e != nil {
		return e
	}
	if v != nil {
		return ErrExists
	}
	return m.t.Update(k[:], i.Digest[:])
}
func (m *MPT) Modify(i Item) error {
	k := Sum([]byte(i.URL))
	v, e := m.t.Get(k[:])
	if e != nil {
		return e
	}
	if v == nil {
		return ErrMissing
	}
	return m.t.Update(k[:], i.Digest[:])
}
func (m *MPT) Delete(u string) error {
	k := Sum([]byte(u))
	v, e := m.t.Get(k[:])
	if e != nil {
		return e
	}
	if v == nil {
		return ErrMissing
	}
	return m.t.Delete(k[:])
}
func (m *MPT) Root() Hash     { return Hash(m.t.Hash()) }
func (m *MPT) Stats() Metrics { return Metrics{} }

type proofDB struct {
	nodes [][]byte
	index map[string][]byte
}

func (p *proofDB) Put(k, v []byte) error {
	p.nodes = append(p.nodes, bytes.Clone(v))
	if p.index == nil {
		p.index = map[string][]byte{}
	}
	p.index[string(k)] = v
	return nil
}
func (p *proofDB) Delete(k []byte) error { delete(p.index, string(k)); return nil }
func (p *proofDB) Get(k []byte) ([]byte, error) {
	if v, ok := p.index[string(k)]; ok {
		return v, nil
	}
	return nil, ErrMissing
}
func (p *proofDB) Has(k []byte) (bool, error) { _, ok := p.index[string(k)]; return ok, nil }
func (m *MPT) Prove(u string) ([]byte, error) {
	p, e := m.GenerateProof(u)
	if e != nil {
		return nil, e
	}
	return p.MarshalBinary()
}
func (m *MPT) GenerateProof(u string) (NativeProof, error) {
	k := Sum([]byte(u))
	v, e := m.t.Get(k[:])
	if e != nil {
		return nil, e
	}
	if v == nil {
		return nil, ErrMissing
	}
	p := &proofDB{}
	if e = m.t.Prove(k[:], p); e != nil {
		return nil, e
	}
	return p, nil
}

func decodeMPT(b []byte) (*proofDB, error) {
	if len(b) < 2 {
		return nil, ErrMissing
	}
	n := int(binary.BigEndian.Uint16(b))
	if n == 0 || n > 65 {
		return nil, ErrMissing
	}
	p := &proofDB{}
	off := 2
	for j := 0; j < n; j++ {
		if off+4 > len(b) {
			return nil, ErrMissing
		}
		l := int(binary.BigEndian.Uint32(b[off:]))
		off += 4
		if l < 1 || off+l > len(b) {
			return nil, ErrMissing
		}
		v := b[off : off+l]
		h := Sum(v)
		p.Put(h[:], v)
		off += l
	}
	if off != len(b) {
		return nil, ErrMissing
	}
	return p, nil
}
func verifyMPT(root Hash, i Item, b []byte) bool {
	p, e := decodeMPT(b)
	if e != nil {
		return false
	}
	k := Sum([]byte(i.URL))
	v, e := trie.VerifyProof(common.Hash(root), k[:], p)
	return e == nil && bytes.Equal(v, i.Digest[:])
}
func subRoot(kind string, i Item, p []byte) (Hash, error) {
	switch kind {
	case "ht":
		return binaryRoot(i, p)
	case "merkle":
		return merkleRoot(i, p)
	case "iavl":
		return iavlRoot(i, p)
	case "smt":
		return smtRoot(i, p)
	case "mpt":
		d, e := decodeMPT(p)
		if e != nil {
			return Hash{}, e
		}
		r := Sum(d.nodes[0])
		if !verifyMPT(r, i, p) {
			return Hash{}, ErrMissing
		}
		return r, nil
	}
	return Hash{}, ErrMissing
}
