require("@nomicfoundation/hardhat-ethers");

const { subtask } = require("hardhat/config");
const {
  TASK_COMPILE_SOLIDITY_GET_SOLC_BUILD,
} = require("hardhat/builtin-tasks/task-names");

// Use the npm-pinned solc-js build. This keeps compilation reproducible and
// avoids an implicit compiler download during artifact evaluation.
subtask(TASK_COMPILE_SOLIDITY_GET_SOLC_BUILD).setAction(
  async ({ solcVersion }, hre, runSuper) => {
    if (solcVersion === "0.8.19") {
      return {
        compilerPath: require.resolve("solc/soljson.js"),
        isSolcJs: true,
        version: "0.8.19",
        longVersion: "0.8.19+commit.7dd6d404",
      };
    }
    return runSuper();
  }
);

// Keep these values literal rather than environment-configurable so that the
// gas-reproduction command always compiles the exact same bytecode.
const optimizerEnabled = true;
const optimizerRuns = 200;

module.exports = {
  solidity: {
    version: "0.8.19",
    settings: {
      optimizer: {
        enabled: optimizerEnabled,
        runs: optimizerRuns,
      },
      // Solidity 0.8.19 targets Paris by default. Setting it explicitly avoids
      // future compiler-default drift.
      evmVersion: "paris",
    },
  },
  networks: {
    hardhat: {
      hardfork: "shanghai",
      initialBaseFeePerGas: 0,
    },
  },
  mocha: {
    timeout: 120000,
  },
};
