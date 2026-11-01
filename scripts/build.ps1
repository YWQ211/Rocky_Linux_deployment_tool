param([string]$GoPath = 'go')
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$savedEnv = @{}
foreach ($key in @('GOCACHE','GOTOOLCHAIN','GOOS','GOARCH','CGO_ENABLED')) { $savedEnv[$key] = [Environment]::GetEnvironmentVariable($key, 'Process') }
Push-Location $projectRoot
try {
    $env:GOCACHE = Join-Path $projectRoot '.cache\go-build'
    $env:GOTOOLCHAIN = 'local'
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    & $GoPath test ./... -count=1
    if ($LASTEXITCODE -ne 0) { throw 'Tests failed' }
    & $GoPath vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'go vet failed' }
    $releaseDir = Join-Path $projectRoot 'dist\1.0.0'
    New-Item -ItemType Directory -Path $releaseDir -Force | Out-Null
    foreach ($target in @(@('windows','amd64','rockdev-windows-amd64.exe'), @('linux','amd64','rockdev-linux-amd64'), @('linux','arm64','rockdev-linux-arm64'))) {
        $env:GOOS = $target[0]
        $env:GOARCH = $target[1]
        & $GoPath build -trimpath -ldflags '-s -w' -o (Join-Path $releaseDir $target[2]) ./cmd/rockdev
        if ($LASTEXITCODE -ne 0) { throw ('Build failed: ' + $target[2]) }
    }
    foreach ($language in @('en','ja')) {
        foreach ($arch in @('amd64','arm64')) {
            $env:GOOS = 'linux'
            $env:GOARCH = $arch
            & $GoPath build -trimpath -ldflags ('-s -w -X rockdev/internal/cli.DefaultLanguage=' + $language) -o (Join-Path $releaseDir ('rockdev-linux-' + $arch + '-' + $language)) ./cmd/rockdev
            if ($LASTEXITCODE -ne 0) { throw 'Language build failed' }
        }
    }
    $packageFiles = @(Get-Item -LiteralPath '.gitignore','go.mod','README.md','README.en.md','README.ja.md','CHANGELOG.md','docs/release-1.0.md','docs/release-1.0.en.md','docs/release-1.0.ja.md','docs/test-results-1.0.md','docs/test-results-1.0.en.md','docs/test-results-1.0.ja.md','docs/test-results-1.0.tsv','docs/test-results.md','docs/test-results.en.md','docs/test-results.ja.md','docs/test-results.tsv','docs/verification.md','docs/vm-acceptance.md','docs/vm-quickstart.md','docs/vm-quickstart.en.md','docs/vm-quickstart.ja.md','scripts/build.ps1','scripts/integration-v1.sh','scripts/integration.sh','scripts/integration-edge.sh','scripts/integration-final.sh') + @(Get-ChildItem -Path cmd,internal,examples -File -Recurse)
    $zipStream = [IO.File]::Open((Join-Path $releaseDir 'rockdev-1.0.0-source.zip'), [IO.FileMode]::Create)
    try {
        $sourceZip = [IO.Compression.ZipArchive]::new($zipStream, [IO.Compression.ZipArchiveMode]::Create)
        try {
            foreach ($sourceFile in $packageFiles) {
                $relativePath = [IO.Path]::GetRelativePath($projectRoot, $sourceFile.FullName).Replace('\','/')
                [IO.Compression.ZipFileExtensions]::CreateEntryFromFile($sourceZip, $sourceFile.FullName, $relativePath, [IO.Compression.CompressionLevel]::Optimal) | Out-Null
            }
        } finally { $sourceZip.Dispose() }
    } finally { $zipStream.Dispose() }
    $artifactNames = @('rockdev-linux-amd64','rockdev-linux-arm64','rockdev-windows-amd64.exe','rockdev-linux-amd64-en','rockdev-linux-amd64-ja','rockdev-linux-arm64-en','rockdev-linux-arm64-ja','rockdev-1.0.0-source.zip')
    $sums = foreach ($name in $artifactNames) { $hash = (Get-FileHash -LiteralPath (Join-Path $releaseDir $name) -Algorithm SHA256).Hash.ToLowerInvariant(); "$hash  $name" }
    [IO.File]::WriteAllLines((Join-Path $releaseDir 'SHA256SUMS'), [string[]]$sums, [Text.UTF8Encoding]::new($false))
    $sums
}
finally {
    Pop-Location
    foreach ($key in $savedEnv.Keys) { [Environment]::SetEnvironmentVariable($key, $savedEnv[$key], 'Process') }
}
