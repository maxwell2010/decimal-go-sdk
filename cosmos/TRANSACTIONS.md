# Cosmos transaction message examples

Run from the `cosmos` module:

```sh
go run ./examples/messages
go run ./examples/messages -json
go test ./examples/messages
```

The executable uses all 43 named constructors in `examples/messages/fixtures.go`.
It fills them with deterministic public protobuf wire fixtures from
`schema/validation-node-corpus.json`, runs `ValidateBasic`, extracts `GetSigners`,
packs each message into `Any` and encodes its own independent `TxBody`.
These exact fixture inputs were accepted by the pinned official node validators
in the local differential reference. Nothing is signed or submitted.

The examples below are **stateless validation fixtures**, not ready business
transactions. Replace fixture addresses, amounts, IDs, metadata, authorities,
proofs and nested messages with your real inputs before signing. In particular,
`MsgCreateTransaction.content` contains an opaque placeholder Any whose
`type_url` is `fixture`; it is accepted by upstream stateless validation but must
be replaced with a real supported message for execution. Placeholder check/proof fields
preserve documented upstream validation omissions, not usable proofs/signatures.

Payloads are protobuf text format. Bytes use protobuf text escaping; integer
amount strings are raw base units, never floating point. For native DEL, 1 DEL
is 1000000000000000000 base units. Custom assets use their own on-chain units.
Decimal rate strings and IDs are not token amounts; retain their schema types.
The exact wire bytes and encoded bodies are available with `-json`.
The text dump omits protobuf default values; it is not a complete field schema.
Use `m.Descriptor().Fields()` to inspect the pinned descriptor.

To adapt an example, call its named constructor, decode its public fixture with
`proto.Unmarshal`, then change fields on the returned dynamic message. For example:

```go
// Imports: encoding/hex, google.golang.org/protobuf/proto,
// cosmos "github.com/maxwell2010/decimal-go-sdk/cosmos".
m := cosmos.NewMsgSendCoin()
wire, err := hex.DecodeString("0a29643031717971737a716770717971737a716770717971737a716770717971737a71677073647765356b1229643031716770717971737a716770717971737a716770717971737a716770717971737a706667756c711a080a03616263120131")
if err != nil { panic(err) }
if err = proto.Unmarshal(wire, m); err != nil { panic(err) }
// Replace fixture fields with your reviewed inputs here, then validate again.
if err = cosmos.ValidateBasic(m); err != nil { panic(err) }
signers, err := cosmos.GetSigners(m)
if err != nil { panic(err) }
packed, err := cosmos.Pack(m)
if err != nil { panic(err) }
body, err := cosmos.EncodeBody([]proto.Message{m}, "offline example", 0)
if err != nil { panic(err) }
_, _, _ = signers, packed, body // unsigned, no submission
```

Admission is source-based, not a live network guarantee: only 8 types have a
standalone native route in the pinned node ante. The other 35 are historical or
schema support. Role checks, balances, nonces, state, oracle membership, bridge
proofs and settlement remain node responsibilities. A passing fixture is not
permission to send an administrative operation. `GetSigners` returns address
bytes; applications must bind those addresses to authorized signer keys.

Signing is separate: obtain chain ID, account number, sequence, fee/gas and the
required public keys; use `SignDirectTx` with explicit signer order and a local
signer callback. The example deliberately stops at unsigned `TxBody` encoding.

## All 43 messages

### 1. `NewMsgCancelUpgrade`

Type URL: `/cosmos.upgrade.v1beta1.MsgCancelUpgrade`. pinned native route.

```textproto
authority: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
```

### 2. `NewMsgSoftwareUpgrade`

Type URL: `/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade`. pinned native route.

```textproto
authority: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
plan: {
  name: "fixture"
  height: 1
  info: "fixture"
}
```

### 3. `NewMsgBurnCoin`

Type URL: `/decimal.coin.v1.MsgBurnCoin`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
coin: {
  denom: "abc"
  amount: "1"
}
```

### 4. `NewMsgBuyCoin`

Type URL: `/decimal.coin.v1.MsgBuyCoin`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
coin_to_buy: {
  denom: "abc"
  amount: "1"
}
max_coin_to_sell: {
  denom: "xyz"
  amount: "1"
}
```

### 5. `NewMsgCreateCoin`

Type URL: `/decimal.coin.v1.MsgCreateCoin`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
denom: "abc"
title: "fixture"
crr: 50
initial_volume: "1000000000000000000"
initial_reserve: "1000000000000000000"
limit_volume: "100000000000000000000"
identity: "fixture"
min_volume: "1000000000000000000"
```

### 6. `NewMsgMultiSendCoin`

Type URL: `/decimal.coin.v1.MsgMultiSendCoin`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
sends: {
  recipient: "d01qgpqyqszqgpqyqszqgpqyqszqgpqyqszpfgulq"
  coin: {
    denom: "abc"
    amount: "1"
  }
}
```

