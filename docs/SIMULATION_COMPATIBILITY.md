# Read-only write-call simulations - 2026-10-29

No key was loaded, no message or transaction was signed, and nothing was broadcast. These are eth_call executions at explicit snapshots, not receipts or settlement. Public accounts are simulated callers; no control of their keys is asserted. No state override was used.

## Reproduce

From the repository root:

```sh
go run ./cmd/simulation-inspect -network testnet -block 22477428 -out /new/path/testnet.json
go run ./cmd/simulation-inspect -network mainnet -block 34336444 -out /new/path/mainnet.json
```

Output files must not exist. The CLI uses only the configured official public RPC URL, validates chain ID, records the raw RPC block hash before/after all probes and refuses non-allowlisted JSON-RPC methods in its HTTP transport. Every contract read and simulation uses the explicit block number. Timeouts are bounded; recent native-account discovery checks at most 32 public blocks. Reverts and missing preconditions are not success.

## Observed results

| Network/snapshot | Route | Result | Exact boundary |
|---|---|---|---|
| testnet 202020 / 22477428 | Reserve token proxy 0x37748f3e857a71e4334125a935052ce1e603b7b5 | eth_call returned canonical true | creator() had 9714968185366177174320 raw units; transfer to itself of 1 raw unit |
| testnet 202020 / 22477428 | Reserveless proxy 0xc310f25e3429efc5ede11a908682f0a31cfc92ce | eth_call returned canonical true | owner() had 100000000000000000000 raw units; transfer to itself of 1 raw unit |
| mainnet 75 / 34336444 | Native self-transfer 1 wei | eth_call success | Public sender recovered from transaction 0x8d20d23028daeb99ebf19a375a9ac3f0d8889f2287c73bf9296025313a290b01; balance 8568026345597857083565 wei, code empty at snapshot |
| testnet 202020 / 22477428 | Native self-transfer | Not tested | No funded EOA candidate found in the bounded 32-block scan; this is not evidence of no funded accounts |
| mainnet | Token writes | Not tested | Versioned testnet ABI not applied to mainnet |
| both | Checks creation/redemption | Not tested | No issued check/proof supplied, no real signing, nonce-generation mismatch not resolved by assumption |
| both | Bridge, NFT, staking, privileged operations | Not tested | Required proof/state/role or exact deployment interface not established by these probes |

Token calls use BindBeaconABI with the reviewed versioned JSON, standard beacon storage/getter binding, and an implementation runtime hash matching token-deployment-sources.json. Reserve uses creator(); reserveless uses owner(), as present in their actual interfaces. Their positive balances are read independently at the same snapshot. The complete target/from/value/calldata/result/code hashes and input provenance are in [testnet evidence](compatibility/simulations-testnet-2026-10-29.json) and [mainnet evidence](compatibility/simulations-mainnet-2026-10-29.json).

Both snapshots remained consistent according to the raw node hash. Testnet hash: 0xfb67691493eda26389cac1a122c8a6396192d97b153857d5ed3c87a10cf22d0c. Mainnet hash: 0xd71b1667291098376553306e483d817232ef821e5e648ec4d7391cf79d871ab7.

## Block hash caution

During verification, go-ethereum types.Header.Hash() recomputed a different value from the node's eth_getBlockByNumber.hash at these Decimal snapshots. The CLI therefore preserves the raw RPC hash and uses it for the before/after consistency guard. It does not relabel a locally recomputed Ethereum header hash as Decimal canonical block identity. A regression test covers raw hash preservation. Other agents/tools must use the same raw identity when joining observations.

## Limits

Self-transfer proves only those positive-value execution paths at those snapshots. It does not prove transfer to another recipient, native gas-payment/nonce admission, allowance, permit, minting, authorization of an application-held key, expiry, bridge settlement or future execution. Identity checks and eth_call are non-atomic across a chain reorganization; the final raw-hash comparison detects an observed change and makes Consistent false, rather than certifying the result. A consistent numeric-block snapshot is not a block-hash RPC execution guarantee.

Offline tests prove that sending/signing/mining/batch RPC requests never reach the transport and that permitted request payloads are unchanged. Tests do not contact a node.
