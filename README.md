# Data Ownership Confirmation and Arbitration

This repository contains the experimental artifact for a blockchain-assisted data ownership confirmation and arbitration scheme. The implementation builds an authenticated data structure over large dynamic datasets and generates layered membership proofs.

## What We Implement

- Off-chain authenticated data structure in Go: hash tables, Merkle subtrees, top-level root aggregation, proof generation, and dynamic updates.
- Synthetic and preprocessed real-dataset benchmarks.
- Solidity contract source for root registration and layered proof verification.

## Quick Start

Install dependencies:

```bash
go mod download
```

Run a small off-chain benchmark:

```bash
go run main.go --n 10000 --m 10 --repeat 1
```

Run quietly and measure wall-clock time:

```bash
time go run main.go --n 100000 --m 10 --repeat 3 --silent
```

Emit one proof in JSON format:

```bash
go run main.go --n 10000 --m 10 --emit-proof --emit-index 0
```

## CLI Options

```text
--n            number of synthetic data items
--m            number of synthetic data types
--size         synthetic item size in bytes
--iterations   per-operation benchmark iterations
--repeat       number of benchmark runs
--load         hash table load factor
--real         load a preprocessed real dataset from data/
--dataset      preprocessed real dataset JSON filename
--company      company name used in generated URLs
--protocol     URL protocol used in generated URLs
--emit-proof   emit one layered proof as JSON
--emit-index   proof item index
--silent       suppress benchmark progress output
```

## Real Dataset Preprocessing

```bash
go run cmd/preprocessor/main.go \
  --path /path/to/dataset \
  --protocol http \
  --company example.org \
  --output RealDataset.json

go run main.go --real --dataset RealDataset.json
```

## Repository Layout

```text
auth/          authenticated data structure
hashtable/     open-addressing hash table
merkle/        Merkle tree and proof paths
dataset/       synthetic dataset generator
cmd/           command-line entry points
preprocessor/  real-dataset preprocessing
contracts/     Solidity contracts
pkg/           shared data types
utils/         hashing and timing helpers
```

Large generated datasets, caches, and dependency directories are intentionally excluded from the repository.
