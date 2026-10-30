# Local stable-readiness follow-up, 2026-10-30

This audit extends published preview.11; its fixes are included in preview.12 without changing earlier immutable tags. Three agents independently reviewed NFT APIs, delegation decoding/fuzz properties and stable/source/Cosmos gaps. [Stable criteria](STABLE_RELEASE.md), [Russian criteria](STABLE_RELEASE.ru.md), [deployment acceptance resources](ACCEPTANCE.md).

Closed local defects and evidence:

| Area | Fixed or verified | Practical boundary |
|---|---|---|
| NFT targets | Nil/empty wrappers return errors; all thirteen action builders and both approval helpers reject a missing/zero target | A zero target can no longer accidentally turn the ERC721 approval helper into revoke calldata; valid operator/ABI behavior remains unchanged |
| Receipt ownership | Decoded event Log.Data/Topics own their buffers | Editing one result cannot mutate the source receipt or sibling results; the regression independently fails against preview.11 via Go overlay |
| Cosmos gRPC | Cancellation takes priority after a successful Invoke | A custom transport returning success after cancellation no longer produces a successful Query/Simulate result |
| Cosmos REST | Cancellation checked after response, body read and JSON decode; body closed on all tested paths | Success/error responses after cancellation do not leak partial account results or underlying transport/body diagnostics; old-source overlays reproduce the defects |
| Recovery ABI | Rebuilt exact bytes from pinned official Go-binding with fixed source SHA256 and ABI SHA256 | Offline importer rejects altered source, altered destination and unreviewed extraction; adjacent stale JSON is not an accepted source |
| Install rollback | Published root and Cosmos modules exercised through preview.11 -> preview.10 -> preview.11 without replace directives | Common EVM API plus Cosmos descriptor compile/run and go mod verify pass at every step; returning to11 uses the permanent installer |
| Decoder fuzzing | Bounded property campaigns cover all six getters in both layouts plus old/new events, address filters and repeated receipt ordering | No malformed-data panic/noncanonical acceptance observed in these runs; this is not a formal proof for every input |

The importer is deliberately local and does not execute the official source:

```sh
python -B tools/rebuild_delegation_compat.py path/to/DecimalDelegation.go --check
```

It pins node commit `a46cb48d7965c1a64a495cebb22e735dfe4b9da3`, source SHA256 `248199aadf86bb4bd6836b70f13f0784b976e7edf65e7908f2c34e6bebd714bb` and ABI SHA256 `600ea8ed2c542b284e7eadf222e738c13663f029becf22b056b56678c5948fa0`. The source manifest's serialization description was corrected to compact JSON plus final LF; reviewed ABI bytes and their hash were unchanged. CI adds retrieval of the immutable official Go-binding followed by this exact-byte check. Importer negative tests run without network access.

Local final verification passes (exit code 0): `pwsh -NoProfile -File tools/check.ps1 -Race -Vulnerabilities -Fuzz` for both modules. The refreshed capture records 1,075 root pass events across 172 top-level entries and 188 Cosmos events across 34 entries. These counts include examples/subtests/fuzz seeds and do not count transaction types. The matrix remains 524 pinned/versioned methods per network; thirteen explicit unified NFT wrappers are separately documented. The new transport regressions and receipt/NFT regressions are required by capture tooling.

The remaining stable blockers for the advertised complete deployment scope are external evidence:

- Exact live deployment identity/generation and stateful authorization, roles, fees and settlement for advertised Safe, permit, NFT and other routes. Saved empty NFT view probes do not prove a funded NFT lifecycle.
- Confirmed initialization/funding and working proxy context for the configured mainnet Checks instance. Local execution of sampled code proves its tested signature path only.
- Authoritative missing bridge-v2/deployment-extension ABI and proof formats. Recovery Delegation interfaces are now known; no speculative ABI or proof is substituted.
- Remote Linux/Windows CI and source/module verification for the exact release commit. Results for preview.12 are linked from its GitHub release page; older preview.11 results do not substitute for them.

No public transaction, approval, real-user signature, production deployment or history replay was performed. Offline tests use synthetic fixtures and isolated EVM state; they do not replace the remaining public deployment evidence. No stable tag is created from these local checks.

The combined run completed all nine bounded fuzz campaigns, including the new delegation getter and event/receipt campaigns. Thirty-four Python tooling tests pass. Both modules pass tests, vet, build, module verification and race checks. The dated vulnerability scan reports zero reachable findings and zero affected imported packages, with the existing one unused module advisory.
