package comparison

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
)

func sample(n, m int) []Item {
	x := make([]Item, n)
	for j := range x {
		x[j] = Item{URL: fmt.Sprintf("http://example.test/c%04d/item%08d", j%m, j), Digest: Sum([]byte(fmt.Sprint(j))), Category: j % m}
	}
	return x
}
func TestAllAdaptersAndLayouts(t *testing.T) {
	for _, k := range Kinds {
		for _, l := range []string{"global", "flat", "tree"} {
			t.Run(k+"/"+l, func(t *testing.T) {
				xs := sample(137, 5)
				f, e := NewForest(k, l, 5, 3, .75)
				if e != nil {
					t.Fatal(e)
				}
				if e = f.Build(xs); e != nil {
					t.Fatal(e)
				}
				check := func(i Item) {
					p, e := f.Prove(i)
					if e != nil {
						t.Fatal(e)
					}
					if !Verify(k, f.Root(), i, p) {
						t.Fatalf("valid proof rejected: %s", i.URL)
					}
					bad := i
					bad.Digest[0] ^= 1
					if Verify(k, f.Root(), bad, p) {
						t.Fatal("changed value accepted")
					}
					bad = i
					bad.URL += "x"
					if Verify(k, f.Root(), bad, p) {
						t.Fatal("changed URL accepted")
					}
					for j := 0; j < len(p); j += max(1, len(p)/9) {
						q := bytes.Clone(p)
						q[j] ^= 1
						if Verify(k, f.Root(), i, q) {
							t.Fatalf("tampered proof accepted byte %d", j)
						}
					}
				}
				for _, i := range xs {
					check(i)
				}
				oldRoot := f.Root()
				oldItem := xs[42]
				p, _ := f.Prove(xs[42])
				xs[42].Digest = Sum([]byte("modified"))
				if e = f.Apply("modify", xs[42]); e != nil {
					t.Fatal(e)
				}
				if f.Root() == oldRoot {
					t.Fatal("root unchanged")
				}
				if Verify(k, f.Root(), oldItem, p) {
					t.Fatal("stale proof accepted")
				}
				check(xs[42])
				for j := 0; j < 60; j++ {
					if e = f.Apply("delete", xs[j]); e != nil {
						t.Fatal(e)
					}
					if _, e = f.Prove(xs[j]); e == nil {
						t.Fatal("deleted key has proof")
					}
				}
				for j := 0; j < 60; j++ {
					if e = f.Apply("insert", xs[j]); e != nil {
						t.Fatal(e)
					}
					check(xs[j])
				}
				for j := 137; j < 650; j++ {
					i := Item{fmt.Sprintf("http://new/c%d/%d", j%5, j), Sum([]byte(fmt.Sprint(j))), j % 5}
					if e = f.Apply("insert", i); e != nil {
						t.Fatal(e)
					}
					check(i)
				}
				if e = f.Apply("insert", xs[0]); e != ErrExists {
					t.Fatalf("duplicate: %v", e)
				}
				if e = f.Apply("delete", Item{URL: "absent"}); e != ErrMissing {
					t.Fatal(e)
				}
			})
		}
	}
}
func TestRandomizedStateMachine(t *testing.T) {
	for _, k := range Kinds {
		t.Run(k, func(t *testing.T) {
			f, _ := NewForest(k, "flat", 3, 1, .75)
			if e := f.Build(nil); e != nil {
				t.Fatal(e)
			}
			state := map[int]Item{}
			r := rand.New(rand.NewSource(8))
			for j := 0; j < 2500; j++ {
				id := r.Intn(180)
				i := Item{fmt.Sprintf("http://x/%d", id), Sum([]byte(fmt.Sprint(j))), id % 3}
				old, ok := state[id]
				op := r.Intn(3)
				var e error
				if !ok {
					e = f.Apply("insert", i)
					state[id] = i
				} else if op == 0 {
					e = f.Apply("delete", old)
					delete(state, id)
				} else {
					e = f.Apply("modify", i)
					state[id] = i
				}
				if e != nil {
					t.Fatal(e)
				}
				if j%47 == 0 {
					for _, v := range state {
						p, e := f.Prove(v)
						if e != nil || !Verify(k, f.Root(), v, p) {
							t.Fatalf("step %d key %s: %v", j, v.URL, e)
						}
					}
				}
			}
		})
	}
}
func TestCategoryDynamics(t *testing.T) {
	for _, k := range Kinds {
		for _, layout := range []string{"global", "flat", "tree"} {
			f, _ := NewForest(k, layout, 2, 2, .75)
			xs := sample(60, 2)
			if e := f.Build(xs); e != nil {
				t.Fatal(e)
			}
			extra := []Item{{"http://new/category/item", Sum([]byte("v")), 2}}
			if e := f.AddCategory(extra); e != nil {
				t.Fatal(e)
			}
			p, e := f.Prove(extra[0])
			if e != nil || !Verify(k, f.Root(), extra[0], p) {
				t.Fatalf("category insert %s %s %v", k, layout, e)
			}
			beforeDrop := f.Stats()
			if e = f.DropLastCategory(extra); e != nil {
				t.Fatal(e)
			}
			dropWork := f.Stats().Sub(beforeDrop)
			if layout == "flat" && (dropWork.HashCalls != 1 || dropWork.HashBytes != 64 || dropWork.Probes != 0 || dropWork.Rotations != 0 || dropWork.Rebuilds != 0 || dropWork.RehashedItems != 0) {
				t.Fatalf("category retirement lost cumulative counters for %s: %+v", k, dropWork)
			}
			if layout != "global" && f.Parts[:cap(f.Parts)][len(f.Parts)] != nil {
				t.Fatal("retired category remains reachable through backing storage")
			}
			if _, e = f.Prove(extra[0]); e == nil {
				t.Fatal("dropped category still accessible")
			}
		}
	}
}
func FuzzProofParsing(f *testing.F) {
	f.Add([]byte{0, 0})
	f.Fuzz(func(t *testing.T, p []byte) {
		for _, k := range Kinds {
			Verify(k, Hash{}, Item{URL: "x"}, p)
		}
	})
}
