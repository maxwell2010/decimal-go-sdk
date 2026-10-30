# Mainnet recovery compatibility audit, 2026-10-30

Scope: the node update associated with recovery height 34342701, pinned to official `go-smart-node` commit `a46cb48d7965c1a64a495cebb22e735dfe4b9da3`. Earlier Stake7/8 work was reused and extended. [Source and fixture hashes](DELEGATION_UPGRADE_SOURCES.json), [API guide](DELEGATION_UPGRADE.md), [Russian examples](DELEGATION_UPGRADE.ru.md).

The implementation was compared with the coordinating PythonSDK checkout `sdk-release-014`, specifically `docs/validation/mainnet-recovery-compatibility-2026-10-30.md` and `src/decimal_web3_sdk/nft.py`. The coordinating project reported 579 passing Python offline tests; that Python suite was not rerun or modified here.

| Requirement | Go SDK result | Relation to Python |
|---|---|---|
| Stake seven/eight fields | Six legacy/current getters, frozen nesting and arrays; nullable HoldStartTime; unchanged amount | Same separation of absent start, explicit zero, ending timestamp and amount |
| Authoritative ABI | New ABI extracted from the pinned Go-binding, not adjacent stale JSON | Same source choice; all 16 legacy event definitions also match official delegationv7 after ignoring internalType descriptions |
| Unified NFT views | Explicit UnifiedNFTDelegation constructor and separate NFTStake tuple10 | Equivalent explicit access to new positions; not decoded as Stake7/8 |
| New NFT actions | Thirteen official user builders; new legacy deposits rejected for chain75 and known retired address | Go uses explicit addresses rather than Python's implicit default-service routing |
| NFT approval operator | Token-specific ERC721 and explicitly broad ERC1155 calldata derive operator from unified destination | Same destination guarantee; constructing the client or action sends no approval |
| Old positions/custom networks | Existing DelegationNFT read/exit builders remain; no Config mutation or global address substitution | Preserves legacy access and unrelated/custom configurations |
| Events | Both original topic/layout generations, exact address filtering, strict known-event validation, unsupported-topic skip | Equivalent receipt handling; source receipt unchanged |
| Multiple NFT exits | Repeated stake IDs retain individual indexes and source order | No stake-ID-only deduplication; tested for legacy, current and unified views |

Offline verification after all code changes:

- `python -B tools/capture_compatibility_tests.py`: 1,022 root test/example/subtest pass events across 168 top-level entries; Cosmos 179 events across 31 entries.
- `pwsh -NoProfile -File tools/check.ps1 -Race`: both modules pass tests, vet, build, dependency verification and race checks; 31 Python tooling tests pass.
- `python -B tools/build_compatibility_matrix.py --check`: passes. The pinned/versioned ABI catalog still has 524 methods per network; separately supplied unified wrappers do not silently certify or replace those profiles.
- Saved mainnet samples: eight eight-field positions at block `0x20c2060`, 16 StakeUpdated/StakeAmountUpdated logs and two empty-position NFT view probes. Copied exact bytes are hashed and decoded offline; no repeat RPC requests for those scenarios.
- Root source fingerprint: `1488596701fcd6eccfc2778f0078c873a1d2d4fb8ce8f76eaad1fb11c1a851c8`.

Boundaries: source and saved-response compatibility do not prove a funded NFT lifecycle, every transaction admission rule, proxy initialization, reserve settlement or all NFT variants. The exact ABI contains ERC721 delegateNFTByPermit but no matching ERC1155 permit route; missing methods are not invented. No real signatures, approvals, public transactions, history replay, production deployment were performed during the audit. These additions are included in v0.1.0-preview.11. Release publication is a separate authorized maintainer step; it does not submit public-chain transactions.
