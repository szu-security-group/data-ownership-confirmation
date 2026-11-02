package comparison

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var CategoryDeadline time.Time
var CategoryCompleted int
var ErrBudget = errors.New("category operation time budget exceeded")

// Forest changes layout without imposing the HT indexing on any alternative.
type Forest struct {
	Kind, Layout string
	Load         float64
	Workers      int
	categoryIDs  map[string]int
	Parts        []ADS
	Roots        []Hash
	root         Hash
	top          []Hash
	stats        Metrics
	h            *hasher
}

func NewForest(kind, layout string, m, workers int, load float64) (*Forest, error) {
	if layout == "global" {
		m = 1
	}
	if m < 1 || workers < 1 {
		return nil, errors.New("invalid configuration")
	}
	if layout != "global" && layout != "flat" && layout != "tree" {
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
	if f.Layout == "global" {
		return 0
	}
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
	switch f.Layout {
	case "global":
		f.root = f.Roots[0]
	case "flat":
		b := make([]byte, 0, 32*len(f.Roots))
		for _, r := range f.Roots {
			b = append(b, r[:]...)
		}
		f.root = f.h.digest(b)
	case "tree":
		n := power2(len(f.Roots))
		if len(f.top) != 2*n || changed < 0 {
			f.top = make([]Hash, 2*n)
			for j := 0; j < n; j++ {
				f.top[n+j] = Empty
				if j < len(f.Roots) {
					f.top[n+j] = f.h.digest([]byte{5}, f.Roots[j][:])
				}
			}
			for j := n - 1; j > 0; j-- {
				f.top[j] = f.h.parent(f.top[j*2], f.top[j*2+1])
			}
		} else {
			j := n + changed
			f.top[j] = f.h.digest([]byte{5}, f.Roots[changed][:])
			for j /= 2; j > 0; j /= 2 {
				f.top[j] = f.h.parent(f.top[2*j], f.top[2*j+1])
			}
		}
		f.root = f.top[1]
	}
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
	switch f.Layout {
	case "flat":
		q.Mode = 1
		q.Aux = make([]Hash, 0, len(f.Roots)-1)
		for j, r := range f.Roots {
			if j != c {
				q.Aux = append(q.Aux, r)
			}
		}
	case "tree":
		q.Mode = 2
		for j := power2(len(f.Parts)) + c; j > 1; j /= 2 {
			q.Aux = append(q.Aux, f.top[j^1])
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
	switch mode {
	case 0:
		if m != 1 || c != 0 || n != 0 {
			return false
		}
	case 1:
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
	case 2:
		height := 0
		for k := power2(m); k > 1; k /= 2 {
			height++
		}
		if n != height {
			return false
		}
		r = Sum([]byte{5}, r[:])
		for j := 0; j < n; j++ {
			s := b[11+j*32 : 43+j*32]
			if c&1 == 0 {
				r = Sum([]byte{1}, r[:], s)
			} else {
				r = Sum([]byte{1}, s, r[:])
			}
			c /= 2
		}
	default:
		return false
	}
	return r == root
}
func (f *Forest) AddCategory(xs []Item) error {
	CategoryCompleted = 0
	if f.Layout == "global" {
		for _, i := range xs {
			if !CategoryDeadline.IsZero() && time.Now().After(CategoryDeadline) {
				return ErrBudget
			}
			if e := f.Parts[0].Insert(i); e != nil {
				return e
			}
			CategoryCompleted++
		}
		f.Roots[0] = f.Parts[0].Root()
		f.aggregate(-1)
		return nil
	}
	a, e := New(f.Kind, f.Load)
	if e != nil {
		return e
	}
	if e = a.Build(xs); e != nil {
		return e
	}
	f.Parts = append(f.Parts, a)
	f.Roots = append(f.Roots, a.Root())
	f.aggregate(-1)
	CategoryCompleted = len(xs)
	return nil
}
func (f *Forest) DropLastCategory(xs []Item) error {
	CategoryCompleted = 0
	if f.Layout == "global" {
		for _, i := range xs {
			if !CategoryDeadline.IsZero() && time.Now().After(CategoryDeadline) {
				return ErrBudget
			}
			if e := f.Parts[0].Delete(i.URL); e != nil {
				return e
			}
			CategoryCompleted++
		}
		f.Roots[0] = f.Parts[0].Root()
		f.aggregate(-1)
		return nil
	}
	if len(f.Parts) < 2 {
		return fmt.Errorf("cannot drop final category")
	}
	// Keep lifetime work counters monotonic after a subtree is retired.
	f.stats = f.stats.Add(f.Parts[len(f.Parts)-1].Stats())
	// Do not retain the removed tree through the slice's backing array.
	f.Parts[len(f.Parts)-1] = nil
	f.Parts = f.Parts[:len(f.Parts)-1]
	f.Roots = f.Roots[:len(f.Roots)-1]
	f.aggregate(-1)
	CategoryCompleted = len(xs)
	return nil
}
