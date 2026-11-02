package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	c "ownership-comparison"
	"runtime"
	"sort"
	"strings"
	"time"
)

type Result struct {
	Kind                string    `json:"kind"`
	Layout              string    `json:"layout"`
	Workload            string    `json:"workload"`
	Dataset             string    `json:"dataset"`
	N                   int       `json:"n"`
	M                   int       `json:"m"`
	Threads             int       `json:"threads"`
	Seed                int64     `json:"seed"`
	Repeat              int       `json:"repeat"`
	Operation           string    `json:"operation"`
	Count               int       `json:"count"`
	MeanNS              float64   `json:"mean_ns"`
	P50NS               float64   `json:"p50_ns"`
	P95NS               float64   `json:"p95_ns"`
	P99NS               float64   `json:"p99_ns"`
	MaxNS               float64   `json:"max_ns"`
	TotalNS             float64   `json:"total_ns"`
	ProofBytes          float64   `json:"proof_bytes"`
	Stats               c.Metrics `json:"work"`
	Status              string    `json:"status"`
	RequestedCount      int       `json:"requested_count"`
	PhaseSeconds        float64   `json:"phase_seconds"`
	HashCountsAvailable bool      `json:"hash_counts_available"`
	Extra               any       `json:"extra,omitempty"`
}

var kind = flag.String("kind", "ht", "")
var layout = flag.String("layout", "global", "")
var n = flag.Int("n", 10000, "")
var m = flag.Int("m", 10, "")
var threads = flag.Int("threads", 1, "")
var seed = flag.Int64("seed", 1, "")
var repeat = flag.Int("repeat", 0, "")
var ops = flag.Int("ops", 10000, "")
var workload = flag.String("workload", "steady", "")
var dataset = flag.String("dataset", "", "")
var output = flag.String("out", "", "")
var proofout = flag.String("proof-out", "", "")
var traceout = flag.String("trace-out", "", "")
var load = flag.Float64("load", .75, "")
var phaseBudget = flag.Float64("steady-budget", 10, "seconds per mutation phase including untimed insertion cleanup")
var dynamicBudget = flag.Float64("dynamic-budget", 60, "seconds per dynamic phase")
var categoryBudget = flag.Float64("category-budget", 60, "seconds per category operation")
var enc *json.Encoder
var categoryNames []string

