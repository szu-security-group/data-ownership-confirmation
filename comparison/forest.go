package comparison

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Forest maintains one subtree per category and hashes their ordered roots.
type Forest struct {
	Kind, Layout string
	Load         float64
	Workers      int
	categoryIDs  map[string]int
	Parts        []ADS
	Roots        []Hash
	root         Hash
	stats        Metrics
	h            *hasher
}

func NewForest(kind, layout string, m, workers int, load float64) (*Forest, error) {
	if m < 1 || workers < 1 {
		return nil, errors.New("invalid configuration")
	}
	if layout != "flat" {
		return nil, errors.New("invalid layout")
	}
	f := &Forest{Kind: kind, Layout: layout, Load: load, Workers: workers, Parts: make([]ADS, m), Roots: make([]Hash, m)}
	f.h = newHasher(&f.stats)
	for j := range f.Parts {
		a, e := New(kind, load)
		if e != nil {
			return nil, e
		}
		f.Parts[j] = a
	}
	return f, nil
}
func (f *Forest) cat(i Item) int {
	if f.categoryIDs != nil {
		name := URLCategory(i.URL)
		id, ok := f.categoryIDs[name]
		if !ok {
			return -1
		}
		return id
	}
	return i.Category
}
func (f *Forest) Build(xs []Item) error {
	groups := make([][]Item, len(f.Parts))
	for _, i := range xs {
		c := f.cat(i)
		if c < 0 || c >= len(groups) {
			return errors.New("invalid category")
		}
		groups[c] = append(groups[c], i)
	}
	jobs := make(chan int, len(groups))
	errs := make(chan error, len(groups))
	for j := range groups {
		jobs <- j
	}
	close(jobs)
	var wg sync.WaitGroup
	for w := 0; w < min(f.Workers, len(groups)); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if e := f.Parts[j].Build(groups[j]); e != nil {
					errs <- e
				}
				f.Roots[j] = f.Parts[j].Root()
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		return e
	}
	f.aggregate(-1)
	return nil
}
func (f *Forest) aggregate(changed int) {
	b := make([]byte, 0, 32*len(f.Roots))
	for _, r := range f.Roots {
		b = append(b, r[:]...)
	}
	f.root = f.h.digest(b)
}

func (f *Forest) Apply(op string, i Item) error {
	c := f.cat(i)
	if c < 0 || c >= len(f.Parts) {
		return ErrMissing
	}
	var e error
	switch op {
	case "insert":
		e = f.Parts[c].Insert(i)
	case "modify":
		e = f.Parts[c].Modify(i)
	case "delete":
		e = f.Parts[c].Delete(i.URL)
	default:
		return errors.New("unknown operation")
	}
	if e != nil {
		return e
	}
	f.Roots[c] = f.Parts[c].Root()
	f.aggregate(c)
	return nil
}
func (f *Forest) Root() Hash { return f.root }
func (f *Forest) Stats() Metrics {
	s := f.stats
	for _, p := range f.Parts {
		s = s.Add(p.Stats())
	}
	return s
}

// ConfigureCategories enables the same URL-derived category routing for every component.
// Call during the timed initialization; supplied names define the common root order.
func (f *Forest) ConfigureCategories(names []string) error {
	if len(names) != len(f.Parts) {
		return fmt.Errorf("category count mismatch")
	}
	f.categoryIDs = make(map[string]int, len(names))
	for j, name := range names {
		if _, exists := f.categoryIDs[name]; exists {
			return fmt.Errorf("duplicate category")
		}
		f.categoryIDs[name] = j
	}
	return nil
}
func URLCategory(u string) string {
	parts := strings.Split(u, "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return "default"
}

// Prove encodes a generated proof for validation/export. Benchmarks time GenerateProof only.
func (f *Forest) Prove(i Item) ([]byte, error) {
	p, e := f.GenerateProof(i)
	if e != nil {
		return nil, e
	}
	return p.MarshalBinary()
}
func (f *Forest) GenerateProof(i Item) (*ForestProof, error) {
	c := f.cat(i)
	if c < 0 || c >= len(f.Parts) {
		return nil, ErrMissing
	}
	var p NativeProof
	var e error
	if a, ok := f.Parts[c].(interface {
		GenerateProofItem(Item) (NativeProof, error)
	}); ok {
		p, e = a.GenerateProofItem(i)
	} else {
		p, e = f.Parts[c].GenerateProof(i.URL)
	}
	if e != nil {
		return nil, e
	}
	q := &ForestProof{Category: c, CategoryCount: len(f.Parts), Sub: p}
	q.Mode = 1
	q.Aux = make([]Hash, 0, len(f.Roots)-1)
	for j, r := range f.Roots {
		if j != c {
			q.Aux = append(q.Aux, r)
		}
	}

	return q, nil
}

func Verify(kind string, root Hash, i Item, b []byte) bool {
	if len(b) < 11 {
		return false
	}
	mode := b[0]
	c := int(binary.BigEndian.Uint32(b[1:]))
	m := int(binary.BigEndian.Uint32(b[5:]))
	n := int(binary.BigEndian.Uint16(b[9:]))
	off := 11 + n*32
	if m < 1 || m > 100000 || c >= m || off > len(b) {
		return false
	}
	r, e := subRoot(kind, i, b[off:])
	if e != nil {
		return false
	}
	if mode != 1 {
		return false
	}
	if n != m-1 {
		return false
	}
	hs := make([]byte, 0, m*32)
	j := 11
	for k := 0; k < m; k++ {
		if k == c {
			hs = append(hs, r[:]...)
		} else {
			hs = append(hs, b[j:j+32]...)
			j += 32
		}
	}
	r = Sum(hs)

	return r == root
}
