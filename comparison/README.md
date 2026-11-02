# Authenticated data structure comparison

This module compares five interchangeable components under identical inputs, operation sequences and category aggregation:

| Component | Implementation | Configuration |
|---|---|---|
| HT-Merkle | `dense.go` | Double-hash open addressing, power-of-two capacity, load factor 0.75 |
| Merkle | cbergoon/merkletree v0.5.0 | `WithLeafIndex`, Keccak-256; updates rebuild the affected category |
| IAVL | cosmos/iavl v1.3.6 | Keccak-256 adaptation, in-memory current working tree |
| SMT | celestiaorg/smt v0.3.0 | Keccak-256, compact membership proofs |
| MPT | go-ethereum v1.17.2 | In-memory trie, Keccak-256 keys, native RLP proof nodes |

IAVL and ICS23 Go v0.10.0 are included in `third_party/`. The Keccak adaptation changes hashing and proof verification while preserving node encodings and AVL balancing rules. The patch, source checksums and upstream licenses accompany the source. The parent module supplies shared data types and hashing utilities; clone the entire repository. The independent implementation under `internal/reference/` is used only by correctness tests to check authentication encodings and operation semantics; benchmarks use `dense.go`.

## Setup and datasets

Use Go 1.26.2 and Python 3.9 or later. From the repository root:

```sh
python3 -m venv comparison/.venv
comparison/.venv/bin/pip install -r comparison/requirements-plot.txt
cd comparison
python3 verify_dependencies.py
go test ./...
go test -race ./...
mkdir -p bin results
go build -o bin/bench ./cmd/bench
python3 run_experiment.py --profile smoke
```

The smoke test runs all five structures on synthetic inputs and does not need dataset downloads. For the full experiment, prepare COCO 2017 (train, validation, test and annotations), CLEVR v1.0, and LibriSpeech train-other-500:

```sh
# From comparison/; downloads approximately 75 GB.
python3 download_datasets.py
go run ./cmd/preprocess
```

Official download URLs are in `download_datasets.py`. Existing archives with their official filenames can instead be placed in `comparison/` or `comparison/datasets/`. Preprocessing verifies archives and creates `datasets/{coco,clevr,librispeech}.jsonl` and `datasets/validation.json`. The expected item/category counts are 163,963/4, 100,005/5 and 151,472/1,166. Content hashing and file I/O occur before benchmark timing. Datasets and generated measurements are not committed.

## Run and plot

From `comparison/`, after the setup above:

```sh
IAVL_FIXTURE_OUT=results/iavl-keccak-fixtures.jsonl go test -count=1 ./... > results/go-test.log
go test -race ./... > results/go-race.log
python3 record_environment.py
python3 run_experiment.py --profile paper
python3 audit_results.py
.venv/bin/python analyze.py
```

`record_environment.py` records toolchain, source, dataset and binary checksums; its CPU detection uses macOS `sysctl`. It refuses to overwrite an existing environment record. Resume an interrupted run with the same command, binary and inputs. Use a separate checkout with empty result directories for a new run.

The full experiment runs 200 processes sequentially: five structures × five synthetic sizes × five seeds, plus five structures × three real datasets × five seeds. Synthetic sizes are 10,000, 100,000, 1,000,000, 2,000,000 and 5,000,000 with ten categories. Each category has an independent subtree; ordered category roots are hashed into the global root. Construction uses four workers and `GOMAXPROCS=4`.

Each process measures construction and 10,000 operations per type: proof generation, insertion, modification and deletion. Construction includes category routing, grouping, component creation, tree construction and root aggregation. Proof timing ends at the native proof object; additional output serialization and validation are outside timing. Updates include generation of the new global root. Insertions are followed by deletion outside timing to retain the dataset size; capacity preparation is also outside timing.

Each update phase has a separate 10-second time limit, checked between complete operations. Partial phases are recorded explicitly. A plotted point requires all five seeds to complete that phase and its preceding update phases; missing points are not zeros.

Outputs are generated locally:

- `results/paper/`: run manifest, per-process JSONL measurements and proof samples.
- `output/summary.csv` and `summary.json`: complete-cell means.
- `output/completion.csv` and `censored_runs.jsonl`: completion coverage and excluded partial measurements.
- `output/figures/scale.pdf` and `real_datasets.pdf`: synthetic and real-data comparisons, with PNG copies.
