# Validation - 2026-10-29

Environment: Windows amd64, Go 1.26.10. No user wallet, mnemonic or private key was loaded. Signing tests use deterministic test keys/vectors. No real network transaction was signed or broadcast. The user explicitly selected continuing without transaction submission.

| Check | Result | Boundary |
|---|---|---|
| `go test ./...` in both modules | Passed | EVM/CLI: 664 passing test/example/subtest events, 73 top-level entries. Cosmos: 178 events, 30 top-level entries. These are test runner counts, not transaction types. |
| `go vet ./...` | Passed | Static Go checks. |
| `go build ./...` | Passed | SDK and read-only inspection command compile. |
| `go mod verify` | All modules verified | Final dependency graph uses btcd's split v2 modules, not the locally modified cached v0.24.2 tree encountered during development. |
| `govulncheck@v1.9.0 ./...` | Zero reachable vulnerabilities, zero affected imported packages | One module-level advisory remains for unused/unmaintained `x/crypto/openpgp`. The SDK does not import that package. This is a dated scan, not a guarantee of future absence. |
| Local `go test -race ./...` in both modules | Passed | Portable LLVM-MinGW, process-local CGo/CC/CXX/PATH. No system configuration changes. Linux/Windows CI is configured but remote CI execution has not been observed. |
| Regeneration | Completed | Importer verifies pinned Git HEADs and records ABI SHA-256. Generated files are checked in as source. |

The scan found and resolved a reachable Gorilla WebSocket masking-key advisory by requiring v1.5.3. `x/crypto` was also updated to a patched version; final versions are in `go.mod`/`go.sum`. The project requires patched Go 1.26.10 rather than silently relying on the host's older Go installation.

