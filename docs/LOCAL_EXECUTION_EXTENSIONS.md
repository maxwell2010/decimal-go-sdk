# Extended local execution coverage

[Русский](LOCAL_EXECUTION_EXTENSIONS.ru.md) | [Earlier local fixtures](LOCAL_EVM_TESTS.md) | [Validation](VALIDATION.md)

These suites add executable evidence with distinct scopes. They do not certify public deployment acceptance, Decimal consensus or settlement. Normal runs require no public RPC, user wallet or funded account. The Safe fixture capture was read-only; all transaction execution below is local.

```sh
go test -run '^(TestLocalEVM|TestLocalChain)' .
```

## Sampled 6318-byte Checks runtime accepts the SDK signing profile

The [current Checks runtime manifest](../testdata/evm/checks-current-manifest.json) records a read-only code sample from official testnet implementation `0xBACA5A733002F5323aBE4c83C91B87212E7A3ca3`, chain 202020, block 22477428, original RPC block hash `0xfb67691493eda26389cac1a122c8a6396192d97b153857d5ed3c87a10cf22d0c`. Its 6318 runtime bytes have keccak256 `0x78da094992fb84bf17517277dcfc9c67ab36ba535f8879b5fd5b5614ac397bf4`. The snapshot SHA-256 and these identities are checked before installing code into empty local state.

`TestLocalEVMObservedCurrentChecksRuntimeSigning` creates a funded local check using the sampled code, obtains its actual returned/emitted hash and signs with the public SDK `SignEVMCheckRedemption` helper. The wrong caller is rejected without effects; the intended recipient redeems 100 raw DEL units with the expected event and balances. ECRECOVER tracing matches the SDK digest for each caller, and replay is rejected without retained effects.

This positively establishes the SDK recipient-bound signing profile for the tested paths of this exact sampled code. It resolves the earlier code-profile uncertainty for that sample while preserving the incompatible 6258-byte historical artifact as a separate regression. Runtime size alone is not the evidence: exact code identity, traced digests and successful execution are. The test does not copy live storage, establish public proxy initialization or ownership, certify every Checks route, or demonstrate public transaction acceptance/settlement. The initial nonce and balances belong to the fresh local fixture.

## Weighted Safe and factory runtime

The [Safe manifest](../testdata/evm/safe-manifest.json) pins runtime code captured from official testnet RPC at chain 202020, block 22477428, original RPC block hash `0xfb67691493eda26389cac1a122c8a6396192d97b153857d5ed3c87a10cf22d0c`. Snapshot SHA-256 and separate implementation/factory keccak256 hashes are checked before execution. The artifact is runtime-only, not verified Solidity source or original creation bytecode.

The local StateDB installs that code without copying live storage. The actual factory then creates and initializes a fresh proxy with synthetic owners weighted 2 and 1, threshold 3. Tests compare the SDK Safe transaction hash with the contract getter, reject insufficient signature weight, execute a signed transfer and reject replay. This validates the sampled code and new local proxy setup, not ownership, initialization or permissions of a public Safe instance. No standard Gnosis contract substitutes for Decimal's weighted Safe.

## Standard EIP2612 permit

The [permit manifest](../testdata/evm/permit-manifest.json) pins official OpenZeppelin Contracts v4.7.2 commit `64e48203cecad94f02de9891ecdeed4d629c6dae`, complete compiler inputs and compiled artifact hashes. This upstream ERC20PermitMock is explicitly a STANDARD compatibility fixture, not a Decimal token deployment. Reproduction uses the checked-in source inputs and pinned compiler; normal tests execute checked-in bytecode.

Tests use the SDK CheckedPermit and typed-data helpers, verify allowance and nonce changes, reject replay, wrong domain name/chain/contract, wrong nonce and expiry, and spend the approved allowance with transferFrom. Passing results establish those standard signing paths only. They do not promote any Decimal permit profile to deployed acceptance.

## Historical token Checks interoperability

The existing official historical NewToken and bundled Checks artifacts execute together. Tests verify approval, token custody, returned check hashes, creation/redemption events and allowance consumption. Missing or partial allowance and insufficient token balance revert without retained logs, custody changes or caller nonce changes.

A two-check redemption with a valid first proof and invalid second proof must roll back the entire batch. A subsequent valid batch redeems both checks, confirming rollback of status as well as transfers; replay fails. The signatures use the historical artifact's observed check-only digest inside tests. The separate JS recipient-bound helper mismatch remains documented and is not silently changed.

## Signed local Ethereum chain

An in-memory simulated Ethereum chain uses chain ID 1337 and synthetic signing keys. SDK Send/Broadcast/WaitReceipt exercise legacy type 0, access-list type 1 and dynamic-fee type 2 transactions, committed block/receipt identity, signed intent, recipient value, sender value-plus-fee delta and consumed nonce. A committed REVERT charges gas and consumes nonce while rolling back value and logs; failed gas estimation consumes no nonce.

This suite really signs and mines local test transactions. It does not submit to a public network. Discovery, external dialing and public RPC URLs are disabled; tests use in-process RPC, ordinary HTTP/WS/IPC listeners are disabled, and any backend-required authenticated service is restricted to ephemeral loopback. This Ethereum dev chain does not reproduce Decimal consensus, native modules, public fee policy or deployment state.

The later [contract route tests](LOCAL_CONTRACT_ROUTES.md) additionally execute permit on the exact observed Decimal reserveless implementation runtime, current Checks token/permit routes and signed weighted-Safe owner administration. This extends local code coverage without claiming public deployment acceptance.
