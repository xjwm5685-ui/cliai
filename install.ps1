param(
  [string]$Version = "",
  [ValidateSet("amd64", "arm64")]
  [string]$Arch = "",
  [string]$InstallDir = "",
  [ValidateSet("Plugin", "HistoryAndPlugin")]
  [string]$PredictionSource = "Plugin",
  [switch]$SkipPathUpdate,
  [switch]$SkipShellIntegration,
  [switch]$Force
)

$ErrorActionPreference = "Stop"

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
    $profileInstaller = Join-Path $TargetInstallDir "scripts\install-powershell.ps1"
    if (-not (Test-Path $exePath)) {
      throw "Installed archive did not contain cliai.exe."
    }
    if (-not (Test-Path $profileInstaller)) {
      throw "Installed archive did not contain scripts\install-powershell.ps1."
    }

    $pathUpdated = $false
    if (-not $DoSkipPathUpdate) {
      $pathUpdated = Add-ToUserPath -PathEntry $TargetInstallDir
    }

    if (-not $DoSkipShellIntegration) {
      & $profileInstaller -ExeName $exePath -ModuleVersion $resolvedVersion -PredictionSource $RequestedPredictionSource
      if ($LASTEXITCODE -ne 0) {
        throw "PowerShell integration step failed."
      }
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

if (-not $Version) {
  $Version = [Environment]::GetEnvironmentVariable("CLIAI_VERSION")
}
if (-not $Arch) {
  $envArch = [Environment]::GetEnvironmentVariable("CLIAI_ARCH")
  if (-not [string]::IsNullOrWhiteSpace($envArch)) {
    $Arch = $envArch
  }
}
if (-not $InstallDir) {
  $InstallDir = [Environment]::GetEnvironmentVariable("CLIAI_INSTALL_DIR")
}
if (-not $PSBoundParameters.ContainsKey("PredictionSource")) {
  $envPredictionSource = [Environment]::GetEnvironmentVariable("CLIAI_PREDICTION_SOURCE")
  if (-not [string]::IsNullOrWhiteSpace($envPredictionSource)) {
    $PredictionSource = $envPredictionSource
  }
}
if (-not $PSBoundParameters.ContainsKey("SkipPathUpdate") -and (Get-BoolEnvValue -Name "CLIAI_SKIP_PATH_UPDATE")) {
  $SkipPathUpdate = $true
}
if (-not $PSBoundParameters.ContainsKey("SkipShellIntegration") -and (Get-BoolEnvValue -Name "CLIAI_SKIP_SHELL_INTEGRATION")) {
  $SkipShellIntegration = $true
}
if (-not $PSBoundParameters.ContainsKey("Force") -and (Get-BoolEnvValue -Name "CLIAI_FORCE_INSTALL")) {
  $Force = $true
}

if (-not (Get-BoolEnvValue -Name "CLIAI_INSTALL_NO_AUTORUN")) {
  Install-Cliai `
    -RequestedVersion $Version `
    -RequestedArch $Arch `
    -TargetInstallDir $InstallDir `
    -RequestedPredictionSource $PredictionSource `
    -DoSkipPathUpdate:$SkipPathUpdate `
    -DoSkipShellIntegration:$SkipShellIntegration `
    -DoForce:$Force
}
