// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface IERC1155Receiver {
    function onERC1155Received(address operator, address from, uint256 id, uint256 value, bytes calldata data)
        external
        returns (bytes4);

    function onERC1155BatchReceived(
        address operator,
        address from,
        uint256[] calldata ids,
        uint256[] calldata values,
        bytes calldata data
    ) external returns (bytes4);
}

// An ERC-1155 multi token, https://eips.ethereum.org/EIPS/eip-1155, modelling
// the events emitted by token interactions; it is not meant for production
// use. The given amounts of the given ids are created for the deployer.
contract ERC1155 {
    mapping(address => mapping(uint256 => uint256)) public balanceOf;
    mapping(address => mapping(address => bool)) public isApprovedForAll;

    event TransferSingle(
        address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value
    );
    event TransferBatch(
        address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values
    );
    event ApprovalForAll(address indexed owner, address indexed operator, bool approved);

    constructor(uint256[] memory ids, uint256[] memory values) {
        for (uint256 i = 0; i < ids.length; i++) {
            balanceOf[msg.sender][ids[i]] += values[i];
        }
        emit TransferBatch(msg.sender, address(0), msg.sender, ids, values);
        _checkBatchReceived(address(0), msg.sender, ids, values, "");
    }

    function setApprovalForAll(address operator, bool approved) external {
        isApprovedForAll[msg.sender][operator] = approved;
        emit ApprovalForAll(msg.sender, operator, approved);
    }

    function safeTransferFrom(address from, address to, uint256 id, uint256 value, bytes calldata data) external {
        require(to != address(0), "zero address");
        require(msg.sender == from || isApprovedForAll[from][msg.sender], "not authorized");
        balanceOf[from][id] -= value;
        balanceOf[to][id] += value;
        emit TransferSingle(msg.sender, from, to, id, value);
        _checkReceived(from, to, id, value, data);
    }

    function safeBatchTransferFrom(
        address from,
        address to,
        uint256[] calldata ids,
        uint256[] calldata values,
        bytes calldata data
    ) external {
        require(to != address(0), "zero address");
        require(ids.length == values.length, "length mismatch");
        require(msg.sender == from || isApprovedForAll[from][msg.sender], "not authorized");
        for (uint256 i = 0; i < ids.length; i++) {
            balanceOf[from][ids[i]] -= values[i];
            balanceOf[to][ids[i]] += values[i];
        }
        emit TransferBatch(msg.sender, from, to, ids, values);
        _checkBatchReceived(from, to, ids, values, data);
    }

    function _checkReceived(address from, address to, uint256 id, uint256 value, bytes memory data) internal {
        if (to.code.length > 0) {
            require(
                IERC1155Receiver(to).onERC1155Received(msg.sender, from, id, value, data)
                    == IERC1155Receiver.onERC1155Received.selector,
                "unsafe recipient"
            );
        }
    }

    function _checkBatchReceived(
        address from,
        address to,
        uint256[] memory ids,
        uint256[] memory values,
        bytes memory data
    ) internal {
        if (to.code.length > 0) {
            require(
                IERC1155Receiver(to).onERC1155BatchReceived(msg.sender, from, ids, values, data)
                    == IERC1155Receiver.onERC1155BatchReceived.selector,
                "unsafe recipient"
            );
        }
    }
}
