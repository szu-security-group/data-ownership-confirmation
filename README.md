# Data Ownership Confirmation and Arbitration Based on Blockchain

Data ownership confirmation is a prerequisite for trustworthy data circulation.
Existing blockchain-based schemes often register each data item independently,
which leads to linear on-chain storage cost and poor support for dynamic updates.
This project implements a scalable data ownership confirmation and arbitration
scheme for large, heterogeneous, and frequently updated datasets.

The core idea is to model ownership confirmation as an authenticated data
structure problem. A data owner commits to an entire dataset with one compact
root hash on chain, while membership proofs for individual data items are
generated off chain and verified by smart contracts when arbitration is needed.

This repository contains the experimental implementation used to evaluate the
proposed scheme.

## System Model

<figure>
  <p align="center">
    <img src="./img/model.png" alt="system model" width="70%">
  </p>
  <p align="center">Figure 1: System model</p>
</figure>

The system includes the following entities.

- Data owner: organizes a large dataset, builds the off-chain authenticated data
  structure, registers the dataset root on chain, and generates ownership proofs
  for disputed data items.
- Data elements market infrastructure: provides data publishing, data search,
  compliance review, and certificate/arbitration support.
- Blockchain and smart contracts: maintain the registered dataset root and
  verify submitted ownership proofs in a public and tamper-resistant manner.
- Data consumer and arbitration process: use the on-chain verification result as
  credible evidence in ownership disputes.

## What We Do

- We build a layered authenticated data structure for large heterogeneous
  datasets. Data items are first grouped by type, stored in hash tables, and then
  authenticated by Merkle subtrees.
- We aggregate all subtree roots into one top root, so the blockchain only needs
  to store a constant-size commitment for the whole dataset.
- We support dynamic insert, update, and delete operations by locally updating
  the affected hash table and Merkle subtree.
- We generate membership proofs for individual data items and provide Solidity
  contract source code for root registration and layered proof verification.
- We provide synthetic and preprocessed real-dataset benchmarks for evaluating
  construction time, proof generation time, verification time, update time, and
  storage overhead.

## Usage

### Installation

The prototype is written in Go. Make sure Go 1.21 or later is installed.

```bash
go mod download
```

Run a quick sanity check:

```bash
go test ./...
go run main.go --n 100 --m 5 --repeat 1 --iterations 5 --silent
```

### Run a Synthetic Benchmark

```bash
go run main.go --n 10000 --m 10 --repeat 1
```

Run silently and measure wall-clock time:

```bash
time go run main.go --n 100000 --m 10 --repeat 3 --silent
```

### Emit an Ownership Proof

```bash
go run main.go --n 10000 --m 10 --emit-proof --emit-index 0
```

The proof object is printed in JSON format at the end of the run. It contains
the top root, data hash, URL, Merkle proof path, other subtree roots, data type
index, and leaf index.

### Preprocess a Real Dataset

The real dataset preprocessor expects a directory organized by data type:

```text
dataset-root/
  images/
    file_1
    file_2
  audio/
    file_3
```

Generate the JSON dataset file:

```bash
go run cmd/preprocessor/main.go \
  --path /path/to/dataset-root \
  --protocol http \
  --company example.org \
  --output RealDataset.json
```

Run the benchmark on the preprocessed dataset:

```bash
go run main.go --real --dataset RealDataset.json
```

Generated dataset JSON files are stored in `data/` and are ignored by Git.

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

## Directory Explanation

```text
auth/          layered authenticated data structure
cmd/           command-line entry points
contracts/     Solidity source code for SCReg and SCArb
dataset/       synthetic dataset generator
hashtable/     open-addressing hash table
img/           figures used by this README
merkle/        Merkle tree and proof path implementation
pkg/           shared data types
preprocessor/  real-dataset preprocessing
utils/         hashing and timing helpers
main.go        benchmark and proof-generation entry point
```

The `contracts/` directory contains the smart contract source code used by the
scheme. Hardhat deployment scripts and JavaScript tests are intentionally not
included in this artifact.

## Notes

This is a research prototype for reproducing the paper experiments. Generated
datasets, build caches, dependency directories, and other local artifacts are
excluded from the repository.
