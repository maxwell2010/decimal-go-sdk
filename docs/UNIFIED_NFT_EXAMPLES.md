# Unified NFT transaction examples

Available from `v0.1.0-preview.11`. Select the actual unified proxy explicitly with `client.UnifiedNFTDelegation(unifiedProxy)`. Every call below returns `(decimal.Call, error)` and only builds calldata. Check each error before using its result. No example submits an approval, creates a signature or sends a transaction.

Addresses `validator`, `newValidator` and `collection` are deployment inputs. `tokenID`, `amount`, `holdTimestamp` and `deadline` are `*big.Int` values in the contract's raw units. `v`, `r`, `s` are an already supplied permit proof; these examples do not generate one. ABI presence does not establish that a given public deployment admits the operation.

| Operation | Offline builder |
|---|---|
| ERC721 delegation | `unified.BuildDelegateNFT(validator, collection, tokenID)` |
| ERC721 held delegation | `unified.BuildDelegateNFTHold(validator, collection, tokenID, holdTimestamp)` |
| ERC721 delegation with supplied permit | `unified.BuildDelegateNFTByPermit(validator, collection, tokenID, deadline, v, r, s)` |
| ERC1155 delegation | `unified.BuildDelegateNFT1155(validator, collection, tokenID, amount)` |
| ERC1155 held delegation | `unified.BuildDelegateNFT1155Hold(validator, collection, tokenID, amount, holdTimestamp)` |
| ERC721 withdrawal | `unified.BuildWithdrawNFT(validator, collection, tokenID)` |
| ERC721 held withdrawal | `unified.BuildWithdrawNFTHold(validator, collection, tokenID, holdTimestamp)` |
| ERC1155 withdrawal | `unified.BuildWithdrawNFT1155(validator, collection, tokenID)` |
| ERC1155 held withdrawal | `unified.BuildWithdrawNFT1155Hold(validator, collection, tokenID, holdTimestamp)` |
| ERC721 validator transfer | `unified.BuildTransferNFT(validator, collection, tokenID, newValidator)` |
| ERC721 held validator transfer | `unified.BuildTransferNFTHold(validator, collection, tokenID, holdTimestamp, newValidator)` |
| ERC1155 validator transfer | `unified.BuildTransferNFT1155(validator, collection, tokenID, newValidator)` |
| ERC1155 held validator transfer | `unified.BuildTransferNFT1155Hold(validator, collection, tokenID, holdTimestamp, newValidator)` |

The official ABI has no corresponding ERC1155 permit or held-permit variants. Withdrawals/transfers have exactly the arguments shown; no unsupported partial-amount argument is invented.

For separate approval review:

```go
approval721, err := unified.BuildERC721Approval(collection, tokenID)
if err != nil {
    return err
}
approval1155, err := unified.BuildERC1155ApprovalForAll(collection)
if err != nil {
    return err
}
fmt.Println(approval721.To, approval1155.To, unified.ApprovalOperator())
```

ERC721 approval is specific to the token. ERC1155 approval grants access across the collection and is intentionally named `ForAll`. The operator is the selected unified action destination. Merely constructing either call changes no permission.

Historical NFT positions stay on their original `client.DelegationNFT(legacyProxy)` route. Do not globally replace old addresses or use a new-route exit for an old position. Known retired mainnet deposit calls fail offline; reads and historical exits remain available. See [compatibility and constraints](DELEGATION_UPGRADE.md), [Russian guide](DELEGATION_UPGRADE.ru.md), and [all pinned ABI examples](../examples/README.md).
