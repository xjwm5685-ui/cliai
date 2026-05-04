param(
  [string]$ExeName = "cliai",
  [switch]$SkipPredictorBuild,
  [string]$ModuleVersion = "",
  [ValidateSet("Plugin", "HistoryAndPlugin")]
  [string]$PredictionSource = "Plugin"
)

$ErrorActionPreference = "Stop"

function Set-MarkedBlock {
  param(
    [Parameter(Mandatory = $true)]
    [AllowEmptyString()]
    [string]$Content,

    [Parameter(Mandatory = $true)]
    [string]$StartMarker,

    [Parameter(Mandatory = $true)]
    [string]$EndMarker,

    [Parameter(Mandatory = $true)]
    [string]$Block
  )

  $pattern = "(?s)\r?\n?$([Regex]::Escape($StartMarker)).*?$([Regex]::Escape($EndMarker))\r?\n?"
  $replacement = "`r`n$StartMarker`r`n$Block`r`n$EndMarker`r`n"
  if ($Content -match $pattern) {
    return ([Regex]::Replace($Content, $pattern, $replacement)).TrimStart("`r", "`n")
  }

  if ([string]::IsNullOrWhiteSpace($Content)) {
    return "$StartMarker`r`n$Block`r`n$EndMarker`r`n"
  }

  return ($Content.TrimEnd() + $replacement)
}

$profileDir = Join-Path $HOME "Documents\PowerShell"
$profilePath = Join-Path $profileDir "Profile.ps1"
if (-not (Test-Path $profileDir)) {
  New-Item -ItemType Directory -Path $profileDir -Force | Out-Null
}

$resolvedExe = try {
  (Get-Command $ExeName -ErrorAction Stop).Source
} catch {
  $ExeName
}
$resolvedExe = [IO.Path]::GetFullPath($resolvedExe)
$exeDir = Split-Path -Parent $resolvedExe

function Resolve-ModuleVersion {
  param(
    [Parameter(Mandatory = $true)]
    [string]$PreferredVersion,

    [Parameter(Mandatory = $true)]
    [string]$ExePath,

    [Parameter(Mandatory = $true)]
    [string]$ExeDirectory
  )

  $version = $PreferredVersion.Trim()
  if ($version) {
    return $version.TrimStart("v")
  }

  $bundledModuleRoot = Join-Path $ExeDirectory "modules\CliaiPredictor"
  if (Test-Path $bundledModuleRoot) {
    $bundledVersion = Get-ChildItem -Path $bundledModuleRoot -Directory -ErrorAction SilentlyContinue |
      Sort-Object Name -Descending |
      Select-Object -First 1 -ExpandProperty Name
    if ($bundledVersion) {
      return $bundledVersion
    }
  }

  try {
    $versionOutput = & $ExePath version 2>$null
    if ($LASTEXITCODE -eq 0) {
      $match = [regex]::Match(($versionOutput -join " "), "v?(\d+\.\d+\.\d+(?:[-+][0-9A-Za-z\.-]+)?)")
      if ($match.Success) {
        return $match.Groups[1].Value
      }
    }
  } catch {
  }

  return "dev"
}

$ModuleVersion = Resolve-ModuleVersion -PreferredVersion $ModuleVersion -ExePath $resolvedExe -ExeDirectory $exeDir

$snippet = & $resolvedExe shell init powershell
if ($LASTEXITCODE -ne 0) {
  throw "无法调用 $resolvedExe shell init powershell"
}
$snippet = ($snippet -join "`r`n")

$bundledModuleCandidates = @(
  (Join-Path $exeDir "modules\CliaiPredictor\$ModuleVersion"),
  (Join-Path (Split-Path -Parent $PSScriptRoot) "modules\CliaiPredictor\$ModuleVersion")
)

$moduleSourceRoot = $null
foreach ($candidate in $bundledModuleCandidates) {
  if (Test-Path (Join-Path $candidate "CliaiPredictor.psd1")) {
    $moduleSourceRoot = $candidate
    break
  }
}

if (-not $moduleSourceRoot) {
  $repoRoot = Split-Path -Parent $PSScriptRoot
  $predictorProject = Join-Path $repoRoot "predictor\CliaiPredictor\CliaiPredictor.csproj"
  $predictorOutput = Join-Path $repoRoot "predictor\CliaiPredictor\bin\Release\net8.0"

  if (-not $SkipPredictorBuild -and (Test-Path $predictorProject)) {
    dotnet build $predictorProject -c Release | Out-Host
    if ($LASTEXITCODE -ne 0) {
      throw "无法编译 CliaiPredictor"
    }
  }

  if (Test-Path (Join-Path $predictorOutput "CliaiPredictor.psd1")) {
    $moduleSourceRoot = $predictorOutput
  }
}

if (-not $moduleSourceRoot) {
  throw "未找到可安装的 CliaiPredictor 模块。请使用包含 modules 目录的发布包，或在源码仓库中运行此脚本。"
}

$moduleVersionRoot = Join-Path $HOME "Documents\PowerShell\Modules\CliaiPredictor\$ModuleVersion"
New-Item -ItemType Directory -Path $moduleVersionRoot -Force | Out-Null

foreach ($fileName in @("CliaiPredictor.dll", "CliaiPredictor.deps.json", "CliaiPredictor.psd1")) {
  $sourcePath = Join-Path $moduleSourceRoot $fileName
  if (-not (Test-Path $sourcePath)) {
    throw "缺少模块文件: $sourcePath"
  }

  Copy-Item -Path $sourcePath -Destination (Join-Path $moduleVersionRoot $fileName) -Force
}

$escapedExe = $resolvedExe.Replace("'", "''")
$helperStart = "# >>> cliai helpers >>>"
$helperEnd = "# <<< cliai helpers <<<"
$predictorStart = "# >>> cliai predictor >>>"
$predictorEnd = "# <<< cliai predictor <<<"

$predictorSnippet = @"
if (`$PSVersionTable.PSVersion -ge [Version]'7.2.0') {
  `$env:CLIAI_EXE = '$escapedExe'
  Import-Module CliaiPredictor -Force -ErrorAction SilentlyContinue
  if (Get-Command Set-PSReadLineOption -ErrorAction SilentlyContinue) {
    Set-PSReadLineOption -PredictionSource $PredictionSource
    Set-PSReadLineOption -PredictionViewStyle InlineView
  }
  if (Get-Command Set-PSReadLineKeyHandler -ErrorAction SilentlyContinue) {
    Set-PSReadLineKeyHandler -Chord Alt+RightArrow -Function AcceptSuggestion
    Set-PSReadLineKeyHandler -Chord Alt+Shift+RightArrow -Function AcceptNextSuggestionWord
  }
}
"@

$current = ""
if (Test-Path $profilePath) {
  $current = Get-Content $profilePath -Raw
}

$updated = Set-MarkedBlock -Content $current -StartMarker $helperStart -EndMarker $helperEnd -Block $snippet
$updated = Set-MarkedBlock -Content $updated -StartMarker $predictorStart -EndMarker $predictorEnd -Block $predictorSnippet
Set-Content -Path $profilePath -Value $updated -Encoding utf8

Write-Host "已写入 PowerShell 7 Profile: $profilePath"
Write-Host "已安装 CliaiPredictor 到 $moduleVersionRoot"
Write-Host "预测源: $PredictionSource"
Write-Host "重新打开 pwsh 后可获得:"
Write-Host "  1. 实时灰字预测"
Write-Host "  2. csg/csi/csc helper 命令"
Write-Host "  3. Alt+RightArrow 接受整条预测"
Write-Host "  4. Alt+Shift+RightArrow 接受下一个预测词"
