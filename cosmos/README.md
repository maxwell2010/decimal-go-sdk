# Decimal Cosmos offline codecs

This independent module implements the Cosmos schema, signing, fee-convergence and query/transport surface identified in `../docs/LEGACY_SCOPE.md`. It does not claim complete Cosmos SDK business/role validation or funded-wallet acceptance.

It includes 43 named request descriptors: 41 Decimal custom request messages plus Cosmos upgrade/cancel-upgrade. `schema/inventory.json` is the exact list. Generated `NewMsg*` constructors create schema-typed `dynamicpb.Message` values; payload fields can be set using protobuf reflection or strict `protojson.Unmarshal`. These are not hand-authored replacement Go payload structs or a substitute for upstream `ValidateBasic`.

`Pack` uses Cosmos Any URLs of the form `/decimal.coin.v1.MsgSendCoin`. The package deterministically marshals and decodes Any, TxBody, AuthInfo, ModeInfo.Single with SIGN_MODE_DIRECT, SignDoc and TxRaw using embedded official descriptors. Ethermint PubKey uses its own type URL and compressed key representation. Account number, sequence and string chain ID are separate signer inputs. TxRaw retains the exact signed body/auth bytes.

`EthermintSigningDigest` follows the actual implementation in [Decimal Ethermint v0.22.13](https://github.com/decimalteam/ethermint/blob/v0.22.13/crypto/ethsecp256k1/ethsecp256k1.go): non-32-byte input is Keccak256-hashed;32-byte input is passed through. The upstream signer returns a65-byte recoverable ECDSA signature. `TransactionHash` is SHA256 over signed TxRaw bytes.

[All 43 transaction examples](TRANSACTIONS.md) | [Все примеры по-русски](TRANSACTIONS.ru.md) | [Install/update both modules](../docs/INSTALL.md)

## Installation

Requires Go 1.26.10 or later:

```sh
go get github.com/maxwell2010/decimal-go-sdk/cosmos@v0.1.0-preview.8
```

Import `github.com/maxwell2010/decimal-go-sdk/cosmos`. The nested module uses its own version tag, `cosmos/v0.1.0-preview.8`, at the same reviewed commit as the parent preview.

## Direct signing and REST

`SignDirectTx(ctx, DirectTx{...})` snapshots body/fee/key inputs before invoking callbacks, constructs ordered AuthInfo, uses each signer's accountNumber and sequence, and verifies every65-byte recoverable signature against its declared compressed public key using go-ethereum crypto. `SignerOrder` is explicit message-required signer order; it must not be replaced by alphabetic address order. Deriving/validating this order against upstream Msg.GetSigners business rules remains the caller's responsibility.

An application importing the parent EVM module can bind an existing reusable signer without copying its key:

```go
callback := func(ctx context.Context, digest [32]byte) ([]byte,error) {
    return keySigner.SignDigest(ctx, common.Hash(digest))
}
// Set DirectSigner.PublicKey to the same signer's compressed public key,
// AccountNumber/Sequence from the Cosmos account query, and Sign:callback.
// The adapter verifies the callback signature against that key.
```

`NewREST(baseURL,httpClient)` provides context-aware `Account`, `Commission`, `Simulate`, explicit single-attempt `Broadcast` and `Transaction` confirmation. Responses are capped at4MiB, redirects are refused. Commission tx_bytes is hex per node query.proto; simulate/broadcast tx_bytes is Base64 per Cosmos JSON. Broadcast uses SYNC: code0 is CheckTx acceptance, not DeliverTx success. Unknown submission outcomes preserve a local hash and are never retried automatically. No invocation is made automatically by offline signing.

## Current node admission is narrower than codec coverage

The pinned node's [app/ante/ante.go](https://bitbucket.org/decimalteam/go-smart-node/src/9e6c6d718d662083c4a524376a66d2c50bd4bc77/app/ante/ante.go#lines-47) selects native Cosmos ante for eight request types: software/cancel upgrade, SetOnline, Halt/Resume/Freeze/UnfreezeChain and UpdateCoinPrices. It otherwise returns ErrUnknownRequest for ordinary native Cosmos transactions. `PinnedNativeRoute` exposes that source classification. The other35 request codecs are useful for history/older deployments but are not advertised as active standalone native workflows on this snapshot. Retained msg_server implementations are not evidence of ante admission. Mixed-message routing and privileged roles require separate controlled-chain acceptance and are not treated as a bypass.

## Rebuild schemas

The source node snapshot must be commit `9e6c6d718d662083c4a524376a66d2c50bd4bc77`. Use Cosmos SDK v0.46.16's proto directory for dependencies not vendored by the node:

```sh
go run ./tools -node /path/to/go-smart-node -sdk /path/to/cosmos-sdk-v0.46.16
gofmt -w messages_generated.go
go test ./...
go vet ./...
```

The compiler reads `proto/custom/decimal/**/tx.proto`, Cosmos upgrade and tx envelopes and Ethermint ethsecp256k1 keys. It follows imports using official node `proto/third_party`, then Cosmos SDK `proto`; `protocompile` includes standard Google imports. All field numbers, types, oneofs and custom options are retained in the embedded FileDescriptorSet. No upstream implementation is executed. The descriptor set and named constructors are committed so runtime users need no compiler.

## Remaining implementation and tests

Stateful role/coin/Msg business validation remains the target node's responsibility. `ValidateBasic` now ports stateless source predicates for all 43 embedded message schemas, with source hashes and boundary fixtures. See [COSMOS_VALIDATION.md](../docs/COSMOS_VALIDATION.md) for important compatibility quirks and the exact differential-test boundary. Validation is not authorization, ante admission or a security policy. Account/sequence queries, node commission query, simulation, REST and gRPC transports, explicit submission/confirmation and concrete verified signing callbacks are implemented. The 43 zero-value Any round trips verify schema availability/serialization, not valid business transactions. SendCoin/envelope and callback/transport tests additionally check fields, direct mode, exact body snapshots, signer order/key mismatch/cancellation, fee hex encoding, simulation bytes and no-retry ambiguous submissions.

## gRPC transport

`DialGRPC(target, credentials)` creates a lazy TLS client with retries disabled and 4 MiB send/receive bounds. Nil credentials select TLS 1.2 or later; plaintext credentials must be explicitly supplied for a controlled local node. `NewGRPC(conn)` accepts a caller-owned connection, whose retry policy is the caller's responsibility.

`NewQueryRequest(service, method)` returns a schema-typed request from the pinned official descriptors. `GRPC.Query` accepts only official Query services and the explicitly allowed read/simulation methods of `cosmos.tx.v1beta1.Service`. Msg services and BroadcastTx cannot be invoked through Query. `Simulate`, `Transaction` and the separate explicit `Broadcast` convenience methods follow protobuf bytes, SYNC CheckTx versus DeliverTx semantics and local transaction-hash reconciliation. Unknown submission outcomes are not automatically retried. No transport is invoked by codecs or signing.

The embedded descriptors now include all custom Decimal query schemas, Cosmos account/bank queries and the Cosmos transaction service, compiled from the same checked node/SDK sources. In-process gRPC wire tests cover simulation encoding, query/write separation, explicit submission, unavailable-server ambiguity and cancellation. These tests do not connect to a real node or send real transactions.

`ConvergeFee` implements bounded signed-size fee convergence with explicit quote/parameter snapshot callbacks, maximum fee, cycle and cancellation checks. See [COSMOS_FEES.md](../docs/COSMOS_FEES.md). `GetSigners` supports all 43 pinned message schemas; `TransactionSigners` preserves first occurrence and fee-payer ordering, and address helpers validate Decimal Bech32 HRPs. See [COSMOS_SIGNERS.md](COSMOS_SIGNERS.md). Applications still bind each required address to its authorized signing key and validate roles through the target chain.

Historical `IssueCheck` and `CreateRedeemCheck` now implement the exact node RLP field order, Base58 encoding, lock and receiver-proof hashes with signer callbacks and declared issuer-key verification. Inputs use copied passphrase bytes and temporary private scalar/hash clearing; caller-owned inputs are not modified. Parsing rejects malformed/noncanonical RLP/Base58, invalid signatures and excessive sizes; redemption additionally checks expected chain, due-block and passphrase. These are historical helpers: current ante admission does not permit standalone MsgRedeemCheck, and no helper broadcasts.

`schema/reference-tx.json` was produced by actually executing `tools/reference`, an isolated pinned Cosmos SDK0.46.16/Ethermint0.22.13 reference module using public deterministic fixtures. `TestSignedTxRawMatchesExecutedOfficialReference` compares body, AuthInfo, SignDoc, ECDSA signature, public key and complete TxRaw bytes. Run `go run .` inside `tools/reference` to reproduce the vector; its older dependency graph is separate from the production module and is used only for offline compatibility testing.

Compiler provenance is recorded in `schema/sources.json`: node commit, each source-file SHA256, Cosmos SDK module checksum and descriptor SHA256. Compilation rejects dirty node proto inputs, untracked node inputs and Cosmos SDK source bytes that differ from the verified v0.46.16 module archive. Runtime compilation remains unnecessary.

No actual transaction has been broadcast. The package is nested as a separate Go module, so run its tests explicitly; `go test ./...` in the parent EVM module does not traverse module boundaries. Node MIT attribution is included in `THIRD_PARTY_NODE_LICENSE`; see also `LICENSE`. Protobuf dependencies retain their upstream licensing.

`schema/validation-node-reference.json` records an actually executed pinned node/SDK reference for ValidateBasic: 129 public wire fixtures across all 43 request types, with zero mismatches, panics or decode failures. See [the isolated reference instructions](tools/node-validation-reference/README.md) and its provenance JSON for checksum verification and reproduction.
