param(
  [string]$ExeName = "cliai"
)

$profileDir = Split-Path -Parent $PROFILE.CurrentUserAllHosts
if (-not (Test-Path $profileDir)) {
  New-Item -ItemType Directory -Path $profileDir -Force | Out-Null
}

$snippet = & $ExeName shell init powershell
if ($LASTEXITCODE -ne 0) {
  throw "无法调用 $ExeName shell init powershell"
}

if (Test-Path $PROFILE.CurrentUserAllHosts) {
  $current = Get-Content $PROFILE.CurrentUserAllHosts -Raw
  if ($current -like "*Invoke-CliaiSuggestion*") {
    Write-Host "PowerShell Profile 已包含 cliai 片段，无需重复写入。"
    exit 0
  }
}

Add-Content -Path $PROFILE.CurrentUserAllHosts -Value "`r`n# cliai`r`n$snippet`r`n"
Write-Host "已写入 $($PROFILE.CurrentUserAllHosts)"
Write-Host "重新打开 PowerShell 后可使用:"
Write-Host "  csg '安装 vscode'        # 查看建议"
Write-Host "  csi 'git st'             # 交互式选择并复制"
Write-Host "  csc 'run tests'          # 只输出最佳命令"
