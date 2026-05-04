$ErrorActionPreference = "Stop"

$CliaiVersion = ""
$CliaiArch = ""
$CliaiInstallDir = ""
$CliaiPredictionSource = "Plugin"
$CliaiSkipPathUpdate = $false
$CliaiSkipShellIntegration = $false
$CliaiForce = $false

function Get-BoolEnvValue {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Name
  )

  $raw = [Environment]::GetEnvironmentVariable($Name)
  if ([string]::IsNullOrWhiteSpace($raw)) {
    return $false
  }

  switch ($raw.Trim().ToLowerInvariant()) {
    "1" { return $true }
    "true" { return $true }
    "yes" { return $true }
    "on" { return $true }
    default { return $false }
  }
}

function Invoke-CliaiWebRequest {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Uri,

    [Parameter(Mandatory = $true)]
    [string]$OutFile
  )

  $params = @{
    Uri     = $Uri
    OutFile = $OutFile
    Headers = @{
      "User-Agent" = "cliai-install-script"
    }
  }

  if ((Get-Command Invoke-WebRequest).Parameters.ContainsKey("UseBasicParsing")) {
    $params["UseBasicParsing"] = $true
  }

  Invoke-WebRequest @params
}

function Resolve-CliaiArchitecture {
  param(
    [string]$PreferredArch
  )

  if ($PreferredArch) {
    return $PreferredArch
  }

  $architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
  switch ($architecture) {
    "arm64" { return "arm64" }
    default { return "amd64" }
  }
}

function Resolve-PredictionSource {
  param(
    [string]$PreferredSource
  )

  if ([string]::IsNullOrWhiteSpace($PreferredSource)) {
    return "Plugin"
  }

  switch ($PreferredSource.Trim()) {
    "Plugin" { return "Plugin" }
    "HistoryAndPlugin" { return "HistoryAndPlugin" }
    default { throw "Unsupported prediction source: $PreferredSource" }
  }
}

function Get-CliaiWindowsAssetNames {
  param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("amd64", "arm64")]
    [string]$Architecture
  )

  switch ($Architecture) {
    "arm64" {
      return @{
        Archive  = "cliai_Windows_ARM64.zip"
        Checksum = "cliai_Windows_ARM64.zip.sha256"
      }
    }
    default {
      return @{
        Archive  = "cliai_Windows_x86_64.zip"
        Checksum = "cliai_Windows_x86_64.zip.sha256"
      }
    }
  }
}

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

function Resolve-InstalledModuleVersion {
  param(
    [Parameter(Mandatory = $true)]
    [string]$InstallRoot,

    [string]$PreferredVersion
  )

  if (-not [string]::IsNullOrWhiteSpace($PreferredVersion)) {
    return $PreferredVersion.Trim().TrimStart("v")
  }

  $moduleRoot = Join-Path $InstallRoot "modules\CliaiPredictor"
  if (Test-Path $moduleRoot) {
    $detected = Get-ChildItem -Path $moduleRoot -Directory -ErrorAction SilentlyContinue |
      Sort-Object Name -Descending |
      Select-Object -First 1 -ExpandProperty Name
    if ($detected) {
      return $detected
    }
  }

  return "dev"
}

function Install-CliaiPowerShellIntegration {
  param(
    [Parameter(Mandatory = $true)]
    [string]$InstallRoot,

    [Parameter(Mandatory = $true)]
    [string]$ExePath,

    [Parameter(Mandatory = $true)]
    [string]$PredictionSource,

    [string]$ModuleVersion
  )

  $resolvedModuleVersion = Resolve-InstalledModuleVersion -InstallRoot $InstallRoot -PreferredVersion $ModuleVersion
  $moduleSourceRoot = Join-Path $InstallRoot "modules\CliaiPredictor\$resolvedModuleVersion"
  if (-not (Test-Path (Join-Path $moduleSourceRoot "CliaiPredictor.psd1"))) {
    throw "Unable to find an installable CliaiPredictor module in the release package."
  }

  $profileDir = Join-Path $HOME "Documents\PowerShell"
  $profilePath = Join-Path $profileDir "Profile.ps1"
  if (-not (Test-Path $profileDir)) {
    New-Item -ItemType Directory -Path $profileDir -Force | Out-Null
  }

  $moduleVersionRoot = Join-Path $HOME "Documents\PowerShell\Modules\CliaiPredictor\$resolvedModuleVersion"
  New-Item -ItemType Directory -Path $moduleVersionRoot -Force | Out-Null

  foreach ($fileName in @("CliaiPredictor.dll", "CliaiPredictor.deps.json", "CliaiPredictor.psd1")) {
    $sourcePath = Join-Path $moduleSourceRoot $fileName
    if (-not (Test-Path $sourcePath)) {
      throw "Missing module file: $sourcePath"
    }
    Copy-Item -Path $sourcePath -Destination (Join-Path $moduleVersionRoot $fileName) -Force
  }

  $snippet = & $ExePath shell init powershell
  if ($LASTEXITCODE -ne 0) {
    throw "Failed to run '$ExePath shell init powershell'."
  }
  $snippet = ($snippet -join "`r`n")

  $escapedExe = $ExePath.Replace("'", "''")
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
}

