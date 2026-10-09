param(
    [Parameter(Mandatory)][string]$Exe,
    [Parameter(Mandatory)][string]$Out
)
$ErrorActionPreference = 'Stop'

function Invoke-Probe([string]$Name) {
    $file = Join-Path $Out "$Name.json"
    # mamori is a gui subsystem program, the shell does not wait for it on its own
    $p = Start-Process -FilePath $Exe -ArgumentList '--probe', $Name, '-out', $file -Wait -PassThru
    if ($p.ExitCode -ne 0) { throw "probe $Name exited with code $($p.ExitCode)" }
    Get-Content $file -Raw | ConvertFrom-Json
}

$summary = Invoke-Probe 'all'
foreach ($id in 'internet', 'inventory', 'firewall', 'antivirus') {
    if ($summary.results.check -notcontains $id) { throw "no result for $id" }
}
$codes = @($summary.results | ForEach-Object { $_.code; $_.findings.code })
if ($codes -contains 'common.not_implemented') { throw 'a check is still a scaffold' }

$rule = Invoke-Probe 'firewall-rule'
$left = @(Get-NetFirewallRule -DisplayName 'mamori-probe-*' -ErrorAction SilentlyContinue)
if ($left.Count -gt 0) { throw "rules left behind: $($left.DisplayName -join ', ')" }

$firewall = $summary.results | Where-Object check -eq 'firewall'
if ($firewall.findings.code -contains 'fw.state.enforcing' -and $rule.findings.code -notcontains 'fw.rule.blocked') {
    throw 'the firewall enforces rules on this runner, but the rule test did not see the block'
}

$lines = @('| check | status | verdict |', '|---|---|---|')
foreach ($r in $summary.results) { $lines += "| $($r.check) | $($r.status) | $($r.code) |" }
$lines += "| firewall-rule | $($rule.status) | $($rule.code) |"
$lines += ''
$lines += "Overall: $($summary.status), $($summary.code). Runners have Defender switched off, so the antivirus check is expected to fail here."
if ($env:GITHUB_STEP_SUMMARY) { $lines | Out-File -Append -Encoding utf8 $env:GITHUB_STEP_SUMMARY }
$lines | Write-Output
