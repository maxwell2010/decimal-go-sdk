param([switch]$Race, [switch]$Vulnerabilities, [switch]$Fuzz)
$ErrorActionPreference = 'Stop'
$sdkCheckRoot = Split-Path $PSScriptRoot -Parent
if ($Race) {
    $sdkCompiler = Join-Path $sdkCheckRoot '.tools/llvm-mingw-20261020-ucrt-x86_64/bin/x86_64-w64-mingw32-clang.exe'
    $env:CGO_ENABLED = '1'
    if ($env:CC) {
        # Keep an explicitly configured compiler (including wrapper arguments).
    } elseif (Test-Path -LiteralPath $sdkCompiler) {
        $env:CC = $sdkCompiler
        $env:CXX = Join-Path (Split-Path $sdkCompiler) 'x86_64-w64-mingw32-clang++.exe'
        $env:PATH = (Split-Path $sdkCompiler) + [IO.Path]::PathSeparator + $env:PATH
    } elseif (-not (Get-Command (go env CC) -ErrorAction SilentlyContinue)) {
        throw 'Race checks require a C compiler. Set CC/CXX or install LLVM-MinGW.'
    }
}
python -B -m unittest discover -s (Join-Path $sdkCheckRoot 'tools') -p 'test_*.py'
if ($LASTEXITCODE -ne 0) { throw 'ABI generator checks failed' }
foreach ($sdkModule in @($sdkCheckRoot, (Join-Path $sdkCheckRoot 'cosmos'))) {
    Push-Location -LiteralPath $sdkModule
    try {
        go mod verify
        if ($LASTEXITCODE -ne 0) { throw 'Dependency verification failed' }
        go test ./...
        if ($LASTEXITCODE -ne 0) { throw 'Tests failed' }
        go vet ./...
        if ($LASTEXITCODE -ne 0) { throw 'Static checks failed' }
        go build ./...
        if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
        if ($Race) {
            go test -race ./...
            if ($LASTEXITCODE -ne 0) { throw 'Race tests failed' }
        }
        if ($Vulnerabilities) {
            go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
            if ($LASTEXITCODE -ne 0) { throw 'Reachable vulnerabilities found' }
        }
    } finally { Pop-Location }
}
if ($Fuzz) {
    Push-Location -LiteralPath $sdkCheckRoot
    try {
        go test -fuzz '^FuzzExactAmounts$' -fuzztime 10s -parallel 2 .
        if ($LASTEXITCODE -ne 0) { throw 'Amount fuzz test failed' }
        go test -fuzz '^FuzzSignatureValidation$' -fuzztime 10s -parallel 2 .
        if ($LASTEXITCODE -ne 0) { throw 'Signature fuzz test failed' }
        foreach ($sdkDecoder in @('FuzzChecksEventDecoding', 'FuzzChecksReceiptDecoding', 'FuzzChecksSignatureRecovery', 'FuzzDelegationStakeDecoding', 'FuzzDelegationEventsAndReceipts')) {
            go test -run '^$' -fuzz $sdkDecoder -fuzztime 7s -parallel 2 .
            if ($LASTEXITCODE -ne 0) { throw "Decoder fuzz test failed: $sdkDecoder" }
        }
    } finally { Pop-Location }
    Push-Location -LiteralPath (Join-Path $sdkCheckRoot 'cosmos')
    try {
        go test -fuzz '^FuzzPinnedMessageDecode$' -fuzztime 10s -parallel 2 .
        if ($LASTEXITCODE -ne 0) { throw 'Cosmos protobuf fuzz test failed' }
        go test -fuzz '^FuzzPinnedValidateBasic$' -fuzztime 10s -parallel 2 .
        if ($LASTEXITCODE -ne 0) { throw 'Cosmos validation fuzz test failed' }
    } finally { Pop-Location }
}
