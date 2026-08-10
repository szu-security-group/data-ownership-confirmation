const hre = require("hardhat");
const { deployContracts } = require("./lib/deploy-contracts");

async function main() {
  const { scReg, scArb, receipts } = await deployContracts(hre.ethers);

  console.log(`SCReg address: ${await scReg.getAddress()}`);
  console.log(`SCReg deployment gas: ${receipts.scRegDeploy.gasUsed}`);
  console.log(`SCArb address: ${await scArb.getAddress()}`);
  console.log(`SCArb deployment gas: ${receipts.scArbDeploy.gasUsed}`);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