The final combined command was `pwsh -NoProfile -File tools/check.ps1 -Race -Vulnerabilities -Fuzz`, exit code 0. Portable compiler release: [LLVM-MinGW 20261020](https://github.com/mstorsjo/llvm-mingw/releases/tag/20261020); archive SHA-256 `fd5462b7f5d93b2ff981fcd3f9c7e427e47f3203c64ea91c27bcc444eebd74cd` was verified before extraction. Its files are ignored under `.tools/`. Concurrent nonce/cancellation/external-signer cases also passed 100 stress iterations and 20 race-enabled iterations.

All 300 pinned and 224 versioned deployment ABI wrappers are exercised against ABI wire encoding/decoding, with provenance checks for every artifact. Final statement coverage is 81.4% for the EVM package and 76.2% for Cosmos; this does not prove contract business rules. Bounded fuzz campaigns passed: amounts 667144 executions, signatures 1176475, protobuf decoding 549891, and stateless validation with valid-signer deep paths 483950. No crash was found in those runs. Thirteen Python generator/release tests also pass, including real generated Go compilation, pinned Git blob identity checks and reproducible archive verification.

Cosmos covers all 43 pinned message descriptors, source-backed GetSigners and ValidateBasic, address conversions, direct signing callbacks, REST/gRPC query and simulation, explicit submission interfaces, bounded fee convergence and historical RLP checks. In-process gRPC wire tests check read/write separation, transport errors and cancellation; they do not connect to a real node. A separately executed Cosmos SDK v0.46.16/Ethermint v0.22.13 reference produced `cosmos/schema/reference-tx.json`; body/auth bytes, SignDoc, signature, public key and full TxRaw match exactly. Its payload is a serialization fixture, not a submitted business transaction.

The official pinned node validators were actually executed for all 43 types in an isolated reference module: 129 valid/empty/invalid-signer fixtures, zero mismatches, panics or decode errors. Each seeded dependency ZIP was checked with Go Hash1 against go.sum, then isolated `go mod verify` passed before execution; modified shared-cache archives were rejected. `cosmos/schema/validation-node-reference.json` and reference provenance preserve the result. This proves these tested cases, not equivalence over every possible input. Stateless compatibility preserves upstream quirks documented in `COSMOS_VALIDATION.md`; authorization remains stateful.

Additional EVM tests cover proxy/beacon implementation binding and upgrade detection, versioned-network guards, full uint256 Safe weights, mixed signatures/offsets, explicit caller-aware ERC-1271 adapters, reset allowance plans, and explicit runtime-bound sender-restricted multisend. API-provided ABI attribution is not bytecode verification, and top-level `eth_call` cannot prove nested STATICCALL behavior.

Core tests cover actual JSON-RPC request serialization against an in-process HTTP server, chain mismatch, exact amounts, gas rounding/caps, insufficient balance, absent contract code, signed intent/sender checks, concurrent nonces, unknown submission outcomes, cancellation, failed/successful receipts, actual versus reserved fees, nested ABI tuples, bounded integer coercion, custom errors, event topics, API path/redirect isolation and authoritative/empty/stale indexer semantics. Frozen-stake batching uses one pinned-block `eth_call` per page rather than one call per position.

The Safe hash and ECDSA signature match the deterministic vector recorded in the local C# SDK's `SafeTransactionsTests.cs`, originally derived from the official JS implementation. The expected hash is `0x7fa98d43d883784f0a11456fc76bbf398389bad8ed326e30929f102b8b4fe6fd`. This validates serialization/signing, not deployed Safe owner weights or execution.

## Read-only live RPC checks

These commands used only the official public network URLs in `Config`, without secrets:

```text
go run ./cmd/decimal-inspect -network testnet
network=testnet chain_id=202020 block=22468577

go run ./cmd/decimal-inspect -network mainnet
network=mainnet chain_id=75 block=34327799
```

They confirm RPC connectivity, chain ID and block-number reads at the observed blocks. They do not establish deployed ABI parity, successful gas estimation for every operation or settlement. No token, NFT, validator, bridge, check or Safe write was sent to either network.

## Remaining deployment acceptance

The SDK includes typed builders/readers for all pinned ABI functions plus explicit deployment-ABI workflows. Full deployment parity is not certified. The public import path is `github.com/maxwell2010/decimal-go-sdk`. Before a stable release, run a separately authorized test-wallet acceptance matrix: token creation/economy/permit, NFT generations, delegation/holds/reset/completion, Safe owner weights/threshold/nonce, Checks expiry and bridge settlement. ABI interfaces and static source review cannot replace those tests.

Cosmos codecs and historical RLP checks live in the independent nested module. The pinned native ante admits only eight of its 43 request types as standalone native transactions; the other 35 are historical/schema support, not advertised live routes. Stateful business/role checks and execution/settlement are not certified. The arbitrary-Multicall ERC20 allowance route remains disabled; the explicit sender-restricted adapter is a separate project extension with reviewed C# provenance and no Go defaults. IPFS content upload and explorer contract verification are outside transaction construction.

Authoritative runtime ABI remains unavailable for some deployments. Official implementation API calls for Delegation/DelegationNFT return HTTP 400; Bridge was found via official TheGraph and verified against storage/code, but its API annotation incorrectly returns a Token interface. Checks' functioning `nonces()` call also contradicts its Token API annotation. No false ABI was imported. Bitbucket repository/branch enumeration, official documentation, legacy documented API paths, GraphQL introspection and runtime-addressed IPFS metadata retrieval were investigated; exact artifacts remain unavailable. See `ADDITIONAL_ABI_EVIDENCE.md`, the operation matrix and `ACCEPTANCE.md`. This external gap prevents claiming complete deployed-ABI parity under the official-sources-only constraint.

Read-only verification of the optional project multisend runtime returned the expected keccak256 `0x7784c7a41662304b6f6c183f2169a9b6844a1a2f192021d8f3d1ccbdacfc821e`, 3127 bytes, at testnet block 22474859/address `0xF5D23aD4Ba70d22554d25bcFe9BaE2aF828A6EA4` and mainnet block 34333921/address `0x9b0658e219636215F7f78a88c241ef3BBff447A9`. This is runtime identity evidence only; no multisend transaction was submitted in this work.

The final audit and fixes independent of unavailable ABI are summarized in [FINAL_AUDIT.md](FINAL_AUDIT.md). The preview uses public module paths for the root and independent Cosmos module, with both license notices packaged separately.
