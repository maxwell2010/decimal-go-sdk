# Public testnet deployment evidence

Read-only snapshot collected by `go run ./cmd/abi-inspect`. Network ID 202020 was checked before querying. Registry reads, ERC1967 implementation storage and runtime code were all pinned to block **22474456**. See `metadata.json` for addresses, public API URLs, code hashes and ABI SHA-256.

The compact-contract API returns proxy-only ABIs for the six system proxy addresses. Those interfaces contain constructors/errors/fallback, without callable business methods. Consequently `LoadDeploymentContract` now fails explicitly for this response; callers must obtain and verify an implementation ABI and apply it to the proxy address.

Implementation ABIs could be downloaded for ContractCenter, TokenCenter, NFTCenter and MasterValidator. Compact API returned HTTP 400 for the Delegation and DelegationNFT implementation addresses. The registry returned zero for the queried slugs `gas-center`, `bridge` and `bridge-v2`; this proves only that those exact slugs were unregistered at the observed block.

NFTCenter implementation `0xa12ffD3f622E5399274E4Ad13032A5cacce02916`, reached through proxy `0x11aE5297d0B5886914661938FcB8AEdc08622194`, exposes these public API ABI signatures:

- `createDRC721((address,string,string,string,bool,bool))`
- `createDRC1155((address,string,string,string,bool,bool))`
- `createDRC721Reserveless((address,string,string,string,bool,bool))`
- `createDRC1155Reserveless((address,string,string,string,bool,bool))`

The six-field tuple is evidence of a deployment generation differing both from the pinned node ABI and from the JS seven-field `NFTCollection` declaration. The operation matrix links this evidence but does not relabel the operations as pinned-node parity. No transaction was signed, simulated or broadcast, and no funded wallet was used. An API-provided ABI plus implementation code hash does not cryptographically prove that every ABI method is implemented or permitted for a given sender. Proxy implementation storage can change after the snapshot.

These are public interface artifacts and observations, not production configuration defaults. Keep the proxy as the transaction target; the implementation address is provenance rather than a replacement transaction destination.

## Versioned typed APIs

Run `python -B tools/import_deployments.py` to verify snapshot identity and artifact checksums, normalize these interfaces and generate four explicitly named profiles:

- `TestnetContractCenter20261029`
- `TestnetTokenCenter20261029`
- `TestnetNFTCenter20261029`
- `TestnetMasterValidator20261029`

There are **109 deployment-artifact methods** in the separate `docs/deployment-catalog.json`; provenance is in `docs/deployment-sources.json`. They are excluded from the 300 pinned node/JS methods. The caller explicitly constructs the versioned wrapper at the proxy address. NFT creation has concrete typed `BuildCreateDRC721`, `BuildCreateDRC1155` and corresponding reserveless builders. These methods can also be selected through reference-operation routing without treating the implementation artifact as a new default ABI. After regeneration run `gofmt -w .`.
