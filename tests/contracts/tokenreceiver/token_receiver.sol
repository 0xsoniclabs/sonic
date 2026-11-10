// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// A recipient of ERC-721 and ERC-1155 safe transfers. It logs every call of
// its hooks, and accepts the tokens only if constructed with accept set to
// true; otherwise it answers with a wrong value.
contract TokenReceiver {
    bool private immutable _accept;

    event ERC721Received(address operator, address from, uint256 tokenId, bytes data);
    event ERC1155Received(address operator, address from, uint256 id, uint256 value, bytes data);
    event ERC1155BatchReceived(address operator, address from, uint256[] ids, uint256[] values, bytes data);

    constructor(bool accept) {
        _accept = accept;
    }

    function onERC721Received(address operator, address from, uint256 tokenId, bytes calldata data)
        external
        returns (bytes4)
    {
        emit ERC721Received(operator, from, tokenId, data);
        return _answer(this.onERC721Received.selector);
    }

    function onERC1155Received(address operator, address from, uint256 id, uint256 value, bytes calldata data)
        external
        returns (bytes4)
    {
        emit ERC1155Received(operator, from, id, value, data);
        return _answer(this.onERC1155Received.selector);
    }

    function onERC1155BatchReceived(
        address operator,
        address from,
        uint256[] calldata ids,
        uint256[] calldata values,
        bytes calldata data
    ) external returns (bytes4) {
        emit ERC1155BatchReceived(operator, from, ids, values, data);
        return _answer(this.onERC1155BatchReceived.selector);
    }

    function _answer(bytes4 selector) private view returns (bytes4) {
        return _accept ? selector : bytes4(0xdeadbeef);
    }
}
