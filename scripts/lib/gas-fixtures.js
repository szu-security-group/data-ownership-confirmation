const {
  concat,
  keccak256,
  solidityPacked,
  toUtf8Bytes,
} = require("ethers");

const LOAD_FACTOR = 0.75;

function nextPowerOfTwo(value) {
  let result = 1;
  while (result < value) result *= 2;
  return result;
}

function categoryCapacity(totalItems, typeCount) {
  // The paper cases divide evenly. This mirrors the Go implementation's
  // int(float64(categorySize) / loadFactor), minimum 16, then power-of-two
  // allocation in hashtable.NewHashTable.
  const categorySize = Math.ceil(totalItems / typeCount);
  const requested = Math.max(16, Math.trunc(categorySize / LOAD_FACTOR));
  return nextPowerOfTwo(requested);
}

function deterministicHash(label) {
  return keccak256(toUtf8Bytes(label));
}

function buildLayeredProofFixture(totalItems, typeCount) {
  const capacity = categoryCapacity(totalItems, typeCount);
  const subtreeHeight = Math.log2(capacity);
  const url = "https://TestCompany/images/file_0_1234";
  const dataHash = deterministicHash(`data-${totalItems}-${typeCount}`);
  const leafHash = keccak256(
    solidityPacked(["string", "uint8", "bytes32"], [url, 0, dataHash])
  );

  // The submitted SCArb.verifyOwnershipLayered entry point fixes leafIndex to
  // zero, so a valid public-entry-point fixture must use the leftmost leaf.
  let subtreeRoot = leafHash;
  const subtreeProofPath = [];
  for (let level = 0; level < subtreeHeight; level += 1) {
    const sibling = deterministicHash(
      `sibling-${totalItems}-${typeCount}-${level}`
    );
    subtreeProofPath.push(sibling);
    subtreeRoot = keccak256(concat([subtreeRoot, sibling]));
  }

  const otherSubtreeRoots = [];
  for (let i = 1; i < typeCount; i += 1) {
    otherSubtreeRoots.push(
      deterministicHash(`subtree-${totalItems}-${typeCount}-${i}`)
    );
  }

  const topRoot = keccak256(concat([subtreeRoot, ...otherSubtreeRoots]));
  return {
    totalItems,
    typeCount,
    categoryCapacity: capacity,
    subtreeHeight,
    url,
    dataHash,
    subtreeProofPath,
    otherSubtreeRoots,
    dataTypeIndex: 0,
    leafIndex: 0,
    topRoot,
  };
}

module.exports = {
  LOAD_FACTOR,
  buildLayeredProofFixture,
  categoryCapacity,
  deterministicHash,
};
