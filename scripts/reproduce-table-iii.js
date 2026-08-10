const fs = require("fs");
const path = require("path");
const hre = require("hardhat");
const { deployContracts } = require("./lib/deploy-contracts");
const { buildLayeredProofFixture } = require("./lib/gas-fixtures");

const GAS_PRICE_GWEI = 1.61;
const ETH_USD = 3158.53;
const REGISTER_VALID_PERIOD = 365 * 24 * 60 * 60;
const REGISTER_IDENTITY = "Test Company Ltd.";
const REGISTER_DESCRIPTION =
  "Synthetic dataset metadata for deterministic gas benchmark (v1).";
const UPDATE_ROOT = `0x${"22".repeat(32)}`;

const VERIFY_CASES = [
  [10000, 10],
  [100000, 10],
  [1000000, 10],
  [2000000, 10],
  [5000000, 10],
  [5000000, 5],
  [5000000, 20],
  [5000000, 50],
];

function costFromGas(gas) {
  const costEth = Number(gas) * GAS_PRICE_GWEI * 1e-9;
  return { costEth, costUsd: costEth * ETH_USD };
}

function measurement(name, gas) {
  const measuredGas = Number(gas);
  return {
    operation: name,
    measuredGas,
    ...costFromGas(measuredGas),
  };
}

function renderMarkdown(rows) {
  const lines = [
    "# Table III gas reproduction",
    "",
    "| Operation | Gas | Cost (ETH) | Cost (USD) |",
    "|---|---:|---:|---:|",
  ];
  for (const row of rows) {
    lines.push(
      `| ${row.operation} | ${row.measuredGas} | ${row.costEth.toFixed(6)} | ${row.costUsd.toFixed(2)} |`
    );
  }
  lines.push("");
  return `${lines.join("\n")}\n`;
}

async function sendViewTransaction(signer, contract, functionName, args) {
  const data = contract.interface.encodeFunctionData(functionName, args);
  const tx = await signer.sendTransaction({
    to: await contract.getAddress(),
    data,
  });
  return tx.wait();
}

async function main() {
  const [owner, verifier] = await hre.ethers.getSigners();
  const { scReg, scArb, receipts } = await deployContracts(hre.ethers);
  const rows = [
    measurement("SCReg Deploy", receipts.scRegDeploy.gasUsed),
    measurement("SCArb Deploy", receipts.scArbDeploy.gasUsed),
  ];

  const firstFixture = buildLayeredProofFixture(...VERIFY_CASES[0]);
  const registerTx = await scReg
    .connect(owner)
    .register(
      firstFixture.topRoot,
      REGISTER_VALID_PERIOD,
      REGISTER_IDENTITY,
      REGISTER_DESCRIPTION
    );
  const registerReceipt = await registerTx.wait();
  rows.push(measurement("Register", registerReceipt.gasUsed));

  const updateTx = await scReg.connect(owner).update(UPDATE_ROOT);
  const updateReceipt = await updateTx.wait();
  rows.push(measurement("Update", updateReceipt.gasUsed));

  let currentRoot = UPDATE_ROOT;
  for (const [totalItems, typeCount] of VERIFY_CASES) {
    const fixture = buildLayeredProofFixture(totalItems, typeCount);
    if (fixture.topRoot !== currentRoot) {
      const tx = await scReg.connect(owner).update(fixture.topRoot);
      await tx.wait();
      currentRoot = fixture.topRoot;
    }

    const args = [
      owner.address,
      fixture.topRoot,
      fixture.dataHash,
      fixture.url,
      fixture.subtreeProofPath,
      fixture.otherSubtreeRoots,
      fixture.dataTypeIndex,
    ];
    const isValid = await scArb.verifyOwnershipLayered.staticCall(...args);
    if (!isValid) {
      throw new Error(`fixture failed verification for N=${totalItems}, m=${typeCount}`);
    }
    const receipt = await sendViewTransaction(
      verifier,
      scArb,
      "verifyOwnershipLayered",
      args
    );
    const name = `Verify N=${totalItems},m=${typeCount}`;
    rows.push({
      ...measurement(name, receipt.gasUsed),
      totalItems,
      typeCount,
      categoryCapacity: fixture.categoryCapacity,
      subtreeHeight: fixture.subtreeHeight,
    });
  }

  const buildInfo = await hre.artifacts.getBuildInfo(
    "contracts/SCReg.sol:SCReg"
  );
  const report = {
    environment: {
      hardhat: require("hardhat/package.json").version,
      ethers: require("ethers").version,
      solc: buildInfo.solcLongVersion,
      optimizer: buildInfo.input.settings.optimizer,
      evmVersion: buildInfo.input.settings.evmVersion,
      hardfork: hre.network.config.hardfork,
      gasPriceGwei: GAS_PRICE_GWEI,
      ethUsd: ETH_USD,
    },
    benchmarkInputs: {
      registerValidPeriod: REGISTER_VALID_PERIOD,
      registerIdentity: REGISTER_IDENTITY,
      registerDescription: REGISTER_DESCRIPTION,
      updateRoot: UPDATE_ROOT,
      verificationEntryPoint: "SCArb.verifyOwnershipLayered",
      loadFactor: 0.75,
      leafIndex: 0,
    },
    rows,
  };

  const outputDir = path.join(process.cwd(), "reproduction");
  fs.mkdirSync(outputDir, { recursive: true });
  const outputPath = path.join(outputDir, "table-iii-gas.json");
  fs.writeFileSync(outputPath, `${JSON.stringify(report, null, 2)}\n`);
  const markdownPath = path.join(outputDir, "table-iii-gas.md");
  fs.writeFileSync(markdownPath, renderMarkdown(rows));

  console.table(
    rows.map((row) => ({
      operation: row.operation,
      gas: row.measuredGas,
      "cost (ETH)": row.costEth.toFixed(6),
      "cost (USD)": row.costUsd.toFixed(2),
    }))
  );
  console.log(`Detailed report: ${outputPath}`);
  console.log(`Markdown table: ${markdownPath}`);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
