#Requires -Version 5.1
$ErrorActionPreference = 'Stop'

$candidates = @(
    "$env:USERPROFILE\go\bin\engram-lite.exe"
    if ($env:GOPATH) { "$env:GOPATH\bin\engram-lite.exe" }
    "$env:LOCALAPPDATA\Programs\engram-lite\engram-lite.exe"
)

$binary = $candidates | Where-Object { $_ -and (Test-Path $_) } | Select-Object -First 1

if (-not $binary) {
    $inPath = Get-Command engram-lite -ErrorAction SilentlyContinue
    if ($inPath) { $binary = $inPath.Source }
}

if (-not $binary) {
    [Console]::Error.WriteLine("engram-lite binary not found.")
    [Console]::Error.WriteLine("Install with: go install github.com/yuberalberto/engram-lite@latest")
    exit 1
}

& $binary mcp --tools=agent
exit $LASTEXITCODE
