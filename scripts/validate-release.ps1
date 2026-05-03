param(
  [Parameter(Mandatory = $true)]
  [string]$ExePath
)

if (-not (Test-Path $ExePath)) {
  throw "Executable not found: $ExePath"
}

$versionOutput = & $ExePath version
if ($LASTEXITCODE -ne 0) {
  throw "Version command failed."
}

$selfTestOutput = & $ExePath selftest --json
if ($LASTEXITCODE -ne 0) {
  throw "Selftest command failed."
}

if (-not $versionOutput) {
  throw "Version output is empty."
}

if (-not $selfTestOutput) {
  throw "Selftest output is empty."
}

Write-Host "Release validation passed."
