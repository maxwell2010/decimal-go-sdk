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

Create immutable tags `v0.1.0-preview.1` and `cosmos/v0.1.0-preview.1` at the same reviewed commit; the nested module needs its own tag. Create a GitHub prerelease on the root tag, attaching only the two reviewed archives and SHA256SUMS. Use a new version for every correction; do not overwrite published tags/assets. GitHub's prerelease flag does not change Go version ordering, so the version also carries a SemVer preview suffix.

After publication, verify unauthenticated repository/asset downloads, compare SHA256SUMS and test `go get` for each module in fresh consumer modules without local replace directives. Check the public GitHub Actions results for the published commit. The root release is a library source release, not a Python wheel or a claim of funded-wallet acceptance. No network transaction campaign is part of publication.
