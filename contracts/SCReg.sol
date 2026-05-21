// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;


contract SCReg {
    struct OwnerRegistration {
        bytes32 currentRootHash;
        uint64 validUntil;
        uint32 version;
    }

    mapping(address => OwnerRegistration) private registrationsByOwner;

    event OwnerRegistered(
        address indexed owner,
        bytes32 indexed rootHash,
        uint32 version,
        uint64 validUntil
    );

    event OwnerUpdated(
        address indexed owner,
        bytes32 indexed oldRootHash,
        bytes32 indexed newRootHash,
        uint32 version
    );

    event OwnerMetadata(
        address indexed owner,
        string identity,
        string description
    );

    function register(
        bytes32 rootHash,
        uint32 validPeriod,
        string calldata identity,
        string calldata description
    ) external {
        require(rootHash != bytes32(0), "invalid root");
        require(validPeriod > 0, "invalid period");

        OwnerRegistration storage reg = registrationsByOwner[msg.sender];
        require(reg.version == 0, "already registered");

        uint64 validUntil = uint64(block.timestamp) + uint64(validPeriod);
        reg.currentRootHash = rootHash;
        reg.version = 1;
        reg.validUntil = validUntil;

        emit OwnerRegistered(msg.sender, rootHash, 1, validUntil);
        
        if (bytes(identity).length > 0 || bytes(description).length > 0) {
            emit OwnerMetadata(msg.sender, identity, description);
        }
    }


    function update(bytes32 newRootHash) external {
        require(newRootHash != bytes32(0), "invalid root");

        OwnerRegistration storage reg = registrationsByOwner[msg.sender];
        require(reg.version > 0, "not registered");
        require(block.timestamp <= reg.validUntil, "expired");
        bytes32 oldRoot = reg.currentRootHash;
        require(oldRoot != newRootHash, "same root");

        reg.currentRootHash = newRootHash;
        reg.version = reg.version + 1;

        emit OwnerUpdated(msg.sender, oldRoot, newRootHash, reg.version);
    }


    function getOwnerRegistration(address owner) external view returns (
        bytes32 currentRootHash,
        uint32 version,
        uint64 validUntil
    ) {
        OwnerRegistration memory r = registrationsByOwner[owner];
        return (r.currentRootHash, r.version, r.validUntil);
    }

    function isOwnerActive(address owner) external view returns (bool) {
        OwnerRegistration memory r = registrationsByOwner[owner];
        return r.version > 0 && block.timestamp <= r.validUntil;
    }

    function getCurrentRoot(address owner) external view returns (bytes32) {
        return registrationsByOwner[owner].currentRootHash;
    }
}
