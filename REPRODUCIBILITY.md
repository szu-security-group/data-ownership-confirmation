# Reproducibility Guide

This repository contains the Go prototype and Solidity contracts used in the
paper. This guide documents the deterministic reproduction path for Table III,
the on-chain gas evaluation.

## Scope

The Table III harness reproduces gas consumption for contract deployment,
registration, root update, and layered membership-proof verification. It runs
on the local Hardhat network and does not require an RPC endpoint, private key,
or access to the raw evaluation datasets.

For large verification cases, the harness constructs deterministic proof
calldata with the category capacity and Merkle-path length implied by the
reported total item count and category count. It does not materialize the full
raw dataset: the contracts store only a root commitment, and verification gas
is determined by the submitted proof calldata rather than raw item contents.

## Requirements

- Node.js 20.x (see `.nvmrc`)
- npm 9.x or later

The repository pins Hardhat, Ethers, Solidity 0.8.19, optimizer settings, the
Paris EVM target, and the Shanghai Hardhat-network hardfork. The compiler is
loaded from the npm-pinned `solc` package; no compiler download is required
after `npm ci`.

## Reproduce Table III

From the repository root, run:

```bash
npm ci
npm run reproduce:table3
```

The command first removes prior Hardhat build artifacts, then compiles and runs
the deterministic gas harness. It writes:

```text
reproduction/table-iii-gas.json
reproduction/table-iii-gas.md
```

The report lists the gas and the corresponding ETH/USD conversion for every
Table III row.

The USD amounts in the paper are deterministic conversions from the fixed
1.61 Gwei gas price and 3158.53 ETH/USD exchange rate encoded in
`scripts/reproduce-table-iii.js`. They are not current mainnet fee estimates.

## Contract Tests

Run the behavioral tests with:

```bash
npm test
```

The tests cover deployment, registration, in-place root update, expiry checks,
and acceptance or rejection of layered membership proofs.

## Notes on Off-Chain Results

The Go prototype can be installed with `go mod download` and checked with
`go test ./...`. Wall-clock results for ADS construction and update operations
depend on CPU, memory, Go version, and dataset layout. The raw real-world
datasets are not redistributed in this repository; users should prepare them
with `cmd/preprocessor/main.go` as described in `README.md`.
