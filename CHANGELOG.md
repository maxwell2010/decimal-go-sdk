# Changelog

## v0.1.0-preview.9 - 2026-10-29

- Contract reads and simulations preserve cancellation, snapshot caller-owned target/block/value/calldata parameters and own response buffers, while preserving custom errors and proxy guards. Regression tests reproduce failures against preview.8.
- Exact observed Decimal reserveless implementation initializes in fresh local state and executes CheckedPermit/permit, Approval, allowance spending, replay/domain/expiry rollback and reinitialization rejection. Reserve code is captured only, not executed.
- Observed current Checks token custody/allowance, atomic receiver-bound batch redemption, standard-token permit issuance and expiry, with actual rollback assertions.
- Weighted Safe owner/threshold self-call administration, owner addition/removal, wrong-domain/outsider rejection and approveHash authorization on actual factory-created local proxies.
- No missing ABI, state layout, public wallet ownership or settlement was invented. Public transactions remain unexecuted.

## v0.1.0-preview.8 - 2026-10-29

- Observed official 6318-byte Checks implementation runtime accepts the recipient-bound SDK helper in local execution, with caller-specific ECRECOVER traces and replay rejection. This resolves the sampled code profile while preserving the historical 6258-byte artifact mismatch and public-state limitations.

- Genuine weighted Safe/factory runtime captured read-only from official testnet at fixed block/hash. Local factory-created proxy tests cover owner weights, threshold, SDK/contract hash equality, signatures, transfer, nonce and replay. No copied live storage or Solidity-source claim.
- Standard EIP2612 execution against pinned official OpenZeppelin v4.7.2 source, with exact compiler reproduction, CheckedPermit, nonce/allowance/Approval, replay/domain/expiry failures. This is not a Decimal token deployment fixture.
- Historical token Checks custody/allowance, creation failures and atomic batch redemption with actual token/Checks bytecode. Historical check-only signatures remain test-only.
- Signed/mined type 0/1/2 transactions in an isolated Ethereum chain 1337, with actual fees, balance deltas, nonces and reverted receipts. No Decimal consensus/typed-envelope admission inferred.
- Both fixture compilers run in CI; fingerprints include all new artifacts. No public transaction was submitted.

## v0.1.0-preview.7 - 2026-10-29

- Actual in-process CREATE/CALL execution of pinned official historical token and Checks artifacts, with persistent state, emitted events and revert assertions. No mined-chain or deployed-runtime parity claim.
- Token transfer/approval/allowance, mint/burn/burnFrom and unauthorized/insufficient-value failure scenarios.
- Checks creation/funding, historical-artifact redemption/replay/expiry and explicit ECRECOVER-traced signing-profile mismatch. The bundled artifact rejects recipient-bound JS helper proofs; the public helper was not silently changed.
- HTTP RPC resilience tests cover 429/503, malformed replies, dropped connections, timeouts, receipt identity and changing block hashes. Uncertain broadcasts remain one attempt.
- Three bounded event/receipt/signature fuzz targets and CI campaigns; exact solc/source/fixture provenance and reproducible compiler output verification.
- Public network transactions remain unexecuted; missing Safe/permit deployment fixtures and official ABI are not replaced with mock acceptance claims.

## v0.1.0-preview.6 - 2026-10-29

- Checks history/receipt readers and the chain guard preserve cancellation even when a backend returns successful data after canceling the request.
- Receipt waiting requires mined block coordinates, known status, nonnegative reported gas price and a nonzero requested transaction hash.
- Checks inspection rejects missing/null block numbers at genesis, preserving the distinction between height zero and an omitted field.
- Pinned official node/Ethermint source review documents native block identity and incomplete requireCanonical semantics. No stronger block-hash state binding is inferred.
- Regression tests and public install/update verification cover both modules. No network transaction was sent and no unavailable ABI was invented.

## v0.1.0-preview.5 - 2026-10-29