func check(e error) {
	if e != nil {
		panic(e)
	}
}
func makeItem(id int, cat int) c.Item {
	var b [16]byte
	binary.BigEndian.PutUint64(b[:8], uint64(*seed))
	binary.BigEndian.PutUint64(b[8:], uint64(id))
	name := fmt.Sprintf("c%04d", cat)
	if cat >= 0 && cat < len(categoryNames) {
		name = categoryNames[cat]
	}
	return c.Item{URL: fmt.Sprintf("http://example.test/%s/item%010d", name, id), Digest: c.Sum(b[:]), Category: cat}
}
func data() []c.Item {
	r := rand.New(rand.NewSource(*seed))
	categoryNames = nil
	if *dataset != "" {
		f, e := os.Open(*dataset)
		check(e)
		defer f.Close()
		s := bufio.NewScanner(f)
		xs := []c.Item{}
		cats := map[string]int{}
		for s.Scan() {
			var v struct{ URL, Digest, Category string }
			check(json.Unmarshal(s.Bytes(), &v))
			k, ok := cats[v.Category]
			if !ok {
				k = len(cats)
				cats[v.Category] = k
				categoryNames = append(categoryNames, v.Category)
				if c.URLCategory(v.URL) != v.Category {
					panic("URL/category mismatch in dataset")
				}
			}
			h, e := hex.DecodeString(v.Digest)
			check(e)
			var d c.Hash
			copy(d[:], h)
			xs = append(xs, c.Item{URL: v.URL, Digest: d, Category: k})
		}
		check(s.Err())
		*n = len(xs)
		*m = len(cats)
		r.Shuffle(len(xs), func(i, j int) { xs[i], xs[j] = xs[j], xs[i] })
		return xs
	}
	categoryNames = make([]string, *m)
	for j := range categoryNames {
		categoryNames[j] = fmt.Sprintf("c%04d", j)
	}
	xs := make([]c.Item, *n)
	for i := range xs {
		xs[i] = makeItem(i, r.Intn(*m))
	}
	r.Shuffle(len(xs), func(i, j int) { xs[i], xs[j] = xs[j], xs[i] })
	return xs
}
func base(op string) Result {
	return Result{Kind: *kind, Layout: *layout, Workload: *workload, Dataset: *dataset, N: *n, M: *m, Threads: *threads, Seed: *seed, Repeat: *repeat, Operation: op, Status: "complete", HashCountsAvailable: *kind == "ht" || *kind == "smt"}
}
func report(op string, ns []float64, stats c.Metrics, proofBytes float64, extra any) {
	v := base(op)
	v.Count = len(ns)
	v.Stats = stats
	v.ProofBytes = proofBytes
	v.Extra = extra
	if x, ok := extra.(map[string]any); ok {
		if want, yes := x["requested_count"].(int); yes {
			v.RequestedCount = want
			if len(ns) < want {
				v.Status = "budget_exceeded"
			}
		}
		if dur, yes := x["phase_seconds"].(float64); yes {
			v.PhaseSeconds = dur
		}
		if st, yes := x["status"].(string); yes {
			v.Status = st
		}
	}
	for _, n := range ns {
		v.TotalNS += n
	}
	if len(ns) > 0 {
		v.MeanNS = v.TotalNS / float64(len(ns))
		sort.Float64s(ns)
		q := func(p float64) float64 { return ns[min(len(ns)-1, int(float64(len(ns)-1)*p))] }
		v.P50NS = q(.5)
		v.P95NS = q(.95)
		v.P99NS = q(.99)
		v.MaxNS = ns[len(ns)-1]
	}
	check(enc.Encode(v))
}
func main() {
	flag.Parse()
	runtime.GOMAXPROCS(*threads)
	if *output != "" {
		f, e := os.OpenFile(*output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		check(e)
		defer f.Close()
		enc = json.NewEncoder(f)
	} else {
		enc = json.NewEncoder(os.Stdout)
	}
	xs := data()
	runtime.GC()
	start := time.Now()
	f, e := c.NewForest(*kind, *layout, *m, *threads, *load)
	check(e)
	if *layout != "global" {
		check(f.ConfigureCategories(categoryNames))
	}
	check(f.Build(xs))
	build := time.Since(start)
	runtime.GC()
	v := base("build")
	v.Count = 1
	v.RequestedCount = 1
	v.MeanNS = float64(build.Nanoseconds())
	v.TotalNS = v.MeanNS
	v.Stats = f.Stats()
	check(enc.Encode(v))
	switch *workload {
	case "steady":
		steady(f, xs)
	case "growth", "hotspot", "churn":
		dynamic(f, xs)
	case "category":
		categories(f, xs)
	default:
		panic("unknown workload")
	}
	runtime.KeepAlive(xs)
	runtime.KeepAlive(f)
}

// plans uses disjoint, unique original keys for modification and deletion where possible.
// At the smallest scale, overlap is allowed, but deletion never repeats an absent key.
func plans(xs []c.Item, count int, r *rand.Rand) map[string][]c.Item {
	if count > len(xs) {
		panic("operation count exceeds original key count")
	}
	perm := r.Perm(len(xs))
	out := map[string][]c.Item{"insert": make([]c.Item, count), "modify": make([]c.Item, count), "delete": make([]c.Item, count)}
	for j := 0; j < count; j++ {
		out["insert"][j] = makeItem(len(xs)+j, r.Intn(*m))
		out["modify"][j] = xs[perm[(len(xs)/2+j)%len(xs)]]
		out["modify"][j].Digest = c.Sum(out["modify"][j].Digest[:], []byte("changed"))
		out["delete"][j] = xs[perm[j]]
	}
	return out
}
func steady(f *c.Forest, xs []c.Item) {
	r := rand.New(rand.NewSource(*seed + 123))
	ids := make([]int, *ops)
	for j := range ids {
		ids[j] = r.Intn(len(xs))
	}
	requests := plans(xs, *ops, r)
	for j := 0; j < min(100, len(ids)); j++ {
		i := xs[ids[j]]
		p, e := f.Prove(i)
		check(e)
		if !c.Verify(*kind, f.Root(), i, p) {
			panic("proof validation failed")
		}
	}
	var pf *os.File
	var pe *json.Encoder
	if *proofout != "" {
		var e error
		pf, e = os.OpenFile(*proofout, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		check(e)
		defer pf.Close()
		pe = json.NewEncoder(pf)
	}
	ns := make([]float64, *ops)
	before := f.Stats()
	for j, id := range ids {
		i := xs[id]
		t := time.Now()
		proof, e := f.GenerateProof(i)
		ns[j] = float64(time.Since(t).Nanoseconds())
		check(e)
		// Serialization, correctness checking and sample output are outside the timer.
		p, e := proof.MarshalBinary()
		check(e)
		if !c.Verify(*kind, f.Root(), i, p) {
			panic("proof rejected")
		}
		if pe != nil && j < 3 {
			root := f.Root()
			check(pe.Encode(map[string]any{"kind": *kind, "layout": *layout, "n": *n, "m": *m, "dataset": *dataset, "seed": *seed, "url": i.URL, "digest": "0x" + hex.EncodeToString(i.Digest[:]), "root": "0x" + hex.EncodeToString(root[:]), "proof": "0x" + hex.EncodeToString(p)}))
		}
	}
	report("prove", ns, f.Stats().Sub(before), 0, map[string]any{"requested_count": *ops, "proof_output_encoding_timed": false})
	// Prepare one insertion slot per requested category using the same public
	// operations for every component. This is outside all reported timers.
	preparationBefore := f.Stats()
	prepared := map[int]bool{}
	for _, i := range requests["insert"] {
		if prepared[i.Category] {
			continue
		}
		check(f.Apply("insert", i))
		check(f.Apply("delete", i))
		prepared[i.Category] = true
	}
	preparationWork := f.Stats().Sub(preparationBefore)
	prefixComplete := true
	for _, op := range []string{"insert", "modify", "delete"} {
		ns = make([]float64, *ops)
		total := c.Metrics{}
		completed := 0
		var requestBytes []byte
		for _, item := range requests[op] {
			requestBytes = append(requestBytes, item.URL...)
			requestBytes = append(requestBytes, 0)
			requestBytes = append(requestBytes, item.Digest[:]...)
		}
		requestHash := c.Sum(requestBytes)
		phaseStart := time.Now()
		for j, i := range requests[op] {
			if completed > 0 && time.Since(phaseStart).Seconds() >= *phaseBudget {
				break
			}
			st := f.Stats()
			t := time.Now()
			e := f.Apply(op, i)
			ns[j] = float64(time.Since(t).Nanoseconds())
			check(e)
			total = total.Add(f.Stats().Sub(st))
			if op == "insert" {
				// Keep the live item count fixed; cleanup is not insertion latency.
				check(f.Apply("delete", i))
			}
			completed++
		}
		elapsed := time.Since(phaseStart).Seconds()
		// Check sampled postconditions without charging validation to the measured phase.
		for j := 0; j < completed; j += max(1, completed/10) {
			i := requests[op][j]
			p, e := f.Prove(i)
			if op == "delete" || op == "insert" {
				if e == nil {
					panic("removed key still has proof")
				}
			} else {
				check(e)
				if !c.Verify(*kind, f.Root(), i, p) {
					panic("updated root/proof mismatch")
				}
			}
		}
		if op == "insert" && *kind == "ht" && total.Rebuilds != 0 {
			panic("timed insertion unexpectedly expanded HT")
		}
		report(op, ns[:completed], total, 0, map[string]any{"requested_count": *ops, "phase_seconds": elapsed, "input_prefix_complete": prefixComplete, "postconditions_checked": true, "request_digest": hex.EncodeToString(requestHash[:]), "restore_between_operations": op == "insert", "insertion_preparation_categories": len(prepared), "insertion_preparation_rebuilds": preparationWork.Rebuilds})
		prefixComplete = prefixComplete && completed == *ops
	}
}

func nextChurnItem(r *rand.Rand, live []c.Item, id int) (c.Item, int) {
	index := r.Intn(len(live))
	return makeItem(id, live[index].Category), index
}

func dynamic(f *c.Forest, xs []c.Item) {
	r := rand.New(rand.NewSource(*seed + 123))
	steps := *ops
	ns := make([]float64, 0, steps)
	s := f.Stats()
	var tr *json.Encoder
	var tf *os.File
	if *traceout != "" {
		var e error
		tf, e = os.Create(*traceout)
		check(e)
		defer tf.Close()
		tr = json.NewEncoder(tf)
	}
	total := 0.
	events := []map[string]any{}
	window := []float64{}
	live := append([]c.Item(nil), xs...)
	phaseStart := time.Now()
	for j := 0; j < steps; j++ {
		if j > 0 && time.Since(phaseStart).Seconds() >= *dynamicBudget {
			break
		}
		cat := r.Intn(*m)
		if *workload == "hotspot" && r.Float64() < .9 {
			cat = r.Intn(max(1, *m/10))
		}
		var i c.Item
		deleteIndex := -1
		if *workload == "churn" {
			i, deleteIndex = nextChurnItem(r, live, *n+j)
		} else {
			i = makeItem(*n+j, cat)
		}
		st := f.Stats()
		t := time.Now()
		if *workload == "churn" {
			check(f.Apply("delete", live[deleteIndex]))
			check(f.Apply("insert", i))
			live[deleteIndex] = i
		} else {
			check(f.Apply("insert", i))
		}
		elapsed := float64(time.Since(t).Nanoseconds())
		total += elapsed
		ns = append(ns, elapsed)
		window = append(window, elapsed)
		diff := f.Stats().Sub(st)
		if diff.Rebuilds > 0 {
			events = append(events, map[string]any{"step": j + 1, "ns": elapsed, "items": diff.RehashedItems})
			if tr != nil {
				check(tr.Encode(map[string]any{"step": j + 1, "ns": elapsed, "total_ns": total, "rebuild": true, "items": diff.RehashedItems}))
			}
		}
		if tr != nil && (j+1)%1000 == 0 {
			sort.Float64s(window)
			check(tr.Encode(map[string]any{"step": j + 1, "ns": window[len(window)/2], "p99_ns": window[int(float64(len(window)-1)*.99)], "max_ns": window[len(window)-1], "total_ns": total, "rebuild": false}))
			window = window[:0]
		}
	}
	measuredWork := f.Stats().Sub(s)
	checkedMembers := 0
	if *workload == "churn" {
		for j := 0; j < min(100, len(live)); j++ {
			i := live[(j*7919)%len(live)]
			p, e := f.Prove(i)
			check(e)
			if !c.Verify(*kind, f.Root(), i, p) {
				panic("final churn membership rejected")
			}
			checkedMembers++
		}
	}
	report(*workload, ns, measuredWork, 0, map[string]any{"requested_count": steps, "phase_seconds": time.Since(phaseStart).Seconds(), "events": events, "transaction_unit": map[bool]string{true: "delete+insert pair", false: "insert"}[*workload == "churn"], "final_membership_checks": checkedMembers})
	runtime.KeepAlive(live)
}
func categories(f *c.Forest, xs []c.Item) {
	for _, fraction := range []float64{.01, .1, .3} {
		k := int(float64(*n) * fraction)
		extra := make([]c.Item, k)
		for j := range extra {
			extra[j] = makeItem(*n+j, *m)
		}
		for _, op := range []string{"add_category", "drop_category"} {
			st := f.Stats()
			t := time.Now()
			c.CategoryDeadline = t.Add(time.Duration(*categoryBudget * float64(time.Second)))
			var e error
			if op == "add_category" {
				e = f.AddCategory(extra)
			} else {
				e = f.DropLastCategory(extra)
			}
			c.CategoryDeadline = time.Time{}
			status := "complete"
			if e == c.ErrBudget {
				status = "budget_exceeded"
			} else {
				check(e)
			}
			report(op, []float64{float64(time.Since(t).Nanoseconds())}, f.Stats().Sub(st), 0, map[string]any{"fraction": fraction, "items": k, "includes_gc": false, "status": status, "completed_items": c.CategoryCompleted, "requested_count": 1})
			if status != "complete" {
				return
			}
		}
	}
}

var _ = strings.Compare
