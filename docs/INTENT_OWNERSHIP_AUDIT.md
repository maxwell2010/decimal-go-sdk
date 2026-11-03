# Intent and result ownership audit, 2026-10-30

This follow-up extends preview.12 and is included in preview.13; earlier immutable tags are unchanged. Three agents reviewed core preparation, Cosmos callbacks/query ownership and unified NFT argument coverage.

- `Client.Prepare` snapshots Call/value/calldata/deployment evidence and TxOptions nonce/fees/access list before the first backend callback. Gas estimation receives a separate destination pointer. Successful RPC responses cannot hide context cancellation. Nil Value validation and nonce/submission behavior remain unchanged.
- The target-mutation and cancellation-after-nonce regressions fail against preview.12 through an ignored Go overlay and pass with the fixes. An explicit-nonce regression also mutates caller-owned inputs during a backend callback and checks the resulting intent.
- Cosmos direct signing now gives context cancellation priority when a callback cancels and returns an error. The new test uses a public generator-point key and returns no signature. Callback diagnostics remain redacted.
- Cosmos gRPC returns an owned protobuf clone, including nested fields. Backend reply mutations and caller result mutations no longer affect each other. Both defects were reproduced with disposable local transports before changes.
- All thirteen manual unified NFT builders are tested with distinct addresses, amounts, timestamps and supplied proof fields, asserting every decoded ABI position. Approval checks also assert token ID and boolean grant. A source overlay swapping validator/collection passes the old published test and fails the strengthened test; production files were not replaced during this comparison.
- [Compiled offline example](../examples/unified_nft/main.go) covers all thirteen user actions and both approval builders. It performs no RPC, signing or submission.
- Linux CI now includes all nine previously available bounded fuzz campaigns: seven EVM campaigns and two Cosmos campaigns. This is local workflow preparation; remote results for a future release must be checked separately.

The refreshed compatibility capture records 1,079 root passing test/example/subtest events across 176 top-level entries and 190 Cosmos events across 36 entries. These are runner counts, not transaction types. New core/Cosmos regressions are required by capture. The matrix retains 524 pinned/versioned ABI methods per network, with unified NFT wrappers documented separately.

The combined verification command passed with exit code 0: `pwsh -NoProfile -File tools/check.ps1 -Race -Vulnerabilities`. No new fuzz implementation was introduced in this follow-up; all nine existing campaigns passed the preceding bounded run, and their CI coverage is now continuous. Local tests do not certify a funded NFT lifecycle, public roles/settlement, mainnet Checks initialization or missing bridge-v2 ABI/proof formats. See [stable criteria](STABLE_RELEASE.md).

No public transaction, real-user signature, permission change, deployment or release publication was performed for this follow-up. Earlier preview.12 remains available unchanged; release verification is linked from preview.13.

Both modules passed tests, vet, build, module verification and race checks; all 34 Python tooling tests passed. The dated vulnerability scan reports zero reachable findings and zero affected imported packages, with the existing one unused module advisory.
