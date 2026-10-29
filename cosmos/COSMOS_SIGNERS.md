# Decimal Cosmos address and signer extraction

This package implements offline extraction for all **43 request message types** in the pinned descriptor inventory: 41 Decimal messages and two Cosmos upgrade messages. It does not authorize or broadcast them. The explicit field/HRP/source-line registry is [schema/signer-sources.json](schema/signer-sources.json).

## Address rules

The node's [cmd/config/config.go](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/cmd/config/config.go#lines-30) sets `d0` for accounts, `d0valoper` for validator operators and `d0valcons` for consensus addresses. These prefixes are different identities at the string-validation boundary, although converting operator bytes to account bytes must not hash or otherwise change the payload.

`EncodeAddress`/`DecodeAddress` reproduce the pinned Cosmos SDK's [Bech32 conversion](https://github.com/cosmos/cosmos-sdk/blob/v0.46.16/types/bech32/bech32.go) and [default address verifier](https://github.com/cosmos/cosmos-sdk/blob/v0.46.16/types/address.go#L164): nonempty addresses of at most 255 bytes, Bech32 checksum and strict bit-padding conversion, decoder input limit 1023. A deployment with a custom address verifier may impose additional restrictions. For the usual Ethermint EVM address the payload is 20 bytes; generic Cosmos conversion does not pretend every address format is necessarily 20 bytes.

`HexToAccountAddress` requires the literal `0x` prefix, parses hex and preserves bytes, matching the node's [GetDecimalAddressFromHex](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/types/utils.go#lines-83). `ValidatorToAccount` performs the `sdk.AccAddress(sdk.ValAddress(...))` conversion used in validator GetSigners methods. `ReencodeAddress` is the explicit arbitrary-source-HRP conversion corresponding to the node's `GetDecimalAddressFromBech32`. Strict GetSigners extraction does not use that permissive prefix conversion.

No legacy public-key address hashing is inferred. In particular, re-encoding an existing address with HRP `dx` does not derive the legacy address of an Ethermint public key. These APIs do not claim that equivalence.

## Source-backed signer rules

| Message family | Required signer source | Expected HRP |
|---|---|---|
| Coin: create/update/send/multisend/buy/sell/sell-all/burn/redeem-check | `sender` | `d0` |
| Legacy return | `sender` | `d0` |
| Multisig: create wallet/create transaction/sign transaction | `sender` | `d0` |
| NFT: mint/update/update reserve/send/burn | `sender` | `d0` |
| Swap: initialize/redeem/activate/deactivate | `sender` | `d0` |
| Validator create/edit | `operator_address` | `d0valoper` |
| Validator online/offline | `validator` | `d0valoper` |
| Delegate/redelegate/undelegate and cancellation, coin/NFT variants | `delegator` | `d0` |
| Emergency halt/resume/freeze/unfreeze | `sender` | `d0` |
| Fee coin-price update | `oracle` | `d0` |
| Software upgrade/cancel | `authority` | `d0` |

Decimal rules come from each `x/*/types/msg*.go` GetSigners implementation and `x/fee/types/tx.go` at node commit `9e6c6d7`. The node's go.mod pins Cosmos SDK v0.46.16; the two upgrade rules are its [GetSigners implementations](https://github.com/cosmos/cosmos-sdk/blob/v0.46.16/x/upgrade/types/msgs.go#L38).

For valid fields, `GetSigners` returns exactly one account-address byte payload per supported message. Unknown message types return `ErrUnsupportedSigners`, rather than guessing from a field named sender. Nil messages, invalid addresses, wrong HRPs or descriptor shapes return errors. This intentionally differs from upstream invalid-input branches that sometimes return nil addresses or panic; callers must not sign invalid input.

`TransactionSigners(messages, feePayer)` preserves first-seen message signer order, removes repeated address bytes, then appends a specified fee payer if not already present. This follows [Cosmos SDK Tx.GetSigners](https://github.com/cosmos/cosmos-sdk/blob/v0.46.16/types/tx/types.go). It returns addresses, not AuthInfo public keys. An application must bind each required account to an authorized public key and preserve this order. Fee granter is not automatically another signer.

## Limits of extraction

An extracted sender is not proof of sufficient balance, validator operator authority, multisig membership/threshold, oracle authorization, governance/module authority, chain activation or active transaction routing. Those checks belong to upstream ValidateBasic/keepers/ante handlers and the deployed state. In particular, signer support for 43 codecs does not make all 43 native transaction routes active.

Tests exercise all 43 source rules against their pinned protobuf descriptors, strict operator/account HRPs, checksum/padding/length failures, hex and HRP round trips, unknown-message rejection, signer order and fee-payer deduplication. No real wallet, signature or network submission is involved.
