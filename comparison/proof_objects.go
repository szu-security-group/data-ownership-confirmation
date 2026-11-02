package comparison

import (
	"encoding/binary"
	"github.com/celestiaorg/smt"
	ics23 "github.com/cosmos/ics23/go"
)

// NativeProof is the in-memory proof returned by the component's native generation API.
// MarshalBinary is intentionally separate and is not part of proof-generation timing.
// Encoding intrinsic to authentication (e.g. MPT's RLP nodes) remains in generation.
type NativeProof interface{ MarshalBinary() ([]byte, error) }

type hashPathProof struct {
	Siblings   []Hash
	Directions []byte
}

func (p *hashPathProof) MarshalBinary() ([]byte, error) {
	b := make([]byte, 2+33*len(p.Siblings))
	binary.BigEndian.PutUint16(b, uint16(len(p.Siblings)))
	for j, h := range p.Siblings {
		b[2+33*j] = p.Directions[j]
		copy(b[3+33*j:], h[:])
	}
	return b, nil
}

type merklePathProof struct {
	Siblings   [][]byte
	Directions []int64
}

func (p *merklePathProof) MarshalBinary() ([]byte, error) {
	b := make([]byte, 2+33*len(p.Siblings))
	binary.BigEndian.PutUint16(b, uint16(len(p.Siblings)))
	for j, h := range p.Siblings {
		b[2+33*j] = byte(1 - p.Directions[j])
		copy(b[3+33*j:], h)
	}
	return b, nil
}

type iavlProof struct{ p *ics23.CommitmentProof }

func (p *iavlProof) MarshalBinary() ([]byte, error) { return p.p.Marshal() }

type smtProof struct{ p smt.SparseCompactMerkleProof }

func (q *smtProof) MarshalBinary() ([]byte, error) {
	p := q.p
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(p.NumSideNodes))
	b = append(b, p.BitMask...)
	for _, h := range p.SideNodes {
		b = append(b, h...)
	}
	return b, nil
}
func (p *proofDB) MarshalBinary() ([]byte, error) {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(len(p.nodes)))
	for _, n := range p.nodes {
		var l [4]byte
		binary.BigEndian.PutUint32(l[:], uint32(len(n)))
		b = append(b, l[:]...)
		b = append(b, n...)
	}
	return b, nil
}

type ForestProof struct {
	Mode                    byte
	Category, CategoryCount int
	Aux                     []Hash
	Sub                     NativeProof
}

func (p *ForestProof) MarshalBinary() ([]byte, error) {
	sub, e := p.Sub.MarshalBinary()
	if e != nil {
		return nil, e
	}
	b := make([]byte, 11, 11+32*len(p.Aux)+len(sub))
	b[0] = p.Mode
	binary.BigEndian.PutUint32(b[1:], uint32(p.Category))
	binary.BigEndian.PutUint32(b[5:], uint32(p.CategoryCount))
	binary.BigEndian.PutUint16(b[9:], uint16(len(p.Aux)))
	for _, h := range p.Aux {
		b = append(b, h[:]...)
	}
	return append(b, sub...), nil
}