### 7. `NewMsgRedeemCheck`

Type URL: `/decimal.coin.v1.MsgRedeemCheck`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
check: "fixture"
proof: "fixture"
```

### 8. `NewMsgSellAllCoin`

Type URL: `/decimal.coin.v1.MsgSellAllCoin`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
coin_denom_to_sell: "abc"
min_coin_to_buy: {
  denom: "xyz"
  amount: "1"
}
```

### 9. `NewMsgSellCoin`

Type URL: `/decimal.coin.v1.MsgSellCoin`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
coin_to_sell: {
  denom: "abc"
  amount: "1"
}
min_coin_to_buy: {
  denom: "xyz"
  amount: "1"
}
```

### 10. `NewMsgSendCoin`

Type URL: `/decimal.coin.v1.MsgSendCoin`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
recipient: "d01qgpqyqszqgpqyqszqgpqyqszqgpqyqszpfgulq"
coin: {
  denom: "abc"
  amount: "1"
}
```

### 11. `NewMsgUpdateCoin`

Type URL: `/decimal.coin.v1.MsgUpdateCoin`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
denom: "abc"
limit_volume: "100000000000000000000"
identity: "fixture"
min_volume: "1000000000000000000"
```

### 12. `NewMsgUpdateCoinPrices`

Type URL: `/decimal.fee.v1.MsgUpdateCoinPrices`. pinned native route.

```textproto
oracle: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
prices: {
  denom: "abc"
  quote: "abc"
  price: "1"
  updated_at: {
    seconds: 1
  }
}
```

### 13. `NewMsgReturnLegacy`

Type URL: `/decimal.legacy.v1.MsgReturnLegacy`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
public_key: "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
```

### 14. `NewMsgCreateTransaction`

Type URL: `/decimal.multisig.v1.MsgCreateTransaction`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
wallet: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
content: {
  type_url: "fixture"
  value: "\x00"
}
```

### 15. `NewMsgCreateWallet`

Type URL: `/decimal.multisig.v1.MsgCreateWallet`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
owners: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
owners: "d01qgpqyqszqgpqyqszqgpqyqszqgpqyqszpfgulq"
weights: 1
weights: 2
threshold: 1
```

### 16. `NewMsgSignTransaction`

Type URL: `/decimal.multisig.v1.MsgSignTransaction`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
id: "d0mstx1qyxd5h6p"
```

### 17. `NewMsgBurnToken`

Type URL: `/decimal.nft.v1.MsgBurnToken`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
token_id: "fixture"
sub_token_ids: 1
sub_token_ids: 2
```

### 18. `NewMsgMintToken`

Type URL: `/decimal.nft.v1.MsgMintToken`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
denom: "abc"
token_id: "fixture"
token_uri: "fixture"
recipient: "d01qgpqyqszqgpqyqszqgpqyqszqgpqyqszpfgulq"
quantity: 1
reserve: {
  denom: "abc"
  amount: "1"
}
```

### 19. `NewMsgSendToken`

Type URL: `/decimal.nft.v1.MsgSendToken`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
token_id: "fixture"
sub_token_ids: 1
sub_token_ids: 2
recipient: "d01qgpqyqszqgpqyqszqgpqyqszqgpqyqszpfgulq"
```

### 20. `NewMsgUpdateReserve`

Type URL: `/decimal.nft.v1.MsgUpdateReserve`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
token_id: "fixture"
sub_token_ids: 1
sub_token_ids: 2
reserve: {
  denom: "abc"
  amount: "1"
}
```

### 21. `NewMsgUpdateToken`

Type URL: `/decimal.nft.v1.MsgUpdateToken`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
token_id: "fixture"
token_uri: "fixture"
```

### 22. `NewMsgActivateChain`

Type URL: `/decimal.swap.v1.MsgActivateChain`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
id: 1
name: "fixture"
```

### 23. `NewMsgDeactivateChain`

Type URL: `/decimal.swap.v1.MsgDeactivateChain`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
id: 1
```

### 24. `NewMsgInitializeSwap`

Type URL: `/decimal.swap.v1.MsgInitializeSwap`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
recipient: "d01qgpqyqszqgpqyqszqgpqyqszqgpqyqszpfgulq"
amount: "1"
token_symbol: "fixture"
transaction_number: "1"
from_chain: 1
dest_chain: 2
```

### 25. `NewMsgRedeemSwap`

Type URL: `/decimal.swap.v1.MsgRedeemSwap`. historical/schema-only.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
from: "fixture"
recipient: "d01qgpqyqszqgpqyqszqgpqyqszqgpqyqszpfgulq"
amount: "1"
token_symbol: "fixture"
transaction_number: "1"
from_chain: 1
dest_chain: 2
v: 1
r: "0000000000000000000000000000000000000000000000000000000000000000"
s: "0000000000000000000000000000000000000000000000000000000000000000"
```