- Read-only deployment/getter snapshots on official testnet and mainnet at explicit block numbers and raw RPC hashes.
- Strict CreatedCheck/CreatedCheckDetails decoders, bounded history and receipt APIs, with six real testnet creation events and one independently verified public receipt.
- Client.BlockIdentity preserves the RPC block hash instead of recomputing Ethereum header identity on Decimal.
- Narrow positive self-transfer eth_call observations: two sampled testnet token implementations and mainnet native DEL. No account-control or settlement claim.
- Reproducible compatibility matrix for all 524 ABI methods per network and 93 reference operations, with independent codec/view/simulation/write-acceptance statuses.
- Mainnet configured Checks has implementation-like runtime and zero getters; initialization/usable-proxy readiness remains unconfirmed. Defaults were not silently replaced.
- No public-network signing, broadcasting, private key loading, initialization or state override occurred in the live inspectors.

## v0.1.0-preview.4 - 2026-10-29

- EVM Checks password-derived signers and recipient-bound redemption proofs for the reviewed official JS profile.
- Exact 52-byte packed hashing and EIP-191 personal-sign, external digest-signing callbacks, owner recovery/verification and normalized low-S signatures.
- Offline verified batch calldata with preserved proof/hash ordering and duplicate rejection.
- Three independently executed ethers reference vectors, negative signature/cancellation/mutation tests and a runnable public-fixture example.
- Official node source audit records check identity/storage/backing and distinguishes the migration nonce layout from the JS getter. No missing deployment ABI or checkDetailsHash derivation is invented.
- No blockchain transaction was submitted. Check hashes remain opaque contract outputs; deployment acceptance is separate.

## v0.1.0-preview.3 - 2026-10-29

- Installer requests the moving channel with no-cache headers and a fresh 128-bit random query key, avoiding stale CDN responses.
- A post-publication check of preview.2 reproduced selection of preview.1 through a cached response; the corrected CLI selects the current published channel.
- Regression coverage preserves the trusted manifest host/path and existing query, and checks unique cache keys. Timestamp-only keys were rejected after a Windows test reproduced a clock-resolution collision.
- Complete transaction documentation and examples from preview.2 are retained. Published tags and assets are preserved.

## v0.1.0-preview.2 - 2026-10-29

- Executable offline examples and bilingual documentation for every one of the 260 ABI write builders across 20 profiles.
- Unsigned transaction envelope examples for EVM types 0/1/2 using the real SDK Prepare and a local Backend.
- All 43 Cosmos message constructors with source-valid public wire fixtures, concrete payloads and current-native versus historical routing labels.
- A permanent latest.json install channel and cross-platform Go utility installing/updating both modules in one command, with print-only and per-module options.
- Install, update, pinning and rollback guides; channel publication requires an already published release with complete assets.
- No real transaction was submitted. ABI, permissions and deployed execution limitations remain unchanged.

## v0.1.0-preview.1 - 2026-10-29

Initial independent MintCandy Decimal Go SDK preview.

- EVM client with exact amounts, offline builders, reusable/external signers, transaction types 0/1/2, bounded fees, nonce coordination and receipt reconciliation.
- 20 ABI profiles and 524 generated methods: 300 pinned node/JS methods plus 224 explicitly selected versioned testnet methods. These are contract methods, not distinct business transaction types.
- ERC20/721/1155, staking, validators, checks, Safe signatures/weights, approvals and explicit proxy/beacon identity checks. Missing deployment interfaces remain explicit.
- Independent Cosmos module with 43 message schemas, stateless validation, signer derivation, direct signing, REST/gRPC and historical check codecs. The pinned current native ante admits only eight request types.
- Final audit fixes cancellation propagation, exhausted nonce rejection, simulation identity checks, foreign protobuf descriptor rejection, positive-height confirmation, generator provenance and generated type/name collisions.
- Russian and English guides, executable read-only/offline examples, reproducible source archives and SHA256SUMS.

This preview does not certify every deployed business operation. Official ABI remains unavailable or misannotated for some Delegation/DelegationNFT, GasCenter and bridge-v2 routes. No real transaction was submitted during Go SDK development or this audit; stable network acceptance remains outstanding. See docs/VALIDATION.md and docs/ADDITIONAL_ABI_EVIDENCE.md.
