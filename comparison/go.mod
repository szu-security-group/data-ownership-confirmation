module ownership-comparison

go 1.24.0

require (
	github.com/celestiaorg/smt v0.3.0
	github.com/ethereum/go-ethereum v1.17.2
	golang.org/x/crypto v0.44.0
)

require (
	cosmossdk.io/core v0.12.1-0.20240725072823-6a2d039e1212 // indirect
	github.com/bits-and-blooms/bitset v1.20.0 // indirect
	github.com/consensys/gnark-crypto v0.18.1 // indirect
	github.com/cosmos/gogoproto v1.5.0 // indirect
	github.com/crate-crypto/go-eth-kzg v1.5.0 // indirect
	github.com/emicklei/dot v1.6.2 // indirect
	github.com/ferranbt/fastssz v0.1.4 // indirect
	github.com/gofrs/flock v0.12.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

// Minimal, published Keccak substitutions for the controlled comparison.
replace github.com/cosmos/iavl => ./third_party/iavl

replace github.com/cosmos/ics23/go => ./third_party/ics23

require (
	data-ownership-system v0.0.0
	github.com/cbergoon/merkletree v0.5.0
	github.com/cosmos/iavl v1.3.6
	github.com/cosmos/ics23/go v0.10.0
)

replace data-ownership-system => ..
