# Local EVM execution and RPC resilience

[Русский](LOCAL_EVM_TESTS.ru.md) | [Validation](VALIDATION.md) | [Compatibility](COMPATIBILITY.md)

## Scope

The local EVM suite executes actual contract creation/runtime bytecode through the in-process go-ethereum EVM and its local state database. It does not replace contract execution with a mock Backend. It requires no node process, public RPC, wallet files or funded account. Calls use deterministic fixture addresses and local balances. No blockchain transaction is signed, mined or sent by this suite.

Run from the SDK source root:

```sh
go test -run '^TestLocalEVM' .
```

The suite exercises the available historical token and Checks fixtures. Its intended assertions cover resulting state, events and expected failures rather than only successful ABI encoding. The recorded results for the release belong in [VALIDATION.md](VALIDATION.md); this page describes scope and reproduction, not a separate claim that every scenario passed.

## Checks signing mismatch found by execution

**The pinned JS source signing profile and the bundled Checks bytecode are not interchangeable.** The bundled artifact has a 6258-byte runtime. Its local ECRECOVER trace receives `accounts.TextHash(checkHash[:])`, independent of the redemption caller in the tested paths. The SDK helper follows the reviewed official JS source profile, which binds the check hash and redeemer before applying the personal-sign prefix. That helper's proof is rejected by this artifact with `InvalidSignature` for both tested callers; rejected calls preserve balances. The mismatch is preserved explicitly in `TestLocalEVMOfficialChecksCreationAndSigningProfileMismatch`, not hidden as successful redemption coverage.

The SDK signing helper remains unchanged. A separate positive path for this historical fixture uses a synthetic signature over the directly observed digest only inside tests. It is not a new public signing API or a recommendation for public deployments. No receiver binding is promised for the historical artifact. The observed public Checks implementation runtime has 6318 bytes, a different artifact; size and this local result do not establish a deployed signing generation or live redemption compatibility. Deployment-specific verification remains required.

## Genuine official fixtures

[Fixture manifest](../testdata/evm/manifest.json) records source pins and SHA-256 values. Sources and artifact files are checked in, so normal Go tests do not invoke a Solidity compiler or download anything beyond Go dependencies.

- Historical `NewToken`: exact `decimalToken.sol` from official `decimalteam/decimal-contracts` commit `018a6c6d00616c3e24ae44a8fac429b2ccabbd12`, compiled with solc `0.8.17+commit.8df45f5f.Emscripten.clang`, optimizer disabled, London EVM target. Available methods include transfer, approve, transferFrom, owner-controlled mint, burn and burnFrom. The constructor creates the specified initial emission scaled by 10^18. This historical contract is not evidence of current deployed Decimal token implementation equivalence; it has no permit route.
- Checks: unmodified ABI/creation/runtime artifact from official `dsc-js-sdk` commit `6790d35e2decb0cbb06a9149a9c476c834f99223`, `src/decimalevm/abi/Checks.json`. Execution of this implementation fixture does not establish initialized proxy state, network deployment identity, funding or production redemption acceptance.

[Compiler package](../testdata/evm/compiler-package.json), [lockfile](../testdata/evm/compiler-package-lock.json) and [rebuild tool](../tools/rebuild_local_evm_fixture.py) support explicit maintainer reproduction with the exact solc package. Read the tool usage before invoking it; it requires the local compiler path. Normal tests use the reviewed checked-in bytecode. Preserve the included source license notices.

Safe and permit execution are not certified by these fixtures. A missing official executable fixture remains an unexecuted case, not a hand-written contract presented as deployment proof. The suite also does not prove Decimal consensus, fee accounting, RPC mining, proxy initialization, cross-chain settlement or stateful compatibility for every SDK ABI profile.

## HTTP JSON-RPC failure tests

```sh
go test -run '^TestRPCResilience' .
```

These tests use the real SDK HTTP transport against loopback `httptest` servers. Covered fault classes include HTTP 429/503, connection closure, malformed JSON, JSON-RPC errors, pending-request deadlines and inconsistent receipt identity/shape. They check redaction and request counts, including no automatic resubmission after an uncertain broadcast response. Signed transaction bytes in these tests are synthetic fixtures delivered only to the local HTTP server.

A loopback transport test is separate from in-process EVM execution and public node acceptance. Reorganization/state consistency observations remain limited as explained in [Node RPC identity](NODE_RPC_IDENTITY.md).

## Bounded decoder fuzzing

Each command starts a separate bounded campaign with two workers:

```sh
go test -run '^$' -fuzz '^FuzzChecksEventDecoding$' -fuzztime 7s -parallel 2 .
go test -run '^$' -fuzz '^FuzzChecksReceiptDecoding$' -fuzztime 7s -parallel 2 .
go test -run '^$' -fuzz '^FuzzChecksSignatureRecovery$' -fuzztime 7s -parallel 2 .
```

Seeds come from the saved official Checks receipt/events and independently executed official-JS signature vectors. Invariants check deterministic decoding/recovery, input preservation, output-buffer ownership and independent ECDSA verification. Input lengths are capped. The initial 2026-10-29 campaigns passed with 214141 event, 327 receipt and 65124 signature executions; counts depend on the machine and fuzz corpus and are not coverage or acceptance percentages. No crash was found in these bounded runs. Future runs may differ and do not prove absence of all defects.