### 26. `NewMsgCancelRedelegation`

Type URL: `/decimal.validator.v1.MsgCancelRedelegation`. historical/schema-only.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator_src: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
validator_dst: "d0valoper1qgpqyqszqgpqyqszqgpqyqszqgpqyqszpamucs"
creation_height: 1
coin: {
  denom: "abc"
  amount: "1"
}
```

### 27. `NewMsgCancelRedelegationNFT`

Type URL: `/decimal.validator.v1.MsgCancelRedelegationNFT`. historical/schema-only.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator_src: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
validator_dst: "d0valoper1qgpqyqszqgpqyqszqgpqyqszqgpqyqszpamucs"
creation_height: 1
token_id: "fixture"
sub_token_ids: 1
sub_token_ids: 2
```

### 28. `NewMsgCancelUndelegation`

Type URL: `/decimal.validator.v1.MsgCancelUndelegation`. historical/schema-only.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
creation_height: 1
coin: {
  denom: "abc"
  amount: "1"
}
```

### 29. `NewMsgCancelUndelegationNFT`

Type URL: `/decimal.validator.v1.MsgCancelUndelegationNFT`. historical/schema-only.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
creation_height: 1
token_id: "fixture"
sub_token_ids: 1
sub_token_ids: 2
```

### 30. `NewMsgCreateValidator`

Type URL: `/decimal.validator.v1.MsgCreateValidator`. historical/schema-only.

```textproto
operator_address: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
reward_address: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
consensus_pubkey: {
  type_url: "fixture"
  value: "\x00"
}
description: {
  moniker: "fixture"
  identity: "fixture"
  website: "fixture"
  security_contact: "fixture"
  details: "fixture"
}
commission: "1"
stake: {
  denom: "abc"
  amount: "1"
}
```

### 31. `NewMsgDelegate`

Type URL: `/decimal.validator.v1.MsgDelegate`. historical/schema-only.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
coin: {
  denom: "abc"
  amount: "1"
}
```

### 32. `NewMsgDelegateNFT`

Type URL: `/decimal.validator.v1.MsgDelegateNFT`. historical/schema-only.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
token_id: "fixture"
sub_token_ids: 1
sub_token_ids: 2
```

### 33. `NewMsgEditValidator`

Type URL: `/decimal.validator.v1.MsgEditValidator`. historical/schema-only.

```textproto
operator_address: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
reward_address: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
description: {
  moniker: "fixture"
  identity: "fixture"
  website: "fixture"
  security_contact: "fixture"
  details: "fixture"
}
```

### 34. `NewMsgFreezeChain`

Type URL: `/decimal.validator.v1.MsgFreezeChain`. pinned native route.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
reason: "fixture"
```

### 35. `NewMsgHaltChain`

Type URL: `/decimal.validator.v1.MsgHaltChain`. pinned native route.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
height: 1
reason: "fixture"
```

### 36. `NewMsgRedelegate`

Type URL: `/decimal.validator.v1.MsgRedelegate`. historical/schema-only.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator_src: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
validator_dst: "d0valoper1qgpqyqszqgpqyqszqgpqyqszqgpqyqszpamucs"
coin: {
  denom: "abc"
  amount: "1"
}
```

### 37. `NewMsgRedelegateNFT`

Type URL: `/decimal.validator.v1.MsgRedelegateNFT`. historical/schema-only.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator_src: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
validator_dst: "d0valoper1qgpqyqszqgpqyqszqgpqyqszqgpqyqszpamucs"
token_id: "fixture"
sub_token_ids: 1
sub_token_ids: 2
```

### 38. `NewMsgResumeChain`

Type URL: `/decimal.validator.v1.MsgResumeChain`. pinned native route.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
```

### 39. `NewMsgSetOffline`

Type URL: `/decimal.validator.v1.MsgSetOffline`. historical/schema-only.

```textproto
validator: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
```

### 40. `NewMsgSetOnline`

Type URL: `/decimal.validator.v1.MsgSetOnline`. pinned native route.

```textproto
validator: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
```

### 41. `NewMsgUndelegate`

Type URL: `/decimal.validator.v1.MsgUndelegate`. historical/schema-only.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
coin: {
  denom: "abc"
  amount: "1"
}
```

### 42. `NewMsgUndelegateNFT`

Type URL: `/decimal.validator.v1.MsgUndelegateNFT`. historical/schema-only.

```textproto
delegator: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
validator: "d0valoper1qyqszqgpqyqszqgpqyqszqgpqyqszqgpseaenx"
token_id: "fixture"
sub_token_ids: 1
sub_token_ids: 2
```

### 43. `NewMsgUnfreezeChain`

Type URL: `/decimal.validator.v1.MsgUnfreezeChain`. pinned native route.

```textproto
sender: "d01qyqszqgpqyqszqgpqyqszqgpqyqszqgpsdwe5k"
```

