# Executable documentation coverage - 2026-10-29

The preview.2 documentation update adds examples for the SDK surface without submitting transactions.

| Surface | Runnable example | Coverage proof |
|---|---|---|
| EVM envelope types 0/1/2 | go run ./examples/envelopes | Real SDK Prepare using a local Backend; type, nonce, unsigned bytes, fee estimates/reserves and round-trip tests |
| ABI write builders | go run ./examples/transactions | All 260 builders across 20 profiles, independently checked against all 4 catalogs, selector and canonical ABI argument encoding |
| Cosmos messages | go -C cosmos run ./examples/messages | All 43 named constructors, exact public wire fixtures from the executed official validation reference, ValidateBasic/GetSigners/Pack/EncodeBody |
| Install/update channel | cmd/decimal-sdk-install | Pure/httptest checks for exact modules/versions/URLs, bounded JSON, cancellation, print-only, active module and invocation flags |
| Channel publication | tools/update_latest.py | Published non-draft release and complete uploaded asset metadata required before atomic manifest update |
| Documentation consistency | tools/test_documentation.py | Local navigation targets and generated catalog documentation are checked offline; external HTTP content and heading anchors are outside this check |

Spaces in commands above are literal; run from the repository root. [English installation](INSTALL.md), [Russian installation](INSTALL.ru.md), [ABI builder guide](TRANSACTIONS.md), [EVM types and workflows](TRANSACTION_ENVELOPES.md), [Cosmos messages](../cosmos/TRANSACTIONS.md).

260 write builders plus 264 readers account for the existing 524 ABI methods; this is not 524 independent business transaction types. Cosmos retains 8 admitted native request types and 35 historical/schema types in the pinned node classification. Signature/proof/role fields in ABI examples and some nested Cosmos payloads are fixtures, not execution authorization.

The installer never invokes blockchain RPC or signs a transaction. Final publication checks verify both module tags at the same commit, unauthenticated release assets and SHA256SUMS, fresh consumer installation and upgrade. latest.json is advanced on main only after the new release is available; the immutable source tag can contain the previous channel version. No real transaction is signed or submitted by these examples or checks.
