# Official node RPC identity and selector evidence - 2026-10-29

## Pinned source chain

The SDK pins official [go-smart-node 9e6c6d7](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/). Its `go.mod` requires `github.com/decimalteam/ethermint v0.22.13`. The downloaded module origin identifies [Decimal Ethermint commit dfd93f4](https://github.com/decimalteam/ethermint/tree/dfd93f4bd53e6d9bc13fd8004350dd2b54340d9b), tag v0.22.13. Each of the four files below was fetched read-only from that exact official GitHub commit and compared byte-for-byte with the local module snapshot. No cached source was changed or executed.

| Source file | SHA-256 |
|---|---|
| rpc/types/block.go | d3c5b349046900467896d182fc9093ff1ef0e2347a3f0a8cf4e2f044ea0f8649 |
| rpc/types/utils.go | 50a81dbedd7b2a5ade95d2251b309fdff3764f887b77b83c2916f484762ba2d1 |
| rpc/backend/blocks.go | c33deb3fad6f6670066fc047bc1f526c924735dac2dcec1e2f058094eff534e1 |
| rpc/namespaces/ethereum/eth/api.go | 09599d50af0bfe731d2b0775eeda376b6a918ba9865a359f29510b6955ae0244 |

This establishes the pinned source behavior, not which binary every public RPC currently runs.

## Why the original block hash matters

[FormatBlock in rpc/types/utils.go](https://github.com/decimalteam/ethermint/blob/dfd93f4bd53e6d9bc13fd8004350dd2b54340d9b/rpc/types/utils.go#L113) receives a Tendermint header and assigns its `Hash()` to JSON `hash`. In the same file, `EthHeaderFromTendermint` constructs a different Ethereum-shaped header containing translated fields. Its Ethereum RLP hash is not the source Tendermint header hash. [GetBlockByNumber and RPCBlockFromTendermintBlock](https://github.com/decimalteam/ethermint/blob/dfd93f4bd53e6d9bc13fd8004350dd2b54340d9b/rpc/backend/blocks.go#L66) use the JSON formatter.

This explains why a Go `types.Header.Hash()` recomputation is unsuitable as Decimal RPC identity. Use `Client.BlockIdentity` and compare its original RPC hash with receipt/log block hashes. A trusted RPC response is still not an independently verified consensus or state proof.

## Hash-shaped selectors are not full EIP1898 guarantees

[BlockNumberOrHash](https://github.com/decimalteam/ethermint/blob/dfd93f4bd53e6d9bc13fd8004350dd2b54340d9b/rpc/types/block.go#L138) supports `blockNumber` or `blockHash` and rejects both together. The struct has no `requireCanonical` field. Its ordinary JSON unmarshal does not reject unknown fields, so sending `requireCanonical: true` does not establish enforcement by this source.

[BlockNumberFromTendermint](https://github.com/decimalteam/ethermint/blob/dfd93f4bd53e6d9bc13fd8004350dd2b54340d9b/rpc/backend/blocks.go#L215) resolves a hash through Tendermint block lookup and returns its height. [PublicAPI.Call](https://github.com/decimalteam/ethermint/blob/dfd93f4bd53e6d9bc13fd8004350dd2b54340d9b/rpc/namespaces/ethereum/eth/api.go#L280) then passes that number to `DoCall`. It also ignores the accepted state-override argument. Therefore accepting a hash-shaped JSON selector must not be advertised as full canonical-selector enforcement or working state overrides. In this pinned parser, string `finalized` maps to latest; this is not an independent finalized-height signal.

The current inspectors explicitly use numbered snapshots plus before/after original-hash checks. This remains a consistency observation with the documented race/trust limitations. No stronger EIP1898 claim or automatic fallback has been introduced.

## Checks provenance does not supply a replacement mainnet address

The previously verified official [node Checks evidence](CHECKS_NODE_EVIDENCE.md) remains applicable. `app/redenom/evm.go` lines 635 onward locate Checks by scanning node-local EVM storage for reconstructed check details. This is an internal upgrade routine, not a public bounded RPC discovery interface or deployment-initialization certificate. The source reviewed here does not justify changing the SDK's configured mainnet Checks address or claiming its initialization. The separate [deployment observations](DEPLOYMENT_COMPATIBILITY.md) retain that unresolved result.

No ABI, nonce interpretation, replacement deployment address or transaction acceptance was inferred. All operations for this investigation were source reads; no wallet, signer, state override or transaction submission was used.
