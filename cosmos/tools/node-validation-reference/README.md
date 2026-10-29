# Executed pinned node validation reference

This isolated historical module executes official `Msg.ValidateBasic` methods from node revision `9e6c6d718d662083c4a524376a66d2c50bd4bc77` (`v0.0.0-20260723114515-9e6c6d718d66`). It imports only message type packages and SDK upgrade types, not the application, keepers, a wallet or a network client. Its historical dependencies are not part of the production Cosmos module dependency graph.

The fixture corpus is generated in the parent module using:

```sh
go test -run '^TestExportNodeValidationCorpus$' -args -export-validation-corpus
```

This produces 129 public deterministic wire cases: valid, empty and bad-signer for each of 43 schemas. In this directory, run `go mod verify` before `go run .`. The program uses SDK account/operator HRPs from the pinned node and reproduces the pinned app's denomination-regex override without importing/starting the app. Each case is decoded into the actual upstream Go type using the upstream protobuf registry, then its actual ValidateBasic method is invoked. Output records upstream acceptance, decoder failures and panics. A nonzero exit indicates disagreement with the local port.

The successful run produced `../../schema/validation-node-reference.json`: 129 cases, 43 types, zero mismatches, zero panics and zero decoder failures. The parent regression test compares its decisions with that frozen output. This is differential evidence for these inputs, not proof of equivalence over every possible message.

## Cache verification and reproduction

The first attempt was stopped before any reference execution because the host's shared module cache failed verification:

```text
github.com/btcsuite/btcd v0.23.4: zip has been modified
github.com/btcsuite/btcd v0.23.4: dir has been modified
github.com/hashicorp/go-getter v1.7.0: dir has been modified
```

No shared files were repaired or deleted. A fully fresh-cache download was stopped with Ctrl-C when large dependency downloads stalled at zero-byte temporary archives. The successful attempt used an isolated cache under ignored `gosdk/.tools/node-validation-seededcache`.

`seed_cache.py SHARED_CACHE ISOLATED_CACHE` copies only module ZIPs whose independently calculated Go `dirhash.Hash1` checksum equals the corresponding pinned `go.sum` entry. It never changes the shared cache and rejects checksum mismatches, including the modified btcd ZIP. It needs `go list -m -json all > modules.json` first. The successful seed verified 227 ZIPs; other required dependencies were fetched normally by Go.

With `GOMODCACHE` set to that isolated directory, `go list -deps .` fetched/extracted only required package sources, then `go mod verify` returned exactly `all modules verified` **before** `go run .` was executed. The Go toolchain was 1.26.10. The isolated module checksum, fixture checksum and direct source versions are recorded in `../../schema/validation-node-reference-provenance.json`.

Do not format, patch or otherwise edit module-cache files. Use a new isolated cache and repeat checksum verification if verification fails. Reference generation does not submit transactions, load user secrets or require a live chain.
