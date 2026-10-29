# Final audit - 2026-10-29

Scope: EVM core/transport, ABI/contracts/Safe/proxy simulation, Cosmos codecs/transport/confirmation, generation and public release packaging. Three agents reviewed independent areas; fixes received regression coverage and a combined verification run. No real transaction, wallet loading or funded-wallet campaign is part of this audit.

| Finding | Resolution | Regression |
|---|---|---|
| In-flight cancellation was replaced by generic errors in six read/subscription wrappers | Preserve context cancellation while hiding transport details | core_audit_test.go |
| Prepare/Broadcast accepted an exhausted MaxUint64 nonce | Reject before signing/submission; preserve the last usable nonce | core_audit_test.go |
| Simulation bypassed proxy/beacon identity checks | Validate immutable evidence and target; pin identity and call to the same block | simulation_deployment_test.go |
| gRPC accepted foreign descriptors with a matching FullName | Require the embedded pinned descriptor identity before invoking transport | cosmos/audit_regression_test.go |
| A Cosmos CheckTx-like result could look like confirmation | Require a positive int64 committed height for REST/gRPC confirmation | cosmos/audit_regression_test.go |
| Generator accepted malformed source types and colliding names | Validate supported ABI syntax; produce valid unique names; reject conflicting generated Go types | tools/test_generate.py, including actual Go compilation |
| Git diff missed untracked or index-hidden source edits | Check tracked identity and compare normalized Git blob content against pinned HEAD | tools/test_generate.py, including assume-unchanged/skip-worktree |
| Release had no public Go module path or standalone artifacts | Public root/nested module paths, license notices, bilingual examples/guides, clean-snapshot archives and checksum verification | tools/test_release.py and consumer installation checks |

All 20 ABI profiles regenerate without changes to the checked-in wrappers. Interface coverage remains 524 methods; all 43 Cosmos schemas retain their source-backed validation and current-ante admission classification.

Detailed final commands, counts and limitations are in VALIDATION.md. Unavailable/misannotated official ABI is documented in ADDITIONAL_ABI_EVIDENCE.md; it was not guessed or replaced by unrelated interfaces. The audit does not establish live execution parity, stateful authorization, bridge settlement or stable production readiness.
