package main

import (
	"bytes"
	"encoding/json"
	"math/rand"
	c "ownership-comparison"
	"testing"
)

func TestFixedSizeInsertionAndProofTiming(t *testing.T) {
	oldKind, oldLayout, oldN, oldM, oldOps, oldBudget, oldNames, oldEncoder := *kind, *layout, *n, *m, *ops, *phaseBudget, categoryNames, enc
	defer func() {
		*kind = oldKind
		*layout = oldLayout
		*n = oldN
		*m = oldM
		*ops = oldOps
		*phaseBudget = oldBudget
		categoryNames = oldNames
		enc = oldEncoder
	}()
	*layout = "flat"
	*n = 36
	*m = 3
	*ops = 20
	*phaseBudget = 100
	categoryNames = []string{"train2017", "val2017", "annotations"}
	for _, k := range c.Kinds {
		t.Run(k, func(t *testing.T) {
			*kind = k
			xs := make([]c.Item, *n)
			for j := range xs {
				xs[j] = makeItem(j, j%*m)
			}
			f, e := c.NewForest(k, "flat", *m, 2, .75)
			if e != nil {
				t.Fatal(e)
			}
			if e = f.ConfigureCategories(categoryNames); e != nil {
				t.Fatal(e)
			}
			if e = f.Build(xs); e != nil {
				t.Fatal(e)
			}
			var out bytes.Buffer
			enc = json.NewEncoder(&out)
			steady(f, xs)
			decoder := json.NewDecoder(&out)
			for _, op := range []string{"prove", "insert", "modify", "delete"} {
				var row Result
				if e = decoder.Decode(&row); e != nil {
					t.Fatal(e)
				}
				if op == "insert" {
					extra := row.Extra.(map[string]any)
					if extra["restore_between_operations"] != true {
						t.Fatal("insertion cleanup missing")
					}
					if k == "ht" && extra["insertion_preparation_rebuilds"] != float64(3) {
						t.Fatal("capacity boundary was not prepared")
					}
					if k == "ht" && row.Stats.Rebuilds != 0 {
						t.Fatal("timed expansion")
					}
				}
				if row.Operation != op || row.Count != *ops || row.Status != "complete" {
					t.Fatalf("bad row %+v", row)
				}
			}
			r := rand.New(rand.NewSource(*seed + 123))
			for j := 0; j < *ops; j++ {
				r.Intn(len(xs))
			}
			p := plans(xs, *ops, r)
			model := map[string]c.Item{}
			for _, i := range xs {
				model[i.URL] = i
			}
			for _, i := range p["insert"] {
				if _, e = f.Prove(i); e == nil {
					t.Fatal("inserted key was not cleaned up")
				}
			}
			for _, i := range p["modify"] {
				model[i.URL] = i
			}
			for _, i := range p["delete"] {
				delete(model, i.URL)
				if _, e = f.Prove(i); e == nil {
					t.Fatal("delete was restored")
				}
			}
			for _, i := range model {
				b, e := f.Prove(i)
				if e != nil || !c.Verify(k, f.Root(), i, b) {
					t.Fatal("consecutive state mismatch", i.URL, e)
				}
			}
		})
	}
}
func TestPlansHaveUniqueDeletionTargets(t *testing.T) {
	savedM, savedNames := *m, categoryNames
	defer func() { *m = savedM; categoryNames = savedNames }()
	*m = 3
	categoryNames = nil
	xs := make([]c.Item, 100)
	for j := range xs {
		xs[j] = makeItem(j, j%3)
	}
	p := plans(xs, len(xs), rand.New(rand.NewSource(3)))
	seen := map[string]bool{}
	for _, i := range p["delete"] {
		if seen[i.URL] {
			t.Fatal("duplicate deletion")
		}
		seen[i.URL] = true
	}
}
