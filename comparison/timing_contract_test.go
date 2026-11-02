package comparison

import (
	"bytes"
	"testing"
)

type encodingSpy struct {
	NativeProof
	calls *int
}

func (p *encodingSpy) MarshalBinary() ([]byte, error) {
	*p.calls++
	return p.NativeProof.MarshalBinary()
}

type proofSpy struct {
	ADS
	calls *int
}

func (a *proofSpy) GenerateProof(u string) (NativeProof, error) {
	p, e := a.ADS.GenerateProof(u)
	return &encodingSpy{p, a.calls}, e
}

func TestProofGenerationDoesNotSerialize(t *testing.T) {
	for _, kind := range Kinds {
		t.Run(kind, func(t *testing.T) {
			xs := sample(20, 1)
			f, _ := NewForest(kind, "flat", 1, 1, .75)
			if e := f.Build(xs); e != nil {
				t.Fatal(e)
			}
			calls := 0
			f.Parts[0] = &proofSpy{f.Parts[0], &calls}
			p, e := f.GenerateProof(xs[2])
			if e != nil {
				t.Fatal(e)
			}
			if calls != 0 {
				t.Fatal("proof generation serialized output")
			}
			b, e := p.MarshalBinary()
			if e != nil {
				t.Fatal(e)
			}
			if calls != 1 || !Verify(kind, f.Root(), xs[2], b) {
				t.Fatal("deferred encoding failed")
			}
		})
	}
}
func TestURLRoutingAndProofSnapshots(t *testing.T) {
	for _, kind := range Kinds {
		t.Run(kind, func(t *testing.T) {
			xs := sample(63, 3)
			f, _ := NewForest(kind, "flat", 3, 2, .75)
			if e := f.ConfigureCategories([]string{"c0000", "c0001", "c0002"}); e != nil {
				t.Fatal(e)
			}
			// Routing must parse the URL, not trust a caller-supplied category number.
			for j := range xs {
				xs[j].Category = -99
			}
			if e := f.Build(xs); e != nil {
				t.Fatal(e)
			}
			i := xs[4]
			oldRoot := f.Root()
			p, e := f.GenerateProof(i)
			if e != nil {
				t.Fatal(e)
			}
			encoded, e := p.MarshalBinary()
			if e != nil {
				t.Fatal(e)
			}
			if !Verify(kind, oldRoot, i, encoded) {
				t.Fatal("proof rejected")
			}
			changed := i
			changed.Digest = Sum([]byte("change"))
			if e := f.Apply("modify", changed); e != nil {
				t.Fatal(e)
			}
			later, e := p.MarshalBinary()
			if e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(later, encoded) || !Verify(kind, oldRoot, i, later) {
				t.Fatal("held proof mutated after update")
			}
			if Verify(kind, f.Root(), i, later) {
				t.Fatal("stale proof accepted")
			}
			for _, op := range []string{"modify", "insert", "delete"} {
				if e := f.Apply(op, Item{URL: "http://x/unknown/key"}); e != ErrMissing {
					t.Fatalf("unknown category: %s %v", op, e)
				}
			}
		})
	}
}
