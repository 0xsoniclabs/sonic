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

// A minimal ERC-1155 multi token, https://eips.ethereum.org/EIPS/eip-1155
// The given amounts of the given ids are created for the deployer. The total
// supply of an id is the sum of the balances of all holders, so that it
// reflects every balance update.
contract ERC1155 {
    mapping(address => mapping(uint256 => uint256)) public balanceOf;
    mapping(address => mapping(address => bool)) public isApprovedForAll;
    address[] holders;
    mapping(address => bool) isHolder;

    event TransferSingle(
        address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value
    );
    event TransferBatch(
        address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values
    );
    event ApprovalForAll(address indexed owner, address indexed operator, bool approved);
    event URI(string value, uint256 indexed id);

    constructor(uint256[] memory ids, uint256[] memory values) {
        require(ids.length == values.length, "length mismatch");
        _addHolder(msg.sender);
        for (uint256 i = 0; i < ids.length; i++) {
            balanceOf[msg.sender][ids[i]] += values[i];
        }
        emit TransferBatch(msg.sender, address(0), msg.sender, ids, values);
        _checkBatchReceived(address(0), msg.sender, ids, values, "");
    }

    function totalSupply(uint256 id) external view returns (uint256 sum) {
        for (uint256 i = 0; i < holders.length; i++) {
            sum += balanceOf[holders[i]][id];
        }
    }

    function supportsInterface(bytes4 interfaceId) external pure returns (bool) {
        return interfaceId == 0x01ffc9a7 || interfaceId == 0xd9b67a26;
    }

    function balanceOfBatch(address[] calldata owners, uint256[] calldata ids)
        external
        view
        returns (uint256[] memory balances)
    {
        require(owners.length == ids.length, "length mismatch");
        balances = new uint256[](owners.length);
        for (uint256 i = 0; i < owners.length; i++) {
            balances[i] = balanceOf[owners[i]][ids[i]];
        }
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
        _addHolder(to);
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
        _addHolder(to);
        emit TransferBatch(msg.sender, from, to, ids, values);
        _checkBatchReceived(from, to, ids, values, data);
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

    function _checkReceived(address from, address to, uint256 id, uint256 value, bytes memory data) internal {
        if (to.code.length > 0) {
            require(
                IERC1155Receiver(to).onERC1155Received(msg.sender, from, id, value, data)
                    == IERC1155Receiver.onERC1155Received.selector,
                "unsafe recipient"
            );
        }
    }

    function _addHolder(address account) internal {
        if (!isHolder[account]) {
            isHolder[account] = true;
            holders.push(account);
        }
    }
}
