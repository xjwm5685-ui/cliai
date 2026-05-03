param(
  [Parameter(Mandatory = $true)]
  [string]$Version,

  [Parameter(Mandatory = $true)]
  [string]$X64Url,

  [Parameter(Mandatory = $true)]
  [string]$X64Sha256,

  [string]$PackageId = "sanqiu-cliai",
  [string]$PackageTitle = "cliai",
  [string]$Authors = "Sanqiu",
  [string]$Owners = "Sanqiu",
  [string]$ProjectUrl = "https://github.com/xjwm5685-ui/cliai",
  [string]$LicenseUrl = "https://github.com/xjwm5685-ui/cliai/blob/main/LICENSE",
  [string]$ProjectSourceUrl = "https://github.com/xjwm5685-ui/cliai",
  [string]$DocsUrl = "https://github.com/xjwm5685-ui/cliai/blob/main/README.md",
  [string]$ReleaseNotesUrl = "https://github.com/xjwm5685-ui/cliai/releases/tag/v$Version",
  [string]$Summary = "Hybrid command prediction CLI for PowerShell",
  [string]$Description = "cliai is a PowerShell-first command prediction CLI that combines local history, project context, user feedback and optional cloud reranking.",
  [string]$Tags = "cliai powershell cli ai productivity command"
)

$targetDir = Join-Path $PSScriptRoot "..\packaging\chocolatey\$Version"
$toolsDir = Join-Path $targetDir "tools"
New-Item -ItemType Directory -Path $toolsDir -Force | Out-Null

$nuspec = @"
<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2015/06/nuspec.xsd">
  <metadata>
    <id>$PackageId</id>
    <version>$Version</version>
    <packageSourceUrl>$ProjectSourceUrl</packageSourceUrl>
    <title>$PackageTitle</title>
    <authors>$Authors</authors>
    <owners>$Owners</owners>
    <projectUrl>$ProjectUrl</projectUrl>
    <docsUrl>$DocsUrl</docsUrl>
    <licenseUrl>$LicenseUrl</licenseUrl>
    <requireLicenseAcceptance>false</requireLicenseAcceptance>
    <projectSourceUrl>$ProjectSourceUrl</projectSourceUrl>
    <bugTrackerUrl>$ProjectSourceUrl/issues</bugTrackerUrl>
    <releaseNotes>$ReleaseNotesUrl</releaseNotes>
    <summary>$Summary</summary>
    <description>$Description</description>
    <tags>$Tags</tags>
  </metadata>
</package>
"@

$installScript = @'
$ErrorActionPreference = 'Stop'

$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$packageName = '{PACKAGE_ID}'
$url64 = '{URL64}'
$checksum64 = '{CHECKSUM64}'
$checksumType64 = 'sha256'

Install-ChocolateyZipPackage `
  -PackageName $packageName `
  -UnzipLocation $toolsDir `
  -Url64bit $url64 `
  -Checksum64 $checksum64 `
  -ChecksumType64 $checksumType64

$exePath = Join-Path $toolsDir 'cliai.exe'
if (-not (Test-Path $exePath)) {
  throw "Expected executable not found after extraction: $exePath"
}

Install-BinFile -Name 'cliai' -Path $exePath
'@
$installScript = $installScript.Replace('{PACKAGE_ID}', $PackageId)
$installScript = $installScript.Replace('{URL64}', $X64Url)
$installScript = $installScript.Replace('{CHECKSUM64}', $X64Sha256)

$beforeModifyScript = @'
$ErrorActionPreference = 'Stop'
Uninstall-BinFile -Name 'cliai'
'@

$uninstallScript = @'
$ErrorActionPreference = 'Stop'
Uninstall-BinFile -Name 'cliai'
'@

$verification = @"
VERIFICATION
Verification is intended to assist the Chocolatey moderators and community
in verifying that this package matches the upstream project source and release.

1. Download the upstream release asset:
   $X64Url

2. Compute the SHA256 checksum of the downloaded file.

3. Confirm the checksum matches:
   $X64Sha256

4. Confirm the package metadata points to:
   $ProjectUrl
"@

Set-Content -Path (Join-Path $targetDir "$PackageId.nuspec") -Value $nuspec -Encoding utf8
Set-Content -Path (Join-Path $toolsDir "chocolateyinstall.ps1") -Value $installScript -Encoding utf8
Set-Content -Path (Join-Path $toolsDir "chocolateybeforemodify.ps1") -Value $beforeModifyScript -Encoding utf8
Set-Content -Path (Join-Path $toolsDir "chocolateyuninstall.ps1") -Value $uninstallScript -Encoding utf8
Set-Content -Path (Join-Path $toolsDir "VERIFICATION.txt") -Value $verification -Encoding utf8

Write-Host "Chocolatey package generated in $targetDir"
