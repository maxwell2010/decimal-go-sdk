# Official node evidence for EVM Checks - 2026-10-29

## Source revision and verification

`git ls-remote https://bitbucket.org/decimalteam/go-smart-node.git refs/heads/master` returned `9e6c6d718d662083c4a524376a66d2c50bd4bc77`, the same revision already pinned by this SDK. Four relevant files were fetched directly from the official Bitbucket source API at that commit and compared byte-for-byte with the local node module snapshot. All four matched. No cached source was edited or executed.

| Official node file | SHA-256 |
|---|---|
| cmd/dscd/stakescan/checks.go | e595fa66193f4ed45b0e4efceaa10eb015ee079acaff94de9dc0b2c7497ae4eb |
| app/redenom/evm.go | 094cd3f1e3e33a930d87bbf325308542ad33f7ffe42dfd5c6bd5d3392fbb8ef8 |
| sdk/tx/checks.go | e81d83748285b878812fe19bc385dcf7568138ac00caa2ce4867a0fa36c19ca3 |
| x/coin/types/check.go | 16e88d6480a1d52035c70a1d7523be8ee72104003965e84721feae885fab960e |

## What the node confirms

[stakescan/checks.go](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/cmd/dscd/stakescan/checks.go#lines-11) describes the namespaced DecimalChecks storage layout: nonce mapping address to uint256 at field0, check mapping at field1 and detail mapping at field2. The status order is None0/Redeemed1/Revoked2; type order is DEL0/Token1. Detail fields reconstructed from storage are typeChecks, amount, dueBlock, token and creator.

Its `CheckHashOf` at lines43-49 explicitly computes `keccak256(checkDetailsHash[32] || signer[20])`, corresponding to `abi.encodePacked(bytes32,address)`. This is the stored check identity. It is not evidence for a redemption signing digest, recipient binding, prefix choice or how checkDetailsHash is initially computed. The reconstruction test uses arbitrary supplied detail hashes; it cannot supply missing signer-message semantics.

[app/redenom/evm.go](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/app/redenom/evm.go#lines-533) documents the DEL backing model: createChecksDEL receives amount multiplied by number of signers and redemption pays msg.sender. Lines552-554 state that DecimalChecks does not self-register in ContractCenter; node migration code therefore falls back to storage reconstruction. This corroborates the earlier registry-discovery limitation; a missing registry slug is not proof the contract does not exist.

Node migration comments reference DecimalChecks.sol, but that Solidity source is not included in this node snapshot. Targeted search of the node source for DecimalChecks, redeemChecks, createChecksDEL, checkDetailsHash and toEthSignedMessageHash did not reveal the EVM check signing preimage or prefix. The official Bitbucket contracts/ directory listing at this commit was also read directly (one page, no next page): center, delegation, delegationNft, nft1155, nft721, nftCenter, token, tokenCenter, validator, contract_test.go and utils.go. It has no Checks source directory. No source change since the previously audited master was found.

## Historical Cosmos checks are a separate protocol

[sdk/tx/checks.go](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/sdk/tx/checks.go#lines-20) derives a passphrase key using SHA256, creates an RLP-based lock, signs an issuer check and Base58-encodes it. Redemption signs the Keccak/RLP receiver-address digest and places the proof in a native MsgRedeemCheck.

[x/coin/types/check.go](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/x/coin/types/check.go#lines-84) defines HashWithoutLock and Hash over string chain ID, denomination, amount, nonce bytes, dueBlock and optional lock. These native RLP formats must not be substituted for EVM Checks ABI/signature formats. The SDK already exposes them separately in its Cosmos module.

## Other official Solidity source check

Official [decimal-contracts](https://bitbucket.org/decimalteam/decimal-contracts/src/018a6c6d00616c3e24ae44a8fac429b2ccabbd12/) master remains `018a6c6d00616c3e24ae44a8fac429b2ccabbd12`. A new shallow read-only clone contains decimalToken.sol, bridgeBsc.sol and bridgeEth sources; no DecimalChecks, Delegation/DelegationNFT or GasCenter Solidity interface was found there. The historical bridge sources are not a bridge-v2 substitute. No missing current deployment ABI was recovered from these two repositories.

Implementation must therefore use the independently reviewed official JS signing format where available, and label that format/version explicitly. The node evidence above corroborates identity/storage/backing only. It does not establish deployed signature acceptance, replay/domain protection or stateful authorization. No transaction was sent, no user secret was loaded, and no ABI was inferred from a same-named unrelated contract.

The migration storage reader describes nonce storage keyed by address, while the pinned/current JS Checks ABI exposes a no-argument nonces() getter. These sources do not establish interchangeable deployment generations. The signing helpers use neither storage interpretation nor an inferred nonce getter.
