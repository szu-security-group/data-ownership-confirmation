const assert = require("node:assert/strict");
const { ethers } = require("hardhat");
const { deployContracts } = require("../scripts/lib/deploy-contracts");
const { buildLayeredProofFixture } = require("../scripts/lib/gas-fixtures");

const VALID_PERIOD = 365 * 24 * 60 * 60;
const IDENTITY = "Test Company Ltd.";
const DESCRIPTION = "Deterministic proof-verification test metadata.";

function proofArgs(owner, fixture) {
  return [
    owner.address,
    fixture.topRoot,
    fixture.dataHash,
    fixture.url,
    fixture.subtreeProofPath,
    fixture.otherSubtreeRoots,
    fixture.dataTypeIndex,
  ];
}

describe("SCArb", function () {
  let owner;
  let scReg;
  let scArb;
  let fixture;

  beforeEach(async function () {
    [owner] = await ethers.getSigners();
    ({ scReg, scArb } = await deployContracts(ethers));
    fixture = buildLayeredProofFixture(100000, 10);
    await scReg
      .connect(owner)
      .register(fixture.topRoot, VALID_PERIOD, IDENTITY, DESCRIPTION);
  });

  it("accepts a valid layered membership proof", async function () {
    const verified = await scArb.verifyOwnershipLayered.staticCall(
      ...proofArgs(owner, fixture)
    );
    assert.equal(verified, true);
  });

  it("rejects proofs with a changed data hash, URL, or Merkle sibling", async function () {
    const alteredHashArgs = proofArgs(owner, fixture);
    alteredHashArgs[2] = `0x${"ff".repeat(32)}`;
    assert.equal(
      await scArb.verifyOwnershipLayered.staticCall(...alteredHashArgs),
      false
    );

    const alteredURLArgs = proofArgs(owner, fixture);
    alteredURLArgs[3] = `${fixture.url}-tampered`;
    assert.equal(
      await scArb.verifyOwnershipLayered.staticCall(...alteredURLArgs),
      false
    );

    const alteredSiblingArgs = proofArgs(owner, fixture);
    alteredSiblingArgs[4] = [...fixture.subtreeProofPath];
    alteredSiblingArgs[4][0] = `0x${"00".repeat(31)}01`;
    assert.equal(
      await scArb.verifyOwnershipLayered.staticCall(...alteredSiblingArgs),
      false
    );
  });

  it("rejects a proof once the registered root has changed", async function () {
    await (await scReg.connect(owner).update(`0x${"22".repeat(32)}`)).wait();

    await assert.rejects(
      scArb.verifyOwnershipLayered.staticCall(...proofArgs(owner, fixture)),
      /Root mismatch/
    );
  });
});
