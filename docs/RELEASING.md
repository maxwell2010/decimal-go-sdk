# Build and publish

The public repository is https://github.com/maxwell2010/decimal-go-sdk . VERSION is the release version source. The first version is v0.1.0-preview.1. Both modules require Go 1.26.10 or later.

## Verify

Run in a clean standalone clone, not inside a private parent repository:

```sh
python -B -m unittest discover -s tools -p "test_*.py"
go mod verify
go test ./...
go vet ./...
go build ./...
go test -race ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
cd cosmos
go mod verify
go test ./...
go vet ./...
go build ./...
go test -race ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
cd ..
```

Race checks need a C compiler. PowerShell users can run `pwsh -File tools/check.ps1 -Race -Vulnerabilities -Fuzz`; an explicitly configured CC/CXX is respected. No check needs wallet credentials or submits a real transaction.

## Package

```sh
python -B tools/release.py
```

The tool reads a clean committed Git HEAD and emits versioned ZIP/TAR.GZ plus SHA256SUMS under ignored dist/. It includes source, tests, examples, ABI/schema evidence and licenses for both modules. It refuses caches, credential paths, binaries, symlinks/submodules and dirty snapshots. Never export private Git history or a parent project.

## Publish

Create immutable tags `v0.1.0-preview.6` and `cosmos/v0.1.0-preview.6` at the same reviewed commit; the nested module needs its own tag. Create a GitHub prerelease on the root tag, attaching only the two reviewed archives and SHA256SUMS. Use a new version for every correction; do not overwrite published tags/assets. GitHub's prerelease flag does not change Go version ordering, so the version also carries a SemVer preview suffix.

After publication, verify unauthenticated repository/asset downloads, compare SHA256SUMS and test `go get` for each module in fresh consumer modules without local replace directives. Check the public GitHub Actions results for the published commit. The root release is a library source release, not a Python wheel or a claim of funded-wallet acceptance. No network transaction campaign is part of publication.

## Advance the permanent install channel

After the new GitHub release is published and all three versioned assets are uploaded, run `python -B tools/update_latest.py`. The tool uses VERSION and refuses drafts, mismatched versions/URLs or missing assets. Review and commit only latest.json to main, then verify the raw manifest and the constant command `go run github.com/maxwell2010/decimal-go-sdk/cmd/decimal-sdk-install@latest -print`. Test a fresh installation and an upgrade from the prior release. Never advance the manifest before publication, overwrite a tag or replace versioned assets.

The manifest includes previews and is independent of GitHub releases/latest and Go's stable-first @latest selection. It may still point to the prior published version in the new tag's source snapshot: it is advanced on main after the new immutable release exists. The installer reads main, not a bundled stale manifest. Consumers explicitly run the tool to update; SDK startup does not fetch updates. Current fixed module paths support major versions 0 and 1; a future v2 requires reviewed versioned module paths and installer changes.

Documentation coverage is executable: tools/generate_transaction_examples.py --check and Go tests enforce all 260 ABI builders; cosmos/examples/messages tests cover 43 named schemas. Keep the generated artifacts reproducible when changing interfaces.

## Refresh read-only compatibility evidence

Use the bounded inspectors with new output paths, review the snapshots and their raw RPC number/hash identity, then update the canonical evidence files. Run `python -B tools/capture_compatibility_tests.py` and `python -B tools/build_compatibility_matrix.py`. The first records actual Go test outcomes and rejects concurrent source changes; the second rejects mismatched snapshots, stale source fingerprints and unrelated simulation payloads. `python -B tools/build_compatibility_matrix.py --check` is offline. Never promote a getter response, historical event or successful self-transfer eth_call to signed-transaction acceptance. A snapshot refresh is an explicit maintainer action, not background polling or a funded-wallet campaign.
