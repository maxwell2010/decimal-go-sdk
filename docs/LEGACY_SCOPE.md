# Legacy Cosmos scope and separate-module plan

Read-only source review: 2026-10-29. No EVM implementation was changed for this review. The word legacy identifies the protobuf transaction family here; it is not evidence that those messages are disabled on a particular deployment.

## Pinned sources

- [Node sdk/tx/types.go](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/sdk/tx/types.go#lines-26) exposes 33 Msg aliases and constructor aliases for its Decimal messages.
- [Node sdk/tx/tx.go](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/sdk/tx/tx.go#lines-28) constructs Cosmos transactions, calculates fees using node ante logic and signs using SIGN_MODE_DIRECT, string chain ID, account number and sequence. These are not Ethereum transaction nonce/EIP-155 signing bytes.
- [Node protobuf sources](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/proto/custom/decimal/) define additional request messages beyond the sdk/tx convenience aliases.
- [Official Go SDK master go.mod](https://bitbucket.org/decimalteam/dsc-go-sdk/src/3ef4a089b6020889e60783c2026df5725fb960e2/go.mod) has module name `dsc-go-sdk` and requires the separate released module `bitbucket.org/decimalteam/dsc-go-sdk v1.5.4`. Its working tree has no top-level `tx/` or `sdk/` implementation directory. Its [decimalevm/api/api.go](https://bitbucket.org/decimalteam/dsc-go-sdk/src/3ef4a089b6020889e60783c2026df5725fb960e2/decimalevm/api/api.go#lines-4) imports that dependency's `tx` package, and uses `dscTx.MsgSendCoin` when decoding a native EVM transfer. It is incorrect to equate absence from the current tree with absence from the dependency API.
- The installed, read-only Go module cache `bitbucket.org/decimalteam/dsc-go-sdk@v1.5.4/tx/types.go` was also inspected. [Released types.go](https://bitbucket.org/decimalteam/dsc-go-sdk/src/v1.5.4/tx/types.go) exposes the 33 baseline aliases below plus `MsgUpdateCoinPrices`, `MsgInitializeSwap`, `MsgRedeemSwap`, and `MsgActivateChain`. The cached file is direct evidence; the tag URL is an upstream navigation link and was not independently fetched in this review.

## Exact baseline excluded from our EVM module

All 33 node sdk/tx aliases below remain outside the current EVM SDK. An EVM operation with a similar name is not a wire-compatible implementation of its Cosmos message.

| Family | Protocol request types | Node source |
|---|---|---|
| Coin, 9 | `MsgCreateCoin`, `MsgUpdateCoin`, `MsgSendCoin`, `MsgMultiSendCoin`, `MsgBuyCoin`, `MsgSellCoin`, `MsgSellAllCoin`, `MsgBurnCoin`, `MsgRedeemCheck` | [coin/v1/tx.proto](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/proto/custom/decimal/coin/v1/tx.proto) |
| NFT, 5 | `MsgMintToken`, `MsgUpdateToken`, `MsgUpdateReserve`, `MsgSendToken`, `MsgBurnToken` | [nft/v1/tx.proto](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/proto/custom/decimal/nft/v1/tx.proto) |
| Multisig, 3 | `MsgCreateWallet`, `MsgCreateTransaction`, `MsgSignTransaction` | [multisig/v1/tx.proto](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/proto/custom/decimal/multisig/v1/tx.proto) |
| Validator, 14 | `MsgCreateValidator`, `MsgEditValidator`, `MsgSetOnline`, `MsgSetOffline`, `MsgDelegate`, `MsgDelegateNFT`, `MsgUndelegate`, `MsgUndelegateNFT`, `MsgRedelegate`, `MsgRedelegateNFT`, `MsgCancelUndelegation`, `MsgCancelUndelegationNFT`, `MsgCancelRedelegation`, `MsgCancelRedelegationNFT` | [validator/v1/tx.proto](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/proto/custom/decimal/validator/v1/tx.proto) |
| Cosmos upgrade, 2 | `MsgSoftwareUpgrade`, `MsgCancelUpgrade` | sdk/tx/types.go aliases Cosmos SDK x/upgrade rather than a Decimal custom protobuf package |

`MultiSendEntry`, `Description`, `Plan`, `FeeParams` and `FeeCalculationOptions` are supporting types, not additional transaction types. Msg*Response records are responses, not requests.

## Additional custom protocol requests outside the aliases

| Family | Requests omitted from node sdk/tx aliases | Released Go SDK v1.5.4 alias evidence |
|---|---|---|
| Fee | `MsgUpdateCoinPrices` | Present |
| Swap | `MsgActivateChain`, `MsgDeactivateChain`, `MsgInitializeSwap`, `MsgRedeemSwap` | Activate/Initialize/Redeem present; Deactivate not aliased in inspected types.go |
| Legacy migration | `MsgReturnLegacy` | Not aliased in inspected types.go |
| Validator chain controls | `MsgHaltChain`, `MsgResumeChain`, `MsgFreezeChain`, `MsgUnfreezeChain` | Not aliased in inspected types.go |

Sources: [fee](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/proto/custom/decimal/fee/v1/tx.proto), [swap](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/proto/custom/decimal/swap/v1/tx.proto), [legacy](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/proto/custom/decimal/legacy/v1/tx.proto), validator source above.

Thus the reviewed custom Decimal protobuf files define **41 request types**, and the sdk/tx surface adds two standard Cosmos upgrade aliases. The union is **43 named request types** for this bounded inventory. This is not a count of every transaction the entire chain accepts: generic Cosmos bank/auth/governance/IBC and Ethermint messages require a separate module-registration inventory. Availability and caller roles of chain-control, fee-oracle and upgrade messages must be checked on the target deployment.

## Other required compatibility surfaces

The node's [sdk/tx/checks.go](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/sdk/tx/checks.go#lines-20) provides `IssueCheck` and `CreateRedeemCheck`. It uses an RLP/Base58 check with a passphrase-derived lock and a Base64 redemption proof. This is distinct from the EVM Checks contract and EIP-712 permits. Issuing a check is an off-chain signing workflow, not another Msg request type.

[sdk/wallet/account.go](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/sdk/wallet/account.go#lines-38) uses Ethermint ethsecp256k1 key semantics, Bech32 addresses, a separate legacy address and signer data containing chain ID/account number/sequence. [cmd/config/config.go](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/cmd/config/config.go#lines-29) specifies `d0` account HRP and validator/consensus variants. Do not replace this with arbitrary Cosmos secp256k1/address hashing or a numeric EVM chain ID.

## Practical separate-module implementation plan

1. Create an independently versioned sibling module, for example `cosmossdk/`, after selecting the public repository/module path. Keep the EVM module dependency graph independent. Pin the target node commit, protobuf schema hashes and relevant Cosmos/Ethermint versions. The reviewed node uses Cosmos SDK v0.46.16 and Decimal Ethermint v0.22.13; copying the released Go SDK's older v0.46.6 dependencies without comparison is not compatibility proof.
2. Import or reproducibly generate the actual gogo protobuf request/query types and register all concrete Msg Any type URLs and crypto interfaces. Preserve field numbers, custom Int/Coin types and constructor/ValidateBasic semantics. Build a catalog for all 41 custom requests plus the two upgrade aliases. Use generic Cosmos requests only after inventorying app registration; do not infer them from standard SDK documentation alone.
3. Implement a wallet/signing adapter for Decimal Ethermint ethsecp256k1, explicit BIP44 derivation and Bech32/legacy conversions. Use exact upstream vectors for address derivation, public keys, SignDoc bytes and signature verification. Do not reuse the EVM transaction signer for raw Cosmos SignDoc bytes without verifying its hashing convention.
4. Implement offline `TxBody`, `AuthInfo`, `SignerInfo`, `Fee`, `SignDoc`, `TxRaw` construction with SIGN_MODE_DIRECT, string chain ID, account number, sequence, memo and multiple Msg support. Explicitly separate build, quote, sign, broadcast and confirmation. Validate signer ordering and multi-signer requirements. Require external sequence coordination across processes and preserve local Tendermint transaction hash on an ambiguous submission.
5. Add context-aware gRPC/REST/Tendermint clients for account number/sequence, parameters, balances, simulation, fee calculation, broadcast and transaction confirmation. Confirm correct chain ID from the target node. Distinguish CheckTx rejection from successful CheckTx followed by failed DeliverTx; a broadcast receipt alone is not success. Handle responses and protobuf Any decoding deterministically.
6. Reproduce fee logic against node ante tests and parameter/oracle snapshots, including signed-byte-length dependence and fee denomination. Bound fee convergence iterations; reject stale/unavailable parameters rather than silently assuming zero fee. Treat upstream's zero-price/zero-fee path as a behavior to verify on the actual deployment, not an unconditional permission to send zero-fee transactions.
7. Add legacy RLP/Base58 check issuance/redemption with public deterministic vectors and malformed/expired/wrong-chain/proof tests. Keep this separate from EVM Checks naming and formats.
8. Acceptance tests: every request must have protobuf encoding/decoding and Any type-URL vectors, invalid-input tests, fee/signature vectors, signer/sequence tests, cancellation and ambiguous-broadcast tests. Compare complete TxRaw bytes against the pinned node SDK and released Go SDK where schemas overlap. Only then run an authorized minimal testnet campaign; privileged governance/oracle/chain-control requests need a dedicated controlled chain and appropriate roles, not a public production validator.

No implementation or live acceptance of this separate Cosmos module is claimed by the current EVM tests. Its dependencies, available roles and active request routes need their own compatibility and vulnerability assessment.

## Follow-up: ante admission restricts retained handlers

Further source tracing found a decisive restriction in [app/ante/ante.go:47-84](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/app/ante/ante.go#lines-47). Although coin/nft/validator/swap module files register real MsgServer handlers, native Cosmos routing is selected only when a transaction contains `MsgSoftwareUpgrade`, `MsgCancelUpgrade`, `MsgSetOnline`, one of `MsgHaltChain/MsgResumeChain/MsgFreezeChain/MsgUnfreezeChain`, or `MsgUpdateCoinPrices`. The other35 inventory types do not receive the normal Cosmos ante route as standalone native transactions in this source snapshot. The default is ErrUnknownRequest. Ethereum extension uses a separate route; Web3/DynamicFee extension branches are commented out.

Do not interpret retained legacy keeper code as successful live transaction support. The allowlist is a source-level admission condition, not authorization: oracle/admin/upgrade roles and Msg validation still apply. The implementation loops over messages and routes when one allowlisted message is present; mixed-message behavior is not endorsed as an admission bypass and requires controlled testing.

A separate `gosdk/cosmos` module now supplies all43 official schema codecs plus verified offline direct-sign callback and explicit REST query/simulation/submission interfaces. Its own README/tests describe the implemented scope and residual requirements; parent EVM test success does not cover that nested module automatically.