function Test-UserPathContains {
  param(
    [Parameter(Mandatory = $true)]
    [string]$PathEntry
  )

  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if ([string]::IsNullOrWhiteSpace($userPath)) {
    return $false
  }

  foreach ($existing in ($userPath -split ";")) {
    if ($existing.Trim().TrimEnd("\") -ieq $PathEntry.Trim().TrimEnd("\")) {
      return $true
    }
  }

  return $false
}

function Add-ToUserPath {
  param(
    [Parameter(Mandatory = $true)]
    [string]$PathEntry
  )

  $normalizedEntry = $PathEntry.Trim().TrimEnd("\")
  if (Test-UserPathContains -PathEntry $normalizedEntry) {
    if ($env:Path -notmatch [regex]::Escape($normalizedEntry)) {
      $env:Path = "$normalizedEntry;$env:Path"
    }
    return $false
  }

  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if ([string]::IsNullOrWhiteSpace($userPath)) {
    $newUserPath = $normalizedEntry
  } else {
    $newUserPath = "$userPath;$normalizedEntry"
  }

  [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
  $env:Path = "$normalizedEntry;$env:Path"
  return $true
}

function Get-DefaultInstallDir {
  if ($env:LOCALAPPDATA) {
    return (Join-Path $env:LOCALAPPDATA "Programs\cliai")
  }

  return (Join-Path $HOME "AppData\Local\Programs\cliai")
}

function Install-Cliai {
  param(
    [string]$RequestedVersion,
    [string]$RequestedArch,
    [string]$TargetInstallDir,
    [string]$RequestedPredictionSource,
    [switch]$DoSkipPathUpdate,
    [switch]$DoSkipShellIntegration,
    [switch]$DoForce
  )

  $resolvedVersion = ""
  if ($RequestedVersion) {
    $resolvedVersion = $RequestedVersion.Trim().TrimStart("v")
    $downloadBase = "https://github.com/xjwm5685-ui/cliai/releases/download/v$resolvedVersion"
    $displayVersion = "v$resolvedVersion"
  } else {
    $downloadBase = "https://github.com/xjwm5685-ui/cliai/releases/latest/download"
    $displayVersion = "the latest release"
  }

  $resolvedArch = Resolve-CliaiArchitecture -PreferredArch $RequestedArch
  $RequestedPredictionSource = Resolve-PredictionSource -PreferredSource $RequestedPredictionSource
  $assetNames = Get-CliaiWindowsAssetNames -Architecture $resolvedArch

  if (-not $TargetInstallDir) {
    $TargetInstallDir = Get-DefaultInstallDir
  }
  $TargetInstallDir = [IO.Path]::GetFullPath($TargetInstallDir)

  $tempRoot = Join-Path ([IO.Path]::GetTempPath()) ("cliai-install-" + [guid]::NewGuid().ToString("N"))
  $archivePath = Join-Path $tempRoot $assetNames.Archive
  $checksumPath = Join-Path $tempRoot $assetNames.Checksum
  $extractRoot = Join-Path $tempRoot "extract"

  try {
    New-Item -ItemType Directory -Path $tempRoot -Force | Out-Null

    Write-Host "Downloading cliai $displayVersion for $resolvedArch..."
    Invoke-CliaiWebRequest -Uri "$downloadBase/$($assetNames.Archive)" -OutFile $archivePath
    Invoke-CliaiWebRequest -Uri "$downloadBase/$($assetNames.Checksum)" -OutFile $checksumPath

    $expectedHash = ((Get-Content $checksumPath -Raw).Trim() -split "\s+")[0].ToLowerInvariant()
    $actualHash = (Get-FileHash $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($expectedHash -ne $actualHash) {
      throw "Checksum verification failed for $($assetNames.Archive)."
    }

    if ((Test-Path $TargetInstallDir) -and $DoForce) {
      Remove-Item $TargetInstallDir -Recurse -Force
    }

    New-Item -ItemType Directory -Path $extractRoot -Force | Out-Null
    Expand-Archive -Path $archivePath -DestinationPath $extractRoot -Force

    New-Item -ItemType Directory -Path $TargetInstallDir -Force | Out-Null
    Copy-Item -Path (Join-Path $extractRoot "*") -Destination $TargetInstallDir -Recurse -Force

    $exePath = Join-Path $TargetInstallDir "cliai.exe"
    if (-not (Test-Path $exePath)) {
      throw "Installed archive did not contain cliai.exe."
    }

    $pathUpdated = $false
    if (-not $DoSkipPathUpdate) {
      $pathUpdated = Add-ToUserPath -PathEntry $TargetInstallDir
    }

    if (-not $DoSkipShellIntegration) {
      Install-CliaiPowerShellIntegration -InstallRoot $TargetInstallDir -ExePath $exePath -PredictionSource $RequestedPredictionSource -ModuleVersion $resolvedVersion
    }

    Write-Host ""
    Write-Host "cliai installed to $TargetInstallDir"
    if ($pathUpdated) {
      Write-Host "Added $TargetInstallDir to your user PATH."
    } elseif (-not $DoSkipPathUpdate) {
      Write-Host "User PATH already contains $TargetInstallDir."
    }
    if ($DoSkipShellIntegration) {
      Write-Host "Skipped shell integration."
    } else {
      Write-Host "PowerShell integration installed."
    }
    Write-Host "Open a new terminal and run: cliai version"
  }
  finally {
    if (Test-Path $tempRoot) {
      Remove-Item $tempRoot -Recurse -Force
    }
  }
}

if (-not $CliaiVersion) {
  $CliaiVersion = [Environment]::GetEnvironmentVariable("CLIAI_VERSION")
}
if (-not $CliaiArch) {
  $envArch = [Environment]::GetEnvironmentVariable("CLIAI_ARCH")
  if (-not [string]::IsNullOrWhiteSpace($envArch)) {
    $CliaiArch = $envArch
  }
}
if (-not $CliaiInstallDir) {
  $CliaiInstallDir = [Environment]::GetEnvironmentVariable("CLIAI_INSTALL_DIR")
}
if ($CliaiPredictionSource -eq "Plugin") {
  $envPredictionSource = [Environment]::GetEnvironmentVariable("CLIAI_PREDICTION_SOURCE")
  if (-not [string]::IsNullOrWhiteSpace($envPredictionSource)) {
    $CliaiPredictionSource = $envPredictionSource
  }
}
if (Get-BoolEnvValue -Name "CLIAI_SKIP_PATH_UPDATE") {
  $CliaiSkipPathUpdate = $true
}
if (Get-BoolEnvValue -Name "CLIAI_SKIP_SHELL_INTEGRATION") {
  $CliaiSkipShellIntegration = $true
}
if (Get-BoolEnvValue -Name "CLIAI_FORCE_INSTALL") {
  $CliaiForce = $true
}

if (-not (Get-BoolEnvValue -Name "CLIAI_INSTALL_NO_AUTORUN")) {
  Install-Cliai `
    -RequestedVersion $CliaiVersion `
    -RequestedArch $CliaiArch `
    -TargetInstallDir $CliaiInstallDir `
    -RequestedPredictionSource $CliaiPredictionSource `
    -DoSkipPathUpdate:$CliaiSkipPathUpdate `
    -DoSkipShellIntegration:$CliaiSkipShellIntegration `
    -DoForce:$CliaiForce
}
