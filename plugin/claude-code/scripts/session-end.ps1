#Requires -Version 5.1
$ErrorActionPreference = 'SilentlyContinue'

$payload = @{}
$input | ForEach-Object {
    if ($_) { try { $payload = $_ | ConvertFrom-Json -ErrorAction SilentlyContinue } catch {} }
}

if ($payload -is [object] -and $payload.sessionContext -and $payload.sessionContext.mem_session_summary_called -eq $true) {
    exit 0
}

@{
    hookSpecificOutput = @{
        additionalContext = 'Session end detected. Call mem_session_summary to persist session summary.'
    }
} | ConvertTo-Json -Depth 5

exit 0
