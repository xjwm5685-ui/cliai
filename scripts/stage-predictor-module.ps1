param(
  [Parameter(Mandatory = $true)]
  [string]$DestinationRoot,

  [Parameter(Mandatory = $true)]
  [string]$ModuleVersion,

  [string]$Configuration = "Release"
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$predictorProject = Join-Path $repoRoot "predictor\CliaiPredictor\CliaiPredictor.csproj"
$predictorOutput = Join-Path $repoRoot "predictor\CliaiPredictor\bin\$Configuration\net8.0"
$moduleOutput = Join-Path $DestinationRoot "modules\CliaiPredictor\$ModuleVersion"
$scriptOutput = Join-Path $DestinationRoot "scripts"

dotnet build $predictorProject -c $Configuration | Out-Host
if ($LASTEXITCODE -ne 0) {
  throw "无法编译 CliaiPredictor"
}

New-Item -ItemType Directory -Path $moduleOutput -Force | Out-Null
New-Item -ItemType Directory -Path $scriptOutput -Force | Out-Null

foreach ($fileName in @("CliaiPredictor.dll", "CliaiPredictor.deps.json", "CliaiPredictor.psd1")) {
  $source = Join-Path $predictorOutput $fileName
  if (-not (Test-Path $source)) {
    throw "缺少 predictor 输出文件: $source"
  }

  Copy-Item -Path $source -Destination (Join-Path $moduleOutput $fileName) -Force
}

Copy-Item -Path (Join-Path $repoRoot "scripts\install-powershell.ps1") -Destination (Join-Path $scriptOutput "install-powershell.ps1") -Force

Write-Host "已暂存 predictor 模块到 $moduleOutput"
