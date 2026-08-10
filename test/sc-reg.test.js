const assert = require("node:assert/strict");
const { ethers } = require("hardhat");
const { deployContracts } = require("../scripts/lib/deploy-contracts");

const VALID_PERIOD = 365 * 24 * 60 * 60;
const IDENTITY = "Test Company Ltd.";
const DESCRIPTION = "Deterministic registration test metadata.";
const ROOT_ONE = `0x${"11".repeat(32)}`;
const ROOT_TWO = `0x${"22".repeat(32)}`;

describe("SCReg", function () {
  let owner;
  let other;
  let scReg;

  beforeEach(async function () {
    [owner, other] = await ethers.getSigners();
    ({ scReg } = await deployContracts(ethers));
  });

  it("stores a registration and emits the registration event", async function () {
    const tx = await scReg
      .connect(owner)
      .register(ROOT_ONE, VALID_PERIOD, IDENTITY, DESCRIPTION);
    const receipt = await tx.wait();

    const registration = await scReg.getOwnerRegistration(owner.address);
    assert.equal(registration.currentRootHash, ROOT_ONE);
    assert.equal(registration.version, 1n);
    assert.ok(registration.validUntil > 0n);
    assert.equal(await scReg.isOwnerActive(owner.address), true);
    assert.ok(receipt.logs.length >= 2);
  });

  it("rejects a second registration from the same owner", async function () {
    await scReg.connect(owner).register(ROOT_ONE, VALID_PERIOD, IDENTITY, DESCRIPTION);

    await assert.rejects(
      scReg.connect(owner).register(ROOT_TWO, VALID_PERIOD, IDENTITY, DESCRIPTION),
      /already registered/
    );
  });

  it("updates the root in place and increments the version", async function () {
    await scReg.connect(owner).register(ROOT_ONE, VALID_PERIOD, IDENTITY, DESCRIPTION);
    await (await scReg.connect(owner).update(ROOT_TWO)).wait();

    const registration = await scReg.getOwnerRegistration(owner.address);
    assert.equal(registration.currentRootHash, ROOT_TWO);
    assert.equal(registration.version, 2n);
  });

  it("rejects unchanged roots and updates after expiry", async function () {
    await scReg.connect(owner).register(ROOT_ONE, 1, IDENTITY, DESCRIPTION);

    await assert.rejects(scReg.connect(owner).update(ROOT_ONE), /same root/);

    await ethers.provider.send("evm_increaseTime", [2]);
    await ethers.provider.send("evm_mine");
    await assert.rejects(scReg.connect(owner).update(ROOT_TWO), /expired/);
    assert.equal(await scReg.isOwnerActive(owner.address), false);
  });

  it("keeps registrations independent across owner addresses", async function () {
    await scReg.connect(owner).register(ROOT_ONE, VALID_PERIOD, IDENTITY, DESCRIPTION);
    await scReg.connect(other).register(ROOT_TWO, VALID_PERIOD, IDENTITY, DESCRIPTION);

    assert.equal((await scReg.getOwnerRegistration(owner.address)).currentRootHash, ROOT_ONE);
    assert.equal((await scReg.getOwnerRegistration(other.address)).currentRootHash, ROOT_TWO);
  });
});
