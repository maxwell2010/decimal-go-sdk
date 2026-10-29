# Changelog

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
