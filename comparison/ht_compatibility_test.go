package comparison

import (
	"bytes"
	"data-ownership-system/auth"
	"data-ownership-system/pkg/types"
	"fmt"
	"math/rand"
	"testing"
)

// The prototype is an independent oracle for the retained encoding and deletion
// semantics. Duplicate insertion is rejected by the common comparison API.
func TestHTPrototypeCompatibility(t *testing.T) {
	for seed := int64(1); seed <= 5; seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			names := []string{"c0000", "c0001", "c0002", "c0003", "c0004"}
			ref := auth.NewAuthenticatedDataStructure(.75)
			f, e := NewForest("ht", "flat", 5, 4, .75)
			if e != nil {
				t.Fatal(e)
			}
			xs := sample(60, 5)
			data := make([]*types.DataItem, len(xs))
			live := map[int]Item{}
			for j, i := range xs {
				data[j] = &types.DataItem{URL: i.URL, Hash: bytes.Clone(i.Digest[:])}
				live[j] = i
			}
			if e := ref.BuildFromDataItems(data, names, nil); e != nil {
				t.Fatal(e)
			}
			if e := f.Build(xs); e != nil {
				t.Fatal(e)
			}
			check := func() {
				t.Helper()
				r := f.Root()
				if !bytes.Equal(r[:], ref.GetTopRoot()) {
					t.Fatal("global root mismatch")
				}
				for j, name := range names {
					ht := ref.HashTables[name]
					d := f.Parts[j].(*Dense)
					if ht.Size() != len(d.slots) || ht.Count() != d.count {
						t.Fatal("capacity/count mismatch")
					}
					for k, x := range ht.GetAllEntries() {
						y := d.slots[k]
						if x.URL != y.url || x.State != int(y.state) || !bytes.Equal(x.Hash, y.digest[:]) {
							t.Fatal("slot mismatch", name, k)
						}
					}
					r := d.Root()
					if !bytes.Equal(r[:], ref.SubRoots[name]) {
						t.Fatal("subroot mismatch")
					}
				}
			}
			check()
			r := rand.New(rand.NewSource(seed))
			next := 60
			for step := 0; step < 500; step++ {
				op := "insert"
				id := r.Intn(next)
				i, exists := live[id]
				if exists {
					if r.Intn(3) == 0 {
						op = "delete"
						delete(live, id)
						e = ref.DeleteDataItem(i.URL)
					} else {
						op = "modify"
						i.Digest = Sum([]byte(fmt.Sprint(seed, step)))
						live[id] = i
						e = ref.UpdateDataItem(i.URL, bytes.Clone(i.Digest[:]))
					}
				} else {
					i = Item{URL: fmt.Sprintf("http://example.test/c%04d/item%08d", next%5, next), Digest: Sum([]byte(fmt.Sprint(next))), Category: next % 5}
					live[next] = i
					next++
					e = ref.InsertDataItem(&types.DataItem{URL: i.URL, Hash: bytes.Clone(i.Digest[:])})
				}
				if e != nil {
					t.Fatal(e)
				}
				if e = f.Apply(op, i); e != nil {
					t.Fatal(e)
				}
				check()
				if op != "delete" {
					p, e := f.Prove(i)
					if e != nil || !Verify("ht", f.Root(), i, p) {
						t.Fatal("membership mismatch")
					}
					q, e := ref.GenerateProof(i.URL, i.Digest[:])
					if e != nil || !ref.VerifyProof(q) {
						t.Fatal("prototype rejected member")
					}
				}
			}
		})
	}
}

func TestHTCapacityAndHashEncoding(t *testing.T) {
	for _, n := range []int{0, 1, 11, 12, 13, 24, 25, 48, 49, 96, 97} {
		f, _ := NewForest("ht", "flat", 1, 1, .75)
		xs := sample(n, 1)
		if e := f.Build(xs); e != nil {
			t.Fatal(e)
		}
		ref := auth.NewAuthenticatedDataStructure(.75)
		data := []*types.DataItem{}
		for _, i := range xs {
			data = append(data, &types.DataItem{URL: i.URL, Hash: bytes.Clone(i.Digest[:])})
		}
		if e := ref.BuildFromDataItems(data, []string{"c0000"}, nil); e != nil {
			t.Fatal(e)
		}
		r := f.Root()
		if !bytes.Equal(r[:], ref.GetTopRoot()) {
			t.Fatal("boundary root", n)
		}
	}
}
