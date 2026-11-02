package comparison

import (
	"fmt"
	"golang.org/x/crypto/sha3"
	"hash"
)

const htEmptyURL = "__EMPTY_SLOT__"
const htDeletedURL = "__DELETED_SLOT__"

var htEmptyHash = Sum([]byte(htEmptyURL), []byte{0}, make([]byte, 32))

type cell struct {
	url    string
	state  byte
	digest Hash
}
type Dense struct {
	load    float64
	stats   Metrics
	slots   []cell
	tree    []Hash
	count   int
	h       hash.Hash
	scratch []byte
	empty   Hash
}

func newDense(kind string, load float64) *Dense {
	o := &Dense{load: load, h: sha3.NewLegacyKeccak256(), scratch: make([]byte, 0, 256)}
	o.empty = htEmptyHash
	return o
}
func (o *Dense) digest(b []byte) (h Hash) {
	o.h.Reset()
	o.h.Write(b)
	o.stats.HashCalls++
	o.stats.HashBytes += uint64(len(b))
	o.h.Sum(h[:0])
	return
}
func (o *Dense) leaf(x cell) Hash {
	o.scratch = append(o.scratch[:0], x.url...)
	o.scratch = append(o.scratch, x.state)
	o.scratch = append(o.scratch, x.digest[:]...)
	return o.digest(o.scratch)
}
func (o *Dense) pair(a, b Hash) Hash {
	o.scratch = append(o.scratch[:0], a[:]...)
	o.scratch = append(o.scratch, b[:]...)
	return o.digest(o.scratch)
}
func hashIndices(s string, n int) (int, int) {
	var a uint32
	var b uint32 = 2166136261
	for j := range len(s) {
		a = a*31 + uint32(s[j])
		b = (b ^ uint32(s[j])) * 31
	}
	return int(a) & (n - 1), (int(b) & (n - 1)) | 1
}
func (o *Dense) alloc(n int) {
	o.slots = make([]cell, n)
	for j := range o.slots {
		o.slots[j].url = htEmptyURL
	}
	o.tree = make([]Hash, 2*n)
}
func (o *Dense) find(s string, insert bool) (int, bool) {
	a, b := hashIndices(s, len(o.slots))
	for k := range len(o.slots) {
		o.stats.Probes++
		j := (a + k*b) & (len(o.slots) - 1)
		x := o.slots[j]
		if insert {
			if x.url == htEmptyURL || x.state == 1 || x.url == s {
				return j, true
			}
		} else {
			if x.url == s || x.url == htEmptyURL {
				return j, x.url == s && x.url != htEmptyURL && x.state != 1
			}
		}
	}
	return -1, false
}
func (o *Dense) place(i Item) (int, error) {
	if i.URL == "" {
		return -1, fmt.Errorf("empty URL")
	}
	j, ok := o.find(i.URL, true)
	if !ok {
		return -1, fmt.Errorf("full")
	}
	if o.slots[j].url == htEmptyURL || o.slots[j].state == 1 {
		o.count++
	}
	o.slots[j] = cell{url: i.URL, digest: i.Digest}
	return j, nil
}
func (o *Dense) rebuild() {
	n := len(o.slots)
	for j, x := range o.slots {
		if x.url == htEmptyURL && x.state == 0 && x.digest == (Hash{}) {
			o.tree[n+j] = o.empty
		} else {
			o.tree[n+j] = o.leaf(x)
		}
	}
	for j := n - 1; j > 0; j-- {
		o.tree[j] = o.pair(o.tree[2*j], o.tree[2*j+1])
	}
}
func (o *Dense) update(j int) {
	n := len(o.slots)
	o.tree[n+j] = o.leaf(o.slots[j])
	for j = (n + j) / 2; j > 0; j /= 2 {
		o.tree[j] = o.pair(o.tree[2*j], o.tree[2*j+1])
	}
}
func (o *Dense) Build(xs []Item) error {
	o.count = 0
	o.alloc(max(16, power2(int(float64(len(xs))/o.load))))
	for _, i := range xs {
		if float64(o.count)/float64(len(o.slots)) >= o.load {
			o.grow()
		}
		if _, e := o.place(i); e != nil {
			return e
		}
	}
	o.rebuild()
	return nil
}
func (o *Dense) grow() {
	old := o.slots
	o.stats.Rebuilds++
	o.stats.RehashedItems += uint64(o.count)
	o.alloc(2 * len(old))
	o.count = 0
	for _, x := range old {
		if x.url != htEmptyURL && x.state != 1 {
			if _, e := o.place(Item{URL: x.url, Digest: x.digest}); e != nil {
				panic(e)
			}
		}
	}
}
func (o *Dense) Insert(i Item) error {
	if len(o.slots) == 0 {
		o.alloc(16)
		o.rebuild()
	}
	if _, ok := o.find(i.URL, false); ok {
		return ErrExists
	}
	if i.URL == "" {
		return fmt.Errorf("empty URL")
	}
	grown := float64(o.count)/float64(len(o.slots)) >= o.load
	if grown {
		o.grow()
	}
	j, e := o.place(i)
	if e != nil {
		return e
	}
	if grown {
		o.rebuild()
	} else {
		o.update(j)
	}
	return nil
}
func (o *Dense) Modify(i Item) error {
	j, ok := o.find(i.URL, false)
	if !ok {
		return ErrMissing
	}
	o.slots[j].digest = i.Digest
	o.update(j)
	return nil
}
func (o *Dense) Delete(s string) error {
	j, ok := o.find(s, false)
	if !ok {
		return ErrMissing
	}
	o.slots[j].url = htDeletedURL
	o.slots[j].state = 1
	o.count--
	o.update(j)
	return nil
}
func (o *Dense) Root() Hash {
	if len(o.tree) == 0 {
		return htEmptyHash
	}
	return o.tree[1]
}
func (o *Dense) Prove(s string) ([]byte, error) {
	p, e := o.GenerateProof(s)
	if e != nil {
		return nil, e
	}
	return p.MarshalBinary()
}
func (o *Dense) GenerateProof(s string) (NativeProof, error) {
	j, ok := o.find(s, false)
	if !ok {
		return nil, ErrMissing
	}
	n := len(o.slots)
	depth := 0
	for k := n; k > 1; k /= 2 {
		depth++
	}
	p := &hashPathProof{Siblings: make([]Hash, depth), Directions: make([]byte, depth)}
	for j, k := n+j, 0; j > 1; j, k = j/2, k+1 {
		p.Directions[k] = byte(j & 1)
		p.Siblings[k] = o.tree[j^1]
	}
	return p, nil
}

func (o *Dense) Stats() Metrics { return o.stats }
