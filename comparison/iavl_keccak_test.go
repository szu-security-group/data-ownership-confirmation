package comparison

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	ics23 "github.com/cosmos/ics23/go"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"math/rand"
	"os"
	"sort"
	"testing"
)

// Independent reconstruction: Geth Keccak, without ICS23 Calculate or our Sum.
func oracleIAVL(t *testing.T, x Item, p []byte) Hash {
	t.Helper()
	var q ics23.CommitmentProof
	if err := q.Unmarshal(p); err != nil {
		t.Fatal(err)
	}
	e := q.GetExist()
	if e == nil || e.Leaf.Hash != ics23.HashOp_KECCAK || e.Leaf.PrehashValue != ics23.HashOp_KECCAK {
		t.Fatal("proof must identify Keccak")
	}
	if !bytes.Equal(e.Key, []byte(x.URL)) || !bytes.Equal(e.Value, x.Digest[:]) {
		t.Fatal("key/value mismatch")
	}
	b := append([]byte(nil), e.Leaf.Prefix...)
	b = binary.AppendUvarint(b, uint64(len(e.Key)))
	b = append(b, e.Key...)
	v := ethcrypto.Keccak256(e.Value)
	b = binary.AppendUvarint(b, uint64(len(v)))
	b = append(b, v...)
	h := ethcrypto.Keccak256(b)
	for _, op := range e.Path {
		if op.Hash != ics23.HashOp_KECCAK {
			t.Fatal("inner op is not Keccak")
		}
		h = ethcrypto.Keccak256(op.Prefix, h, op.Suffix)
	}
	q.GetExist().Leaf.Hash = ics23.HashOp_SHA256
	if ics23.VerifyMembership(ics23.IavlSpec, h, &q, []byte(x.URL), x.Digest[:]) {
		t.Fatal("SHA proof accepted under Keccak profile")
	}
	var out Hash
	copy(out[:], h)
	return out
}
func TestIAVLKeccakIndependentOracle(t *testing.T) {
	var fixtures *json.Encoder
	if path := os.Getenv("IAVL_FIXTURE_OUT"); path != "" {
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		fixtures = json.NewEncoder(f)
	}
	wrap := func(p []byte) string {
		b := make([]byte, 11)
		binary.BigEndian.PutUint32(b[5:], 1)
		return "0x" + hex.EncodeToString(append(b, p...))
	}

	for seed := int64(1); seed <= 5; seed++ {
		a := newIAVL()
		if err := a.Build(nil); err != nil {
			t.Fatal(err)
		}
		if a.Root() != Hash(ethcrypto.Keccak256Hash(nil)) {
			t.Fatal("wrong empty root")
		}
		state := map[string]Item{}
		r := rand.New(rand.NewSource(seed))
		for step := 0; step < 500; step++ {
			x := Item{URL: fmt.Sprintf("key/%04d", r.Intn(90)), Digest: Sum([]byte(fmt.Sprint(seed, step)))}
			_, ok := state[x.URL]
			var err error
			if !ok {
				err = a.Insert(x)
				state[x.URL] = x
			} else if r.Intn(3) == 0 {
				err = a.Delete(x.URL)
				delete(state, x.URL)
			} else {
				err = a.Modify(x)
				state[x.URL] = x
			}
			if err != nil {
				t.Fatal(err)
			}
			root := a.Root()
			if fixtures != nil && (step == 0 || step == 99 || step == 199 || step == 399 || step == 499) {
				keys := make([]string, 0, len(state))
				for k := range state {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				x := state[keys[0]]
				p, err := a.Prove(x.URL)
				if err != nil {
					t.Fatal(err)
				}
				var q ics23.CommitmentProof
				if err = q.Unmarshal(p); err != nil {
					t.Fatal(err)
				}
				e := q.GetExist()
				e.Leaf.Hash = ics23.HashOp_SHA256
				e.Leaf.PrehashValue = ics23.HashOp_SHA256
				for _, op := range e.Path {
					op.Hash = ics23.HashOp_SHA256
				}
				bad, err := q.Marshal()
				if err != nil {
					t.Fatal(err)
				}
				shaRoot, err := e.Calculate()
				if err != nil {
					t.Fatal(err)
				}
				if err = q.Unmarshal(p); err != nil {
					t.Fatal(err)
				}
				q.GetExist().Leaf.Prefix[0] = 2
				badMeta, err := q.Marshal()
				if err != nil {
					t.Fatal(err)
				}
				err = fixtures.Encode(map[string]any{"seed": seed, "step": step, "kind": "iavl", "url": x.URL, "digest": "0x" + hex.EncodeToString(x.Digest[:]), "root": "0x" + hex.EncodeToString(root[:]), "proof": wrap(p), "sha_root": "0x" + hex.EncodeToString(shaRoot), "sha_proof": wrap(bad), "metadata_proof": wrap(badMeta)})
				if err != nil {
					t.Fatal(err)
				}
			}
			if int(a.t.Size()) != len(state) {
				t.Fatal("size mismatch")
			}
			if step%23 == 0 || step == 499 {
				visited := 0
				previous := ""
				_, err = a.t.Iterate(func(k, v []byte) bool {
					x, ok := state[string(k)]
					if !ok || !bytes.Equal(v, x.Digest[:]) || string(k) <= previous {
						t.Fatal("iteration differs from reference map")
					}
					previous = string(k)
					visited++
					return false
				})
				if err != nil || visited != len(state) {
					t.Fatal("iteration size", err)
				}
				for _, v := range state {
					p, err := a.Prove(v.URL)
					if err != nil {
						t.Fatal(err)
					}
					if oracleIAVL(t, v, p) != root {
						t.Fatal("independent root mismatch")
					}
				}
			}
		}
		for k := range state {
			if err := a.Delete(k); err != nil {
				t.Fatal(err)
			}
		}
		if a.Root() != Hash(ethcrypto.Keccak256Hash(nil)) {
			t.Fatal("delete-to-empty root")
		}
	}
}
