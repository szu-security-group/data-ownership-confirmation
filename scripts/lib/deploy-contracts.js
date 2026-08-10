async function deployContracts(ethers) {
  const SCReg = await ethers.getContractFactory("SCReg");
  const scReg = await SCReg.deploy();
  const scRegDeployment = await scReg.deploymentTransaction().wait();

  const SCArb = await ethers.getContractFactory("SCArb");
  const scArb = await SCArb.deploy(await scReg.getAddress());
  const scArbDeployment = await scArb.deploymentTransaction().wait();

  return {
    scReg,
    scArb,
    receipts: {
      scRegDeploy: scRegDeployment,
      scArbDeploy: scArbDeployment,
    },
  };
}

module.exports = { deployContracts };
