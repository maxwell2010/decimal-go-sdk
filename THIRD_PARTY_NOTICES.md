# Third-party notices

The independent MintCandy Go implementation is MIT licensed; it is not an official Decimal release.

- Pinned Decimal node interfaces and source-derived compatibility rules: BIT TEAM GROUP LTD, MIT. See THIRD_PARTY_NODE_LICENSE and docs/sources.json.
- Checks/Multicall ABI artifacts and interoperability requirements studied from official dsc-js-sdk: package declares ISC. See THIRD_PARTY_JS_LICENSE. The complete JS SDK is not bundled; an offline Checks reference runner and pinned dependency manifest are included with source provenance in docs/CHECKS_JS_EVIDENCE.md. Its npm dependencies are not shipped or used at Go runtime.
- Deployment ABI JSON is a versioned public official API artifact with source URLs, checksums and limits in docs/*deployment-sources.json. No role or execution certification is implied.
- The optional sender-restricted multisend adapter is a MintCandy project extension, not an official Decimal deployment default.
- Runtime Go dependencies are downloaded separately by Go and retain their own licenses. go.mod/go.sum pin the module graph. The historical reference modules under cosmos/tools have separate dependency graphs and do not enter runtime dependency resolution.

The Cosmos module includes compiled protobuf descriptors derived from the source inputs listed in cosmos/schema/sources.json. Its standalone notices and accompanying license texts are in cosmos/THIRD_PARTY_NOTICES.md and cosmos/LICENSES/.
