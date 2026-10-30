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
    New-Item -ItemType Directory -Path dist -Force | Out-Null
    foreach ($target in @(@('windows','amd64','rockdev-windows-amd64.exe'), @('linux','amd64','rockdev-linux-amd64'), @('linux','arm64','rockdev-linux-arm64'))) {
        $env:GOOS = $target[0]
        $env:GOARCH = $target[1]
        & $GoPath build -trimpath -ldflags '-s -w' -o (Join-Path 'dist' $target[2]) ./cmd/rockdev
        if ($LASTEXITCODE -ne 0) { throw ('Build failed: ' + $target[2]) }
    }
    Compress-Archive -Path cmd,internal,docs,scripts,go.mod,README.md -DestinationPath 'dist\rockdev-0.1.0-source.zip' -Force
    $artifactNames = @('rockdev-linux-amd64','rockdev-linux-arm64','rockdev-windows-amd64.exe','rockdev-0.1.0-source.zip')
    $sums = foreach ($name in $artifactNames) { $hash = (Get-FileHash -LiteralPath (Join-Path 'dist' $name) -Algorithm SHA256).Hash.ToLowerInvariant(); "$hash  $name" }
    [IO.File]::WriteAllLines((Join-Path $projectRoot 'dist\SHA256SUMS'), [string[]]$sums, [Text.UTF8Encoding]::new($false))
    $sums
}
finally {
    Pop-Location
    foreach ($key in $savedEnv.Keys) { [Environment]::SetEnvironmentVariable($key, $savedEnv[$key], 'Process') }
}
