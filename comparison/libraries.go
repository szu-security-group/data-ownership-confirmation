package comparison

import (
	"bytes"
	"encoding/binary"
	mt "github.com/cbergoon/merkletree"
	"github.com/cosmos/iavl"
	db "github.com/cosmos/iavl/db"
	ics23 "github.com/cosmos/ics23/go"
	"golang.org/x/crypto/sha3"
)

type IAVL struct{ t *iavl.MutableTree }

func newIAVL() *IAVL { return &IAVL{iavl.NewMutableTree(db.NewMemDB(), 0, false, iavl.NewNopLogger())} }
func (a *IAVL) Build(xs []Item) error {
	for _, x := range xs {
		updated, e := a.t.Set([]byte(x.URL), bytes.Clone(x.Digest[:]))
		if e != nil {
			return e
		}
		if updated {
			return ErrExists
		}
	}
	a.t.WorkingHash()
	return nil
}
func (a *IAVL) Insert(x Item) error {
	v, e := a.t.Get([]byte(x.URL))
	if e != nil {
		return e
	}
	if v != nil {
		return ErrExists
	}
	_, e = a.t.Set([]byte(x.URL), bytes.Clone(x.Digest[:]))
	return e
}
func (a *IAVL) Modify(x Item) error {
	v, e := a.t.Get([]byte(x.URL))
	if e != nil {
		return e
	}
	if v == nil {
		return ErrMissing
	}
	_, e = a.t.Set([]byte(x.URL), bytes.Clone(x.Digest[:]))
	return e
}
func (a *IAVL) Delete(u string) error {
	_, ok, e := a.t.Remove([]byte(u))
	if e != nil {
		return e
	}
	if !ok {
		return ErrMissing
	}
	return nil
}
func (a *IAVL) Root() (h Hash) { copy(h[:], a.t.WorkingHash()); return }
func (a *IAVL) Stats() Metrics { return Metrics{} }
func (a *IAVL) Prove(u string) ([]byte, error) {
	p, e := a.GenerateProof(u)
	if e != nil {
		return nil, e
	}
	return p.MarshalBinary()
}
func (a *IAVL) GenerateProof(u string) (NativeProof, error) {
	v, e := a.t.Get([]byte(u))
	if e != nil {
		return nil, e
	}
	if v == nil {
		return nil, ErrMissing
	}
	p, e := a.t.GetMembershipProof([]byte(u))
	if e != nil {
		return nil, e
	}
	return &iavlProof{p}, nil
}
func iavlRoot(i Item, p []byte) (h Hash, e error) {
	var q ics23.CommitmentProof
	if e = q.Unmarshal(p); e != nil {
		return
	}
	x := q.GetExist()
	if x == nil || !bytes.Equal(x.Key, []byte(i.URL)) || !bytes.Equal(x.Value, i.Digest[:]) {
		return h, ErrMissing
	}
	if e = x.CheckAgainstSpec(ics23.IavlSpec); e != nil {
		return
	}
	root, e := x.Calculate()
	if e != nil {
		return h, e
	}
	if len(root) != 32 {
		return h, ErrMissing
	}
	copy(h[:], root)
	return h, nil
}

type merkleContent struct{ item Item }

func (c merkleContent) CalculateHash() ([]byte, error) {
	h := sha3.NewLegacyKeccak256()
	var n [4]byte
	binary.BigEndian.PutUint32(n[:], uint32(len(c.item.URL)))
	h.Write([]byte{0})
	h.Write(n[:])
	h.Write([]byte(c.item.URL))
	h.Write(c.item.Digest[:])
	return h.Sum(nil), nil
}
func (c merkleContent) Equals(other mt.Content) (bool, error) {
	v, ok := other.(merkleContent)
	return ok && c.item.URL == v.item.URL && c.item.Digest == v.item.Digest, nil
}

type Merkle struct {
	t  *mt.MerkleTree
	xs []mt.Content
}

func newMerkle() *Merkle { return &Merkle{} }
func (a *Merkle) Build(xs []Item) error {
	a.xs = make([]mt.Content, len(xs))
	for j, x := range xs {
		a.xs[j] = merkleContent{x}
	}
	return a.rebuild()
}
func (a *Merkle) rebuild() error {
	if len(a.xs) == 0 {
		a.t = nil
		return nil
	}
	if a.t == nil {
		var e error
		a.t, e = mt.NewTreeWithOptions(a.xs, mt.WithHasher(sha3.NewLegacyKeccak256), mt.WithLeafIndex())
		return e
	}
	return a.t.RebuildTreeWith(a.xs)
}
func (a *Merkle) find(u string) int {
	for j, c := range a.xs {
		if c.(merkleContent).item.URL == u {
			return j
		}
	}
	return -1
}
func (a *Merkle) Insert(x Item) error {
	if a.find(x.URL) >= 0 {
		return ErrExists
	}
	a.xs = append(a.xs, merkleContent{x})
	return a.rebuild()
}
func (a *Merkle) Modify(x Item) error {
	j := a.find(x.URL)
	if j < 0 {
		return ErrMissing
	}
	a.xs[j] = merkleContent{x}
	return a.rebuild()
}
func (a *Merkle) Delete(u string) error {
	j := a.find(u)
	if j < 0 {
		return ErrMissing
	}
	copy(a.xs[j:], a.xs[j+1:])
	a.xs[len(a.xs)-1] = nil
	a.xs = a.xs[:len(a.xs)-1]
	return a.rebuild()
}
func (a *Merkle) Root() (h Hash) {
	if a.t == nil {
		return Empty
	}
	copy(h[:], a.t.MerkleRoot())
	return
}
func (a *Merkle) Stats() Metrics { return Metrics{} }
func (a *Merkle) Prove(u string) ([]byte, error) {
	j := a.find(u)
	if j < 0 {
		return nil, ErrMissing
	}
	return a.ProveItem(a.xs[j].(merkleContent).item)
}
func (a *Merkle) ProveItem(i Item) ([]byte, error) {
	p, e := a.GenerateProofItem(i)
	if e != nil {
		return nil, e
	}
	return p.MarshalBinary()
}
func (a *Merkle) GenerateProof(u string) (NativeProof, error) {
	j := a.find(u)
	if j < 0 {
		return nil, ErrMissing
	}
	return a.GenerateProofItem(a.xs[j].(merkleContent).item)
}
func (a *Merkle) GenerateProofItem(i Item) (NativeProof, error) {
	if a.t == nil {
		return nil, ErrMissing
	}
	path, ds, e := a.t.GetMerklePath(merkleContent{i})
	if e != nil {
		return nil, ErrMissing
	}
	return &merklePathProof{path, ds}, nil
}

func merkleRoot(i Item, p []byte) (h Hash, e error) {
	if len(p) < 2 {
		return h, ErrMissing
	}
	n := int(binary.BigEndian.Uint16(p))
	if n > 256 || len(p) != 2+33*n {
		return h, ErrMissing
	}
	raw, _ := (merkleContent{i}).CalculateHash()
	copy(h[:], raw)
	for j := 0; j < n; j++ {
		d := p[2+33*j]
		s := p[3+33*j : 35+33*j]
		if d == 0 {
			h = Sum(h[:], s)
		} else if d == 1 {
			h = Sum(s, h[:])
		} else {
			return h, ErrMissing
		}
	}
	return h, nil
}
