# HT-Merkle evaluation

This experiment measures dataset-size scalability, category-count scalability and performance on real datasets, using the HT implementation in `../comparison`.

## Run

Use Go 1.26.2. Install the plotting dependencies and prepare the three real datasets as described in [comparison/README.md](../comparison/README.md). From the repository root:

```sh
go run evaluation/prepare_real.go comparison/datasets evaluation/data
cd comparison
go test ./...
go build -o ../evaluation/bench ./cmd/evaluate
cd ../evaluation
python3 run.py
python3 analyze.py
../comparison/.venv/bin/python plot.py
```

Metadata preparation refuses to overwrite existing input files. `run.py` generates missing synthetic metadata before measurement, using `dataset/generator.go` through `cmd/generate-dataset`. Dataset caches, binaries and results are generated locally and excluded from Git. An interrupted run can resume only with the same binary and input checksums; use a fresh checkout for a new run.

## Protocol

- Synthetic payload: 1,024 pseudorandom bytes per item; metadata stores URLs and Keccak-256 digests. Generation and content hashing are outside ADS timing; load factor is 0.75.
- Dataset size: N = 10,000 / 100,000 / 1,000,000 / 2,000,000 / 5,000,000; m = 10.
- Category count: N = 2,000,000; m = 1 / 10 / 100 / 1,000 / 5,000.
- Real datasets: COCO 2017, CLEVR v1.0 and LibriSpeech train-other-500. Metadata is grouped by category and ordered lexically within each category. Category-group iteration order is unspecified.
- Ten rounds per configuration, with 1,000 operations per type per round. Proofs and deletions use the first 1,000 items; modifications use 1,000 items starting at the midpoint. Results average the per-round means.
- Construction uses one goroutine per category and default GOMAXPROCS. No explicit garbage collection, warm-up or phase time limit is applied.
- Construction and update timing ends at the global root; proof timing ends at the native proof object. Insertion capacity preparation and deletion cleanup occur outside timing.

`results/jobs/` stores per-round measurements and operation samples; `results/manifest.json` records inputs and protocol. `analyze.py` checks completeness and writes `output/summary.csv` and `summary.json`. `plot.py` writes the scale, category-count and real-dataset PDF/PNG figures to `output/figures/`. These commands do not modify the manuscript.
