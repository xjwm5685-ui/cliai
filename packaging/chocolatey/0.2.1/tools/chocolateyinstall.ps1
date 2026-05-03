$ErrorActionPreference = 'Stop'

$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$packageName = 'sanqiu-cliai'
$url64 = 'https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.1/cliai_Windows_x86_64.zip'
$checksum64 = '5AD1D13C48AAEC89C359523182847AEEB14FA8F986F3F5B6AB29A4B0A871D3B2'
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
