#Requires -Version 5.1
$ErrorActionPreference = 'Continue'

$rawPayload = @($input) | Out-String
$payload = @{}
if (-not [string]::IsNullOrWhiteSpace($rawPayload)) {
    try { $payload = $rawPayload | ConvertFrom-Json -ErrorAction SilentlyContinue } catch {}
}

$cwd = if ($payload.cwd) { $payload.cwd } else { Get-Location | Select-Object -ExpandProperty Path }
$sessionId = $payload.session_id
$projectName = if ($cwd) { Split-Path -Leaf $cwd } else { 'unnamed-project' }

Write-Output @"
## engram-lite Memory Protocol — Session Started

Call mem_context now to load prior session memory for project: $projectName
Working directory: $cwd
Session ID: $sessionId
"@

exit 0
