package comparison

import (
	"bytes"
	"encoding/binary"
	mt "github.com/cbergoon/merkletree"
	ics23 "github.com/cosmos/ics23/go"
	"golang.org/x/crypto/sha3"
	"testing"
)

func TestNativeMembershipVerifiers(t *testing.T) {
	for _, n := range []int{1, 2, 3, 17, 137} {
		xs := sample(n, 1)
		a := newIAVL()
		b := newMerkle()
		if e := a.Build(xs); e != nil {
			t.Fatal(e)
		}
		if e := b.Build(xs); e != nil {
			t.Fatal(e)
		}
		for _, x := range xs {
			p, e := a.Prove(x.URL)
			if e != nil {
				t.Fatal(e)
			}
			var q ics23.CommitmentProof
			if e = q.Unmarshal(p); e != nil {
				t.Fatal(e)
			}
			root := a.Root()
			if !ics23.VerifyMembership(ics23.IavlSpec, root[:], &q, []byte(x.URL), x.Digest[:]) {
				t.Fatal("native IAVL verifier rejected")
			}
			got, e := iavlRoot(x, p)
			if e != nil || got != root {
				t.Fatal("IAVL root reconstruction differs")
			}
			p, e = b.ProveItem(x)
			if e != nil {
				t.Fatal(e)
			}
			depth := int(binary.BigEndian.Uint16(p))
			path := make([][]byte, depth)
			dirs := make([]int64, depth)
			for j := 0; j < depth; j++ {
				path[j] = p[3+33*j : 35+33*j]
				dirs[j] = int64(1 - p[2+33*j])
			}
			root = b.Root()
			ok, e := mt.VerifyProof(merkleContent{x}, path, dirs, root[:], mt.WithHasher(sha3.NewLegacyKeccak256))
			if e != nil || !ok {
				t.Fatal("native Merkle verifier rejected", e)
			}
		}
		contents := make([]mt.Content, len(xs))
		for j, x := range xs {
			contents[j] = merkleContent{x}
		}
		plain, e := mt.NewTreeWithOptions(contents, mt.WithHasher(sha3.NewLegacyKeccak256))
		if e != nil {
			t.Fatal(e)
		}
		root := b.Root()
		if !bytes.Equal(plain.MerkleRoot(), root[:]) {
			t.Fatal("index changed root")
		}
	}
}
