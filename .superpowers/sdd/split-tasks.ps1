$plan = "d:/projects/ai-dev-sop/docs/superpowers/plans/2026-09-16-mem0-graph-memory.md"
$outDir = "d:/projects/ai-dev-sop/.superpowers/sdd"
New-Item -ItemType Directory -Force -Path $outDir | Out-Null
$lines = Get-Content $plan
$tasks = @{}
$current = $null
foreach ($line in $lines) {
  if ($line -match '^### Task (\d+):') { $current = $Matches[1]; $tasks[$current] = @() }
  elseif ($current) { $tasks[$current] += $line }
}
foreach ($n in $tasks.Keys) {
  $file = Join-Path $outDir "task-$n-brief.md"
  ("# Task $n (extracted from implementation plan)`n" + ($tasks[$n] -join "`n")) | Set-Content -Path $file -Encoding UTF8
  Write-Output "wrote $file"
}
