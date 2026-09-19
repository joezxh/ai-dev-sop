$ErrorActionPreference = 'Stop'
$p = 'd:\projects\ai-dev-sop\docs\quick-ref\mem0-manual.md'
$Utf8 = [System.Text.UTF8Encoding]::new($false)
$t = [System.IO.File]::ReadAllText($p, $Utf8)

$note = @"

> ⚡ **一键完成**：以上 5 步可由 ``scripts/mem0-setup.ps1``（Windows）/ ``scripts/mem0-setup.sh``（macOS·Linux）
> 一键完成，服务名自动约定为 ``mem0`` 并内置端到端校验（MCP 端点存活 + REST 密钥往返 + 配置语法回读）。
> 手工流程保留用于理解原理与特殊场景。
"@

$pairs = @(
    @('## 4.2 CodeBuddy 落地配置（5 步）', '## 4.2 CodeBuddy 落地配置（5 步）'),
    @('## 4.3 Qoder 落地配置（5 步）', '## 4.3 Qoder 落地配置（5 步）'),
    @('## 4.4 Cursor 落地配置（5 步）', '## 4.4 Cursor 落地配置（5 步）'),
    @('## 4.5 Codex CLI 与 IDE 落地配置（5 步）', '## 4.5 Codex CLI 与 IDE 落地配置（5 步）'),
    @('## 4.6 Claude Code 落地配置（5 步）', '## 4.6 Claude Code 落地配置（5 步）')
)

$total = 0
foreach ($pair in $pairs) {
    $n = ([regex]::Matches($t, [regex]::Escape($pair[0]))).Count
    Write-Host ("x{0,-3} {1}" -f $n, $pair[0])
    if ($n -ne 1) { Write-Host "  !! expected exactly 1, skip"; continue }
    $t = $t.Replace($pair[0], $pair[0] + $note)
    $total += 1
}

# 校验清单增加脚本校验项（F5）
$oldCheck = '- [ ] `~/.codebuddy/mcp.json` 含 `mem0` 且 `transport: "streamable-http"`，面板绿色'
$newCheck = "- [ ] 已运行 ``mem0-setup``（``-DryRun`` 预览 → 实跑）且全部输出 ✓`n" + $oldCheck
$n2 = ([regex]::Matches($t, [regex]::Escape($oldCheck))).Count
Write-Host ("checklist x{0}" -f $n2)
if ($n2 -eq 1) { $t = $t.Replace($oldCheck, $newCheck); $total += 1 }

[System.IO.File]::WriteAllText($p, $t, $Utf8)
Write-Host "TOTAL insertions: $total"

$after = [System.IO.File]::ReadAllText($p, $Utf8)
Write-Host ("verify '一键完成' : " + ([regex]::Matches($after, [regex]::Escape('一键完成'))).Count)
Write-Host ("verify 'mem0-setup' 校验项 : " + ([regex]::Matches($after, [regex]::Escape('已运行 `mem0-setup`'))).Count)