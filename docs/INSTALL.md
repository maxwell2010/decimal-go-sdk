# Install and update

[Русский](INSTALL.ru.md) | [Getting started](GETTING_STARTED.md) | [Transaction examples](TRANSACTIONS.md)

Requires Go 1.26.10 or later and internet access to download modules. Check `go version`. Normal Go module proxy use does not require GitHub login; cloning source or using direct Git fetches requires Git. The SDK is a library dependency of your Go application.

## One permanent install/update command

Inside an existing project with a go.mod file, always run:

```sh
go run github.com/maxwell2010/decimal-go-sdk/cmd/decimal-sdk-install@latest
```

The tool reads the [permanent latest.json](https://raw.githubusercontent.com/maxwell2010/decimal-go-sdk/main/latest.json), then installs or updates both EVM and Cosmos modules together with one exact-version go get command. The published channel includes previews. Go downloads dependencies and updates the application's go.mod/go.sum. Updates happen only when you explicitly run the command; the SDK never updates itself.

Permanent human-readable guide: https://github.com/maxwell2010/decimal-go-sdk/blob/main/docs/INSTALL.md .

For a new application:

```sh
mkdir decimal-demo
cd decimal-demo
go mod init example.com/decimal-demo
go run github.com/maxwell2010/decimal-go-sdk/cmd/decimal-sdk-install@latest
```

Inspect the selected command without changing a project:

```sh
go run github.com/maxwell2010/decimal-go-sdk/cmd/decimal-sdk-install@latest -print
```

Select a single module if needed:

```sh
go run github.com/maxwell2010/decimal-go-sdk/cmd/decimal-sdk-install@latest -evm-only
go run github.com/maxwell2010/decimal-go-sdk/cmd/decimal-sdk-install@latest -cosmos-only
```

The tool requires an existing go.mod and never initializes a project implicitly. Manifest fetch/validation failures stop installation; it does not choose an arbitrary fallback version. If a module proxy has not indexed the new installer immediately after publication, wait for indexing or use the pinned module command below.

## Pinning and rollback

For a reproducible installation of this release:

```sh
go get github.com/maxwell2010/decimal-go-sdk@v0.1.0-preview.4 github.com/maxwell2010/decimal-go-sdk/cosmos@v0.1.0-preview.4
```

To roll back, replace both versions with an earlier published version such as v0.1.0-preview.1. Published tags and assets are never overwritten. Verify your application afterward:

```sh
go list -m github.com/maxwell2010/decimal-go-sdk github.com/maxwell2010/decimal-go-sdk/cosmos
go mod verify
go test ./...
```

Import `decimal "github.com/maxwell2010/decimal-go-sdk"` and optional `cosmos "github.com/maxwell2010/decimal-go-sdk/cosmos"`. Only add imports your source uses; go mod tidy removes unused dependencies.

## Standard Go latest versus the published channel

Direct `go get github.com/maxwell2010/decimal-go-sdk@latest` follows Go's preference for stable releases over previews. The channel installer instead reads the exact reviewed published SDK version from the manifest. The @latest query on the install command selects the utility version, while its current manifest selects the SDK version. See [Go version queries](https://go.dev/ref/mod#version-queries).

## Archives and runnable examples

latest.json also links the current release, source ZIP/TAR.GZ and SHA256SUMS. The archives include both modules, tests, examples, documentation and licenses. Verify downloaded files against SHA256SUMS. GitHub releases/latest and Go @latest have their own selection policies; this manifest remains at one permanent URL. See [GitHub release links](https://docs.github.com/en/repositories/releasing-projects-on-github/linking-to-releases).

From the downloaded source or repository clone:

```sh
go run ./examples/offline_transfer
go run ./examples/token_transfer
go run ./examples/envelopes
go run ./examples/transactions
go run ./examples/checks_signing
go -C cosmos run ./examples/messages
```

These commands build/encode locally without submission. Checks additionally signs a public fixture message, never a blockchain transaction or user key. The separate read_balance example performs RPC reads using a public address. See [all 260 ABI builders](TRANSACTIONS.md), [envelope types 0/1/2 and workflows](TRANSACTION_ENVELOPES.md), and [43 Cosmos message examples](../cosmos/TRANSACTIONS.md). Builders/codecs do not establish deployment acceptance or stateful authorization.
