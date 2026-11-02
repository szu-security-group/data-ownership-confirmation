package main

import (
	"bytes"
	"data-ownership-system/dataset"
	"data-ownership-system/utils"
	c "ownership-comparison"
	"testing"
)

func TestOriginalSelection(t *testing.T) {
	for j := 0; j < 1000; j++ {
		if proveIndex(j, 10000) != j || modifyIndex(j, 10000) != 5000+j {
			t.Fatal("query selection does not match the protocol")
		}
	}
	if deleteCount(10000, 1000) != 1000 {
		t.Fatal("delete count")
	}
}
func TestOriginalPayloadAndRound(t *testing.T) {
	cfg := &dataset.DatasetConfig{NumItems: 2000, NumTypes: 10, ItemSize: 1024, CompanyName: "TestCompany", Protocol: "https"}
	gen := dataset.NewDatasetGenerator(cfg, t.TempDir())
	old, names, e := gen.GenerateDataset()
	if e != nil {
		t.Fatal(e)
	}
	dynamic := gen.GenerateRandomDataItem(names)
	if len(dynamic.Content) != 1024 || !bytes.Equal(dynamic.Hash, utils.Keccak256(dynamic.Content)) {
		t.Fatal("expected a 1 KiB payload")
	}
	xs := make([]c.Item, len(old))
	for j, i := range old {
		xs[j] = convert(i)
	}
	for r := 0; r < 2; r++ {
		rs := round(xs, names, gen, 1024, 100)
		if len(rs) != 5 {
			t.Fatal("missing operation")
		}
		for _, v := range rs {
			want := 100
			if v.Operation == "build" {
				want = 1
			}
			if len(v.SamplesNS) != want || !v.Checks {
				t.Fatal("incomplete round")
			}
			if v.Operation == "insert" && v.Work.Rebuilds != 0 {
				t.Fatal("timed expansion")
			}
		}
	}
}
