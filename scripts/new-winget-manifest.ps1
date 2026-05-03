param(
  [Parameter(Mandatory = $true)]
  [string]$Version,

  [Parameter(Mandatory = $true)]
  [string]$X64Url,

  [Parameter(Mandatory = $true)]
  [string]$X64Sha256,

  [string]$Arm64Url,
  [string]$Arm64Sha256,

  [string]$PackageIdentifier = "Sanqiu.Cliai",
  [string]$PackageName = "cliai",
  [string]$Publisher = "Sanqiu",
  [string]$Command = "cliai",
  [string]$PublisherUrl = "https://github.com/xjwm5685-ui/cliai",
  [string]$PackageUrl = "https://github.com/xjwm5685-ui/cliai",
  [string]$License = "MIT",
  [string]$LicenseUrl = "https://github.com/xjwm5685-ui/cliai/blob/main/LICENSE"
)

$targetDir = Join-Path $PSScriptRoot "..\packaging\winget\$Version"
New-Item -ItemType Directory -Path $targetDir -Force | Out-Null

$manifestVersion = "1.10.0"

$versionFile = @"
PackageIdentifier: $PackageIdentifier
PackageVersion: $Version
DefaultLocale: zh-CN
ManifestType: version
ManifestVersion: $manifestVersion
"@

$installerLines = @(
  "PackageIdentifier: $PackageIdentifier",
  "PackageVersion: $Version",
  "InstallerType: zip",
  "NestedInstallerType: portable",
  "NestedInstallerFiles:",
  "  - RelativeFilePath: cliai.exe",
  "    PortableCommandAlias: $Command",
  "Installers:",
  "  - Architecture: x64",
  "    InstallerUrl: $X64Url",
  "    InstallerSha256: $X64Sha256"
)

if ($Arm64Url -and $Arm64Sha256) {
  $installerLines += @(
    "  - Architecture: arm64",
    "    InstallerUrl: $Arm64Url",
    "    InstallerSha256: $Arm64Sha256"
  )
}

$installerLines += @(
  "ManifestType: installer",
  "ManifestVersion: $manifestVersion"
)

$installerFile = $installerLines -join "`r`n"

$localeFile = @"
PackageIdentifier: $PackageIdentifier
PackageVersion: $Version
PackageLocale: zh-CN
Publisher: $Publisher
PublisherUrl: $PublisherUrl
PackageName: $PackageName
PackageUrl: $PackageUrl
License: $License
LicenseUrl: $LicenseUrl
ShortDescription: Hybrid command prediction CLI for PowerShell
Moniker: cliai
Tags:
  - cli
  - powershell
  - winget
  - ai
  - productivity
ManifestType: defaultLocale
ManifestVersion: $manifestVersion
"@

Set-Content -Path (Join-Path $targetDir "$PackageIdentifier.yaml") -Value $versionFile -NoNewline -Encoding utf8
Set-Content -Path (Join-Path $targetDir "$PackageIdentifier.installer.yaml") -Value $installerFile -NoNewline -Encoding utf8
Set-Content -Path (Join-Path $targetDir "$PackageIdentifier.locale.zh-CN.yaml") -Value $localeFile -NoNewline -Encoding utf8

Write-Host "winget manifests generated in $targetDir"
