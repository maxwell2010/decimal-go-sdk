# Additional ABI evidence - 2026-10-29

All network operations in this investigation were reads. No wallet was loaded,
no transaction was signed, and no transaction was submitted.

## Imported interfaces

Three static-address interfaces from the official JS endpoint map were recorded
at testnet block 22474733: weighted Safe (33 methods), factory (5), and Safe
MultiSend (1). `additional-deployment-sources.json` records their runtime hashes,
public compact-API URLs and ABI SHA-256. Reproduce the import offline with
`python -B tools/import_extra_deployments.py`. These profiles are explicit;
they do not replace defaults or demonstrate initialized-wallet ownership.

Two token implementation interfaces add 76 methods. Official TokenCenter view
getters identified reserve implementation `0x8e215123f9cf7d2c7D99457C5096BeA82a868cbE`
and reserveless implementation `0x9D3e4E9953B031F2c89Da77c0F2abaD169eDd43F`.
Standard EIP1967 beacon storage and `implementation()` calls at block 22474794
independently linked the sampled token proxies to those implementations.
`token-deployment-sources.json` records the chain, blocks, getter, beacon and
codehash evidence. Reproduce with `python -B tools/import_token_deployments.py`.

The reserveless interface includes `mint(address,uint256)`, `cap()`, `mintable()`,
`burnable()`, `withCap()` and `owner()`. The reserve interface includes
`updateIdentity(string)` and `updateMaxTotalSupply(uint256)`; it does not justify
renaming these to the JS wrapper's selectors or inventing minimum-supply updates.
Sending to an implementation instead of the intended proxy is not supported by
this evidence.

## API annotation is not runtime identity

The API returned the same reserve-token-shaped ABI for the configured GasCenter
and the Checks implementation. That annotation was not imported under either
role. At the same block, actual Checks `nonces()` returned zero, while token
`name()`, `symbol()` and `nonces(address)` reverted. This supports preserving the
Checks configuration and rejecting the misleading Token ABI, rather than
deleting Checks based on API metadata. All four sampled GasCenter getters failed;
that does not prove the contract is absent or establish its role.

The observations are saved under `deployments/testnet-2026-10-29-additional`.
`cmd/abi-role-inspect` reproduces bounded read-only calls and refuses snapshot
overwrites. `cmd/abi-beacon-inspect` records the standard beacon binding.

## Still unavailable

The implementation compact API returns HTTP 400 for Delegation and DelegationNFT.
Their deployed Solidity 0.8.20 runtime CBOR trailers identify compiler-metadata
CIDs `QmWM4xhHphrHDXEuPmPfauZkbZkRrHgzjuV8ZN42sYRTGA` and
`QmYGEY7aWncZaaVFnPeLvKmhQNSgzfKeM1xLQFct9rQLas`. Bounded fetches through the
official Decimal NFT IPFS endpoint and public transport gateways did not obtain
the metadata. No ABI was inferred from selectors or unrelated contracts.

`cmd/abi-metadata-inspect` reconstructs a single-block UnixFS DAG-PB file node
before comparing the CIDv0 SHA-256 multihash. SHA-256 of raw HTTP bytes alone is
not a valid CIDv0 check. Files above 256 KiB require a multi-block verifier and
are refused. Retrieved content is untrusted data and is never executed.

Official `go-smart-node` stage commit `88dd556783e8dbb56c95ac12324122ae1f1d1aa8`
was also inspected: it adds no relevant ABI methods over the pinned master.
The old staging revision has no contracts directory. Official
`decimal-contracts` master `018a6c6d00616c3e24ae44a8fac429b2ccabbd12` contains a
historical bridge with a different protocol; it is not evidence of bridge-v2.

These remaining sources do not establish the missing current Delegation/NFT,
GasCenter or bridge-v2 interfaces. A successful offline builder is not a claim
of deployed contract acceptance, roles or settlement.

## Bridge discovery and exhausted source paths

The official JS `checkConnect('bridge')` does not use the ContractCenter slug.
It calls `getBridgeContracts()` against the official TheGraph endpoint. The
read-only query returned proxy `0x827eA5381aDB912Ed0befDA26AfE8CDc5A5e050a` and
implementation `0x06D7AC3874adC1f773c5165E91077c8037B789ff`. Standard EIP1967
storage and both runtime codehashes were confirmed at block 22474979. The compact
API nonetheless returned the same token-shaped 48-method interface for both
addresses. It was not imported as a Bridge ABI. Evidence is under
`deployments/testnet-2026-10-29-bridge`.

Official Bridge GraphQL schema introspection exposes only `id` and
`implementation`; there is no ABI/content field. The other five official JS
bridge endpoints returned GraphQL errors. The documented historical Go API
`TestnetGate/evm-contracts/{address}` returned HTTP 404 for the Bridge,
Delegation and DelegationNFT implementation addresses. No guessed endpoint or
verification mutation was invoked. `cmd/abi-source-inspect` reproduces these
read-only source-discovery calls.

Bridge runtime metadata identified CID
`QmbxENjmADRrrpHeQS4NKKM3sModCiQKy472rnDupWTp8G`. A final bounded metadata pass
also tried direct CIDv1 subdomain transport to avoid path-gateway redirects,
without obtaining a verifiable artifact. Its evidence is in
`deployments/testnet-2026-10-29-metadata-subdomain`. Gateway content would be
accepted only as the content-addressed artifact referenced by official runtime,
after CID verification, not as an independent ABI authority.

The public Bitbucket workspace enumeration returned 19 repositories in one page,
with no next page; only `decimal-contracts` and `dsc-evm-docs` were relevant
contract/documentation repositories beyond the three original SDK/node sources.
`git ls-remote --heads decimal-contracts` exposed only master, so there was no
unexamined public bridge-v2/develop branch there. No Wormhole interface or
same-named external contract was substituted.
