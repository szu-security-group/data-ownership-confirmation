// HT-Merkle scalability, categorization and real-dataset evaluation.
// Measure proof objects and updated roots; perform insertion cleanup outside timing.
package main

import (
	"crypto/rand"
	"data-ownership-system/dataset"
	"data-ownership-system/pkg/types"
	"data-ownership-system/utils"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	c "ownership-comparison"
	"runtime"
	"time"
)

var n = flag.Int("n", 10000, "dataset size")
var m = flag.Int("m", 10, "category count")
var size = flag.Int("size", 1024, "payload bytes")
var repeats = flag.Int("repeat", 10, "rounds in the same process")
var ops = flag.Int("iterations", 1000, "operations per round")
var dir = flag.String("data-dir", "data", "synthetic dataset cache")
var real = flag.String("real-json", "", "preprocessed real metadata")
var out = flag.String("out", "", "JSONL output")
var group = flag.String("group", "scale", "experiment group")
var ds = flag.String("dataset-name", "", "real dataset name")

type Record struct {
	Group     string    `json:"group"`
	Dataset   string    `json:"dataset"`
	N         int       `json:"n"`
	M         int       `json:"m"`
	Round     int       `json:"round"`
	Threads   int       `json:"threads"`
	Operation string    `json:"operation"`
	Count     int       `json:"count"`
	MeanNS    float64   `json:"mean_ns"`
	SamplesNS []int64   `json:"samples_ns"`
	Work      c.Metrics `json:"work"`
	Checks    bool      `json:"checks_passed"`
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
func convert(i *types.DataItem) c.Item {
	if len(i.Hash) != 32 {
		panic("invalid digest")
	}
	var h c.Hash
	copy(h[:], i.Hash)
	return c.Item{URL: i.URL, Digest: h}
}
func load() ([]c.Item, []string, *dataset.DatasetGenerator, int) {
	cfg := &dataset.DatasetConfig{NumItems: *n, NumTypes: *m, ItemSize: *size, CompanyName: "TestCompany", Protocol: "https"}
	var old []*types.DataItem
	var names []string
	var gen *dataset.DatasetGenerator
	if *real == "" {
		gen = dataset.NewDatasetGenerator(cfg, *dir)
		var e error
		old, names, e = gen.GenerateDataset()
		check(e)
	} else {
		file, e := os.Open(*real)
		check(e)
		defer file.Close()
		var data dataset.DatasetFile
		check(json.NewDecoder(file).Decode(&data))
		cfg = data.Config
		names = data.DataTypes
		old = make([]*types.DataItem, len(data.Items))
		for j, i := range data.Items {
			h, e := hex.DecodeString(i.Hash)
			check(e)
			old[j] = &types.DataItem{URL: i.URL, Hash: h}
		}
		gen = dataset.NewDatasetGenerator(cfg, *dir)
	}
	xs := make([]c.Item, len(old))
	for j, i := range old {
		xs[j] = convert(i)
	}
	if len(xs) != cfg.NumItems || len(names) != cfg.NumTypes {
		panic("metadata counts mismatch")
	}
	return xs, names, gen, cfg.ItemSize
}

// Select queries from the beginning, middle and beginning for proofs, modifications and deletions.
func proveIndex(j, n int) int      { return j % n }
func modifyIndex(j, n int) int     { start := n / 2; return start + j%(n-start) }
func deleteCount(n, count int) int { return min(n/2, count) }
func build(xs []c.Item, names []string) (*c.Forest, int64) {
	t := time.Now()
	f, e := c.NewForest("ht", "flat", len(names), len(names), .75)
	check(e)
	check(f.ConfigureCategories(names))
	check(f.BuildPerCategory(xs))
	return f, time.Since(t).Nanoseconds()
}
func valid(f *c.Forest, i c.Item) {
	p, e := f.Prove(i)
	check(e)
	if !c.Verify("ht", f.Root(), i, p) {
		panic("membership validation failed")
	}
}
func absent(f *c.Forest, i c.Item) {
	if _, e := f.GenerateProof(i); e != c.ErrMissing {
		panic(fmt.Sprintf("deleted item not absent: %v", e))
	}
}
func round(xs []c.Item, names []string, gen *dataset.DatasetGenerator, itemSize int, count int) []Record {
	f, bt := build(xs, names)
	results := []Record{{Operation: "build", SamplesNS: []int64{bt}, Work: f.Stats()}}
	proofNS := make([]int64, count)
	proofRoot := f.Root()
	var lastProof *c.ForestProof
	var proofItem c.Item
	for j := 0; j < count; j++ {
		i := xs[proveIndex(j, len(xs))]
		t := time.Now()
		p, e := f.GenerateProof(i)
		proofNS[j] = time.Since(t).Nanoseconds()
		check(e)
		lastProof = p
		proofItem = i
	}
	results = append(results, Record{Operation: "prove", SamplesNS: proofNS})
	// Generate the insertion payload and timestamped URL.
	inserts := make([]c.Item, count)
	for j := range inserts {
		item := gen.GenerateRandomDataItem(names)
		item.URL = fmt.Sprintf("%s_test_add_%d_%d", item.URL, time.Now().UnixNano(), j)
		inserts[j] = convert(item)
	}
	// Prepare insertion capacity outside the timer.
	prepared := map[string]bool{}
	for _, i := range inserts {
		cat := c.URLCategory(i.URL)
		if !prepared[cat] {
			check(f.Apply("insert", i))
			check(f.Apply("delete", i))
			prepared[cat] = true
		}
	}
	st := f.Stats()
	ns := make([]int64, count)
	for j, i := range inserts {
		t := time.Now()
		e := f.Apply("insert", i)
		ns[j] = time.Since(t).Nanoseconds()
		check(e)
		check(f.Apply("delete", i))
	}
	insertWork := f.Stats().Sub(st) // Includes cleanup; rebuild count must still be zero.
	if insertWork.Rebuilds != 0 {
		panic("timed insertion unexpectedly expanded HT")
	}
	results = append(results, Record{Operation: "insert", SamplesNS: ns, Work: insertWork})
	ns = make([]int64, count)
	modified := make([]c.Item, count)
	st = f.Stats()
	for j := 0; j < count; j++ {
		i := xs[modifyIndex(j, len(xs))]
		content := make([]byte, itemSize)
		_, e := rand.Read(content)
		check(e)
		copy(i.Digest[:], utils.Keccak256(content))
		modified[j] = i
		t := time.Now()
		e = f.Apply("modify", i)
		ns[j] = time.Since(t).Nanoseconds()
		check(e)
	}
	results = append(results, Record{Operation: "modify", SamplesNS: ns, Work: f.Stats().Sub(st)})
	nd := deleteCount(len(xs), count)
	ns = make([]int64, nd)
	st = f.Stats()
	for j := 0; j < nd; j++ {
		t := time.Now()
		e := f.Apply("delete", xs[j])
		ns[j] = time.Since(t).Nanoseconds()
		check(e)
	}
	results = append(results, Record{Operation: "delete", SamplesNS: ns, Work: f.Stats().Sub(st)})
	// Correctness checks occur after all measured operations, never as a warmup.
	p, e := lastProof.MarshalBinary()
	check(e)
	if !c.Verify("ht", proofRoot, proofItem, p) {
		panic("proof rejected")
	}
	for j := 0; j < count; j += max(1, count/10) {
		valid(f, modified[j])
		absent(f, inserts[j])
	}
	for j := 0; j < nd; j += max(1, nd/10) {
		absent(f, xs[j])
	}
	runtime.KeepAlive(f)
	for j := range results {
		results[j].Checks = true
	}
	return results
}
func main() {
	flag.Parse()
	if *ops < 1 || *ops > 1000 || *repeats < 1 {
		panic("invalid evaluation protocol")
	}
	xs, names, gen, itemSize := load()
	file, e := os.Create(*out)
	check(e)
	defer file.Close()
	enc := json.NewEncoder(file)
	fmt.Printf("Loaded n=%d m=%d payload=%d rounds=%d operations=%d GOMAXPROCS=%d (unmodified default)\n", len(xs), len(names), itemSize, *repeats, *ops, runtime.GOMAXPROCS(0))
	for r := 1; r <= *repeats; r++ {
		rr := round(xs, names, gen, itemSize, *ops)
		for j := range rr {
			v := &rr[j]
			v.Group = *group
			v.Dataset = *ds
			v.N = len(xs)
			v.M = len(names)
			v.Round = r
			v.Threads = runtime.GOMAXPROCS(0)
			v.Count = len(v.SamplesNS)
			for _, ns := range v.SamplesNS {
				v.MeanNS += float64(ns)
			}
			v.MeanNS /= float64(v.Count)
			check(enc.Encode(v))
		}
		fmt.Printf("Round %d/%d build %.3f ms; proof/insert/modify/delete %.3f %.3f %.3f %.3f us\n", r, *repeats, rr[0].MeanNS/1e6, rr[1].MeanNS/1e3, rr[2].MeanNS/1e3, rr[3].MeanNS/1e3, rr[4].MeanNS/1e3)
	}
}
