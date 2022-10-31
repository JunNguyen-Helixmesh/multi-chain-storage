// SPDX-License-Identifier: MIT
pragma solidity ^0.8.4;

import "@openzeppelin/contracts/token/ERC1155/ERC1155.sol";
import "@openzeppelin/contracts/token/ERC1155/extensions/ERC1155URIStorage.sol";
import "@openzeppelin/contracts/token/ERC1155/extensions/ERC1155Supply.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/Counters.sol";

// TODO: add admin role, create factory contract, test scripts
contract MCSCollection is ERC1155, ERC1155Supply, ERC1155URIStorage, Ownable {
    using Counters for Counters.Counter;
    Counters.Counter private _tokenIdCounter;

    string public contractURI;

    /// @custom:oz-upgrades-unsafe-allow constructor
    constructor(string memory contractURI_) ERC1155("") {
        contractURI = contractURI_;
    }

    // mints a new token
    function mint(address account, uint256 amount, string memory newUri, bytes memory data)
        public
    {
        _tokenIdCounter.increment();
        uint256 tokenId = _tokenIdCounter.current();

        _mint(account, tokenId, amount, data);
        _setURI(tokenId, newUri);
    }

    // mints more of an existing token
    function mintMore(address account, uint256 id, uint256 amount, bytes memory data)
        public
    {
        require(exists(id), 'Supply: tokenId does not exist');
        _mint(account, id, amount, data);
    }

    // sets URI for existing token if not already set
    function setURI(uint id, string memory newUri) public {
        require(exists(id), 'Supply: tokenId does not exist'); 
        _setURI(id, newUri);
    }

    // overrides
    function uri(uint256 tokenId) public view virtual override(ERC1155, ERC1155URIStorage) returns (string memory) {
        return ERC1155URIStorage.uri(tokenId);
    }

    function _beforeTokenTransfer(
        address operator,
        address from,
        address to,
        uint256[] memory ids,
        uint256[] memory amounts,
        bytes memory data
    )internal virtual override(ERC1155, ERC1155Supply) {
        ERC1155Supply._beforeTokenTransfer(operator, from, to, ids, amounts, data);
    }

    // get the current id count of tokens
    function idCount() public view returns (uint256) {
        return _tokenIdCounter.current();
    }
}