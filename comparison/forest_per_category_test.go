package comparison

import (
	"bytes"
	"data-ownership-system/pkg/types"
	"fmt"
	"ownership-comparison/internal/reference/auth"
	"testing"
)

func TestPerCategoryMatchesReference(t *testing.T) {
	for _, m := range []int{1, 10, 100, 1000} {
		t.Run(fmt.Sprint(m), func(t *testing.T) {
			xs := sample(2053, m)
			names := make([]string, m)
			for j := range names {
				names[j] = fmt.Sprintf("c%04d", j)
			}
			one, _ := NewForest("ht", "flat", m, m, .75)
			pool, _ := NewForest("ht", "flat", m, 4, .75)
			for _, f := range []*Forest{one, pool} {
				if e := f.ConfigureCategories(names); e != nil {
					t.Fatal(e)
				}
			}
			if e := one.BuildPerCategory(xs); e != nil {
				t.Fatal(e)
			}
			if e := pool.Build(xs); e != nil {
				t.Fatal(e)
			}
			if one.Root() != pool.Root() {
				t.Fatal("scheduler changed root")
			}
			ref := auth.NewAuthenticatedDataStructure(.75)
			old := make([]*types.DataItem, len(xs))
			for j, i := range xs {
				old[j] = &types.DataItem{URL: i.URL, Hash: bytes.Clone(i.Digest[:])}
			}
			if e := ref.BuildFromDataItems(old, names, nil); e != nil {
				t.Fatal(e)
			}
			root := one.Root()
			if !bytes.Equal(root[:], ref.GetTopRoot()) {
				t.Fatal("reference root mismatch")
			}
			for _, j := range []int{0, len(xs) / 2, len(xs) - 1} {
				p, e := one.Prove(xs[j])
				if e != nil || !Verify("ht", one.Root(), xs[j], p) {
					t.Fatal("proof rejected")
				}
			}
			i := xs[len(xs)/2]
			i.Digest = Sum([]byte("change"))
			for _, op := range []string{"modify", "delete"} {
				if e := one.Apply(op, i); e != nil {
					t.Fatal(e)
				}
				if e := pool.Apply(op, i); e != nil {
					t.Fatal(e)
				}
				if one.Root() != pool.Root() {
					t.Fatal("update differs by scheduler")
				}
			}
		})
	}
}
