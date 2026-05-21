// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;


interface ISCReg {
    function isOwnerActive(address owner) external view returns (bool);
    function getCurrentRoot(address owner) external view returns (bytes32);
}

contract SCArb {
    ISCReg public scRegContract;

    constructor(address _scRegContract) {
        require(_scRegContract != address(0), "SCReg contract address cannot be zero");
        scRegContract = ISCReg(_scRegContract);
    }

    
    function verifyOwnership(
        address owner,
        bytes32 rootHash,
        bytes32 dataHash,
        string memory url,
        bytes32[] memory proofPath
    ) external view returns (bool success) {
        require(scRegContract.isOwnerActive(owner), "Owner not active");
        bytes32 currentRoot = scRegContract.getCurrentRoot(owner);
        require(currentRoot == rootHash, "Root mismatch");

        bool verifyResult = _verifyLayeredProof(rootHash, dataHash, url, proofPath, new bytes32[](0), 0, 0);
        return verifyResult;
    }
    
    
    
    
    function verifyOwnershipLayered(
        address owner,
        bytes32 rootHash,
        bytes32 dataHash,
        string memory url,
        bytes32[] memory subtreeProofPath,
        bytes32[] memory otherSubtreeRoots,
        uint256 dataTypeIndex
    ) external view returns (bool) {
        require(scRegContract.isOwnerActive(owner), "Owner not active");
        bytes32 currentRoot = scRegContract.getCurrentRoot(owner);
        require(currentRoot == rootHash, "Root mismatch");

        bool verifyResult = _verifyLayeredProof(
            rootHash, dataHash, url, subtreeProofPath, otherSubtreeRoots, dataTypeIndex, 0
        );
        return verifyResult;
    }
    
    
    function _verifyLayeredProof(
        bytes32 rootHash,
        bytes32 dataHash,
        string memory url,
        bytes32[] memory subtreeProofPath,
        bytes32[] memory otherSubtreeRoots,
        uint256 dataTypeIndex,
        uint256 leafIndex
    ) internal pure returns (bool) {
        
        bytes32 leafHash = keccak256(abi.encodePacked(url, uint8(0), dataHash));
        bytes32 subtreeRoot = leafHash;
        
        uint256 currentIndex = leafIndex;
        for (uint256 i = 0; i < subtreeProofPath.length; i++) {
            bytes32 siblingHash = subtreeProofPath[i];
            if (currentIndex % 2 == 0) {
                subtreeRoot = keccak256(abi.encodePacked(subtreeRoot, siblingHash));
            } else {
                subtreeRoot = keccak256(abi.encodePacked(siblingHash, subtreeRoot));
            }
            currentIndex = currentIndex / 2;
        }
        
        bytes32 computedTopRoot;
        
        if (otherSubtreeRoots.length == 0) {
            computedTopRoot = subtreeRoot;
        } else if (otherSubtreeRoots.length == 1) {
            if (dataTypeIndex == 0) {
                computedTopRoot = keccak256(abi.encodePacked(subtreeRoot, otherSubtreeRoots[0]));
            } else {
                computedTopRoot = keccak256(abi.encodePacked(otherSubtreeRoots[0], subtreeRoot));
            }
        } else {
            uint256 totalSubtreeCount = otherSubtreeRoots.length + 1;
            
            bytes32[] memory allRoots = new bytes32[](totalSubtreeCount);
            
            uint256 otherIndex = 0;
            for (uint256 i = 0; i < totalSubtreeCount; i++) {
                if (i == dataTypeIndex) {
                    allRoots[i] = subtreeRoot;
                } else {
                    allRoots[i] = otherSubtreeRoots[otherIndex];
                    otherIndex++;
                }
            }
            
            computedTopRoot = keccak256(abi.encodePacked(allRoots));
        }
        
        return computedTopRoot == rootHash;
    }


    
    function verifyLayeredProofOnly(
        bytes32 rootHash,
        bytes32 dataHash,
        string memory url,
        bytes32[] memory subtreeProofPath,
        bytes32[] memory otherSubtreeRoots,
        uint256 dataTypeIndex,
        uint256 leafIndex
    ) external pure returns (bool) {
        return _verifyLayeredProof(rootHash, dataHash, url, subtreeProofPath, otherSubtreeRoots, dataTypeIndex, leafIndex);
    }

    
    
    
    function computeLeafHash(
        string memory url,
        uint8 state,
        bytes32 dataHash
    ) external pure returns (bytes32) {
        return keccak256(abi.encodePacked(url, state, dataHash));
    }
    
    
    function computeParentHash(bytes32 leftChild, bytes32 rightChild) external pure returns (bytes32) {
        if (leftChild < rightChild) {
            return keccak256(abi.encodePacked(leftChild, rightChild));
        } else {
            return keccak256(abi.encodePacked(rightChild, leftChild));
        }
    }
    
    
    function updateSCRegContract(address _scRegContract) external {
        require(_scRegContract != address(0), "SCReg contract address cannot be zero");
        scRegContract = ISCReg(_scRegContract);
    }
}


