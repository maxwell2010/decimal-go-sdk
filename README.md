# Decimal Go SDK

An independent EVM Go module, `github.com/maxwell2010/decimal-go-sdk`, based on pinned official Decimal node, JS SDK and Go SDK sources, with workflow requirements from our Python and C# implementations. Go 1.26.10+ is required. The current public version is `v0.1.0-preview.7`, a preview with no claim of complete deployment parity.

By [MintCandy](https://mintcandy.ru/) and [@Maxwell2019](https://t.me/Maxwell2019). MIT license.

[Русский README](README.ru.md) | [Getting started](docs/GETTING_STARTED.md) | [Examples](examples/README.md) | [Changelog](CHANGELOG.md)

[Russian guide with examples](README.ru.md) · [Contract API](docs/API.md) · [Operation matrix](docs/OPERATION_MATRIX.md) · [Source hashes](docs/sources.json) · [Validation](docs/VALIDATION.md).

The package contains 11 pinned ABI profiles and 300 generated methods: 157 offline transaction builders and 143 typed readers. Administrative functions are included. This is a contract-method count, not a count of user workflows or unique protocol transaction types.

Nine explicitly selected, versioned testnet profiles provide 224 more typed methods: system contracts, Safe/factory/MultiSend and reserve/reserveless token interfaces. Their addresses, observed blocks, code hashes and ABI hashes are recorded in [system deployment sources](docs/deployment-sources.json), [Safe sources](docs/additional-deployment-sources.json) and [token sources](docs/token-deployment-sources.json). These artifacts remain separate from the pinned node ABI and never silently replace defaults. All 524 wrappers are exercised by the ABI integration suite.

The pinned JS index has 93 active write entry points. 55 have at least one related pinned-ABI builder, 35 require an explicit deployment ABI, two use native transfers, and mixed multisend supports DEL while the unsafe generic-Multicall ERC20 spender route is disabled. Related selectors do not establish complete workflow or deployment equivalence.

## Read-only network compatibility

[Compatibility matrix](docs/COMPATIBILITY.md) separates local ABI tests, observed getters, historical Checks events and limited eth_call simulations. It records fixed blocks and raw RPC hashes for both networks. The configured mainnet Checks address needs deployment review; an initialized proxy is not established. No default address was changed.

Use `Client.BlockIdentity` for the RPC-reported block hash; `types.Header.Hash()` may differ on Decimal. `ChecksCreationHistory` and `ChecksCreationReceipt` read existing creation hashes without a wallet. [History API and example](docs/CHECKS_HISTORY.md).

## Features

- Native DEL, memo and DEL multisend.
- ERC20, reservable/reserveless token creation, exact-input/output economy calls, convert, permits and approval planning.
- Delegation, holds, reset, transfer, withdrawal and completion; typed ordinary/held/frozen stake reads and block-pinned batch verification.
- ERC721/1155 collections, minting/reserve/transfer/burn/approval, batch transfers and DelegationNFT.
- Validator controls and contract administration from the pinned ABI.
- Checks from the official JS ABI, with recipient-bound EIP-191 proofs, signer recovery/verification and offline batch construction. [Signing format and example](docs/EVM_CHECKS.md).
- EIP-712 permit and Safe signing, owner-bound/sorted signature packing, explicit deployment-ABI execution and proxy factory construction.
- Bridge and GasCenter builders using a supplied deployment ABI; nonce widths are not inferred from JavaScript number types.
- Context-aware RPC, fees, balances, code, logs, receipts, subscriptions through a WebSocket backend, bounded HTTP API reads and authoritative withdrawal indexing.

Legacy Cosmos protobuf transactions from `go-smart-node/sdk/tx` use another protocol and are outside this EVM module. IPFS uploads and explorer contract verification are also outside its scope. Full network parity and production readiness are not claimed.

The separate [Cosmos module](cosmos/README.md) provides 43 schema-backed message constructors and transaction envelopes. Run its checks separately, or use `pwsh -File tools/check.ps1 -Race -Vulnerabilities` to check both modules. Its own README distinguishes implemented codecs/pipeline from semantic and network acceptance.

## Install and update from one permanent command

Inside an existing Go project:

```sh
go run github.com/maxwell2010/decimal-go-sdk/cmd/decimal-sdk-install@latest
```

This installs or updates **both EVM and Cosmos** to the reviewed published version in [latest.json](latest.json), including previews. The command never changes with the release number. No automatic background updates occur. Use `-print` to inspect the selected go get command without modifying a project, or `-evm-only` / `-cosmos-only` to select one module. New project? Run `go mod init example.com/myapp` first. [Full install/update/rollback guide](docs/INSTALL.md).

[All 260 ABI write-builder examples](docs/TRANSACTIONS.md) | [EVM envelope types 0/1/2](docs/TRANSACTION_ENVELOPES.md) | [All 43 Cosmos message examples](cosmos/TRANSACTIONS.md). These are offline encoding examples, not deployed execution certificates.

## Install a fixed version

Requires Go 1.26.10 or later. In your application's Go module:

```sh
go get github.com/maxwell2010/decimal-go-sdk@v0.1.0-preview.7
```

The independent Cosmos module is optional:

```sh
go get github.com/maxwell2010/decimal-go-sdk/cosmos@v0.1.0-preview.7
```

Go downloads runtime dependencies automatically. GitHub is the release source; the tag pins the preview. Downloadable source ZIP/TAR.GZ files and `SHA256SUMS` are attached to the [release](https://github.com/maxwell2010/decimal-go-sdk/releases/tag/v0.1.0-preview.7). No compiler-generated bindings or external ABI compiler are needed to use the library.

For local development, clone the repository and run:

```sh
go test ./...
go vet ./...
go run ./cmd/decimal-inspect -network testnet
```

The inspection command is read-only and requires no wallet. Mainnet is configured for EVM chain ID 75; testnet uses 202020. `Dial` verifies the RPC chain. System child addresses are resolved using `ResolveSystemContract`, rather than copied between networks.

## Quick start

A complete offline example, requiring no wallet or RPC:

```go
package main

import (
    "fmt"
    decimal "github.com/maxwell2010/decimal-go-sdk"
)

func main() {
    amount, err := decimal.ParseUnits("1.25", 18)
    if err != nil { panic(err) }
    to, err := decimal.ParseAddress("0x1000000000000000000000000000000000000001")
    if err != nil { panic(err) }
    call, err := decimal.NativeTransfer(to, amount, "invoice-123")
    if err != nil { panic(err) }
    fmt.Println(call.To.Hex(), decimal.FormatUnits(call.Value, 18), string(call.Data))
}
```

The address is a fixture. [Runnable examples](examples/README.md) include read-only balance queries and offline ERC20 encoding. [Getting started](docs/GETTING_STARTED.md) covers installation and transaction preparation.

## Offline construction

```go
amount, err := decimal.ParseUnits("1.25", 18)
if err != nil { return err }
to, err := decimal.ParseAddress(recipientAddress)
if err != nil { return err }
call, err := decimal.NativeTransfer(to, amount, "invoice-123")
if err != nil { return err }
```

Use `decimal "github.com/maxwell2010/decimal-go-sdk"` as the import. Values are exact base-unit integers; floating-point amounts are not accepted. `Build*` methods never perform network I/O, sign or broadcast.

## Explicit submission

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
client, err := decimal.Dial(ctx, decimal.Testnet())
if err != nil { return err }
defer client.Close()
signer, err := decimal.NewKeySigner(privateKeyFromSecretStore)
if err != nil { return err }
defer signer.Close()

prepared, err := client.Prepare(ctx, signer.Address(), call, decimal.TxOptions{})
if err != nil { return err }
fmt.Println(prepared.Fee.EstimatedFee, prepared.Fee.ReservedFee)

sent, err := client.Send(ctx, signer, call, decimal.TxOptions{}) // broadcasts
if err != nil { return err }
mined, err := client.WaitReceipt(ctx, sent.Hash, time.Second)
if err != nil { return err }
fmt.Println(mined.ActualFee)
```

The additional imports are `context`, `fmt` and `time`. Secret and recipient variables come from the application. No secret is embedded in the examples. `SignerFromMnemonic` supports BIP39/BIP32 and defaults to `m/44'/60'/0'/0/0`; derive once and reuse the signer. Hardware/external signers can implement `Signer`.

Preparation checks network, code for contract calls, estimate, limits and available DEL. The default gas-limit buffer is 10%; it is reserved capacity, not the amount necessarily charged. Receipts expose actual fees only when `effectiveGasPrice` is available. Failed receipts return `RevertedError`.

`Send` serializes automatic nonces per sender in one Client. Multiple clients/processes must coordinate externally. `Prepare` does not reserve a nonce. `SubmissionUnknownError` preserves the locally computed transaction hash; reconcile it before sending again. No automatic endpoint failover or fee retry is performed after submission. Legacy EIP-155 is the default. Explicit `TxOptions.TxType` 1 selects EIP-2930 access-list transactions; type 2 selects EIP-1559 with `FeeCap`, `TipCap` and optional `BaseFeeEstimate`. Dynamic reserves use the fee cap; without a base-fee estimate the estimated effective fee is unknown. Network support and the global minimum fee are not inferred from `eth_gasPrice`.

## Contracts and deployment variants

Reference routing recognizes explicitly selected versioned token profiles. Both
support their matching ERC20 operations; `mintTokenReserveless` is restricted to
the reserveless profile. For `TestnetReserveToken20261029`, reference operation
`updateTokenIdentity` explicitly builds `updateIdentity(string)`. This selector
adaptation does not apply to pinned `Token` or arbitrary deployment ABIs.
`BuildReferenceOperationSignature` applies the same operation/profile whitelist.

```go
address, err := client.ResolveSystemContract(ctx, "delegation")
if err != nil { return err }
staking, err := client.Delegation(address)
if err != nil { return err }
call, err := staking.BuildDelegateDEL(amountWei, validator)
if err != nil { return err }
```

The generated contract API uses typed inputs, including nested tuples, arrays and overloads. Result fields `Value0`, `Value1`, etc. follow ABI output order. Decode events with `DecodeLog`. The full method-to-signature mapping is in `docs/API.md`.

For versions missing from the pinned node ABI, use `ContractWithABI(address, verifiedABI)`. `BuildReferenceOperation` adapts a reference JS operation name using ABI-order arguments in raw units. Missing or ambiguous methods fail before RPC/signing. `BuildNamed` resolves an unambiguous overload; `Build` accepts an exact Solidity signature. `API.DeploymentABI` can read an official compact-contract response but never silently replaces a pinned ABI.

`PlanERC20Action` keeps approve and action separate. Wait for approval and re-estimate the action. Tokens requiring allowance reset need an explicit additional zero-approval step. `CheckedPermit` checks the actual nonce/domain separator. Callers choose domain version and deadline. Do not grant generic Multicall token allowance.

Additional helpers now cover zero-reset approval plans, full-uint256 weighted Safe policies, mixed EOA/contract signatures and explicitly selected modern/legacy ERC-1271 verification. See [SAFE_WORKFLOWS.md](docs/SAFE_WORKFLOWS.md). The opt-in sender-restricted multisend adapter is a project extension derived from our C# contract evidence, not an official Decimal ABI/default; it requires explicit chain/address/runtime-hash binding. Only read-only runtime checks were performed in this Go work.

`LoadProxyDeploymentContract` and `LoadBeaconDeploymentContract` resolve standard ERC-1967 implementations at a requested block while retaining the proxy as the target. Bound calls recheck code/slot evidence before preparation; reads are also pinned and checked. `BindProxyABI`/`BindBeaconABI` accept caller-reviewed artifacts when a public API is unavailable or misannotated. They do not prove that an arbitrary API ABI matches bytecode or prevent an upgrade after the last check. [ERC-1967](https://eips.ethereum.org/EIPS/eip-1967) defines the storage slots and beacon getter.

For Safe, prefer `PackSafeSignatures`, which binds each signature to an expected owner and transaction hash. The lower-level `PackSafeEOASignatures` only recovers/sorts signers; it does not establish membership. On-chain owner weights, threshold and nonce remain deployment checks. Contract signatures require a separate deployment-specific encoder.

`FrozenWithdrawalsBatch` verifies up to 500 known indexes in one block-pinned call; `FrozenWithdrawals` supports deployments with only the single getter. Index enumeration requires an authoritative indexer. `API.WithdrawalIndex` rejects unavailable, incomplete or stale results. Maturity is separate from completion.

## Reproducible source import

`tools/generate.py --node PATH --js PATH --go PATH` verifies pinned Git revisions, normalizes plain/wrapped/string-encoded ABIs, copies ABI interfaces with SHA-256 provenance and generates the typed API. `tools/compare.py --js PATH --go PATH` produces the source-linked operation matrix. Run `go fmt ./...` afterwards, and run it separately inside the Cosmos module. Avoid recursively formatting ignored dependency/tool caches. Generator input identifiers are validated before producing Go source; `python -B -m unittest discover -s tools -p test_generate.py` tests those boundaries.

No upstream Go or JS implementation is copied or executed by these tools. Node MIT attribution is preserved in `THIRD_PARTY_NODE_LICENSE`. The JS package declares ISC without a standalone LICENSE in the snapshot; its interface-artifact attribution is recorded separately for publication review.

[Node RPC block identity and hash selector limits](docs/NODE_RPC_IDENTITY.md).

[Local EVM execution, RPC resilience and signing-profile mismatch](docs/LOCAL_EVM_TESTS.md).

[Stable release criteria](docs/STABLE_RELEASE.md).
