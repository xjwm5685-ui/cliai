param(
  [Parameter(Mandatory = $true)]
  [string]$Directory
)

if (-not (Test-Path $Directory)) {
  throw "Directory not found: $Directory"
}

$files = Get-ChildItem -Path $Directory -File -Filter *.yaml
if ($files.Count -lt 3) {
  throw "Expected at least 3 manifest files in $Directory"
}

$requiredPatterns = @(
  "PackageIdentifier:",
  "PackageVersion:",
  "ManifestType:",
  "ManifestVersion:"
)

foreach ($file in $files) {
  $content = Get-Content $file.FullName -Raw
  foreach ($pattern in $requiredPatterns) {
    if ($content -notmatch [regex]::Escape($pattern)) {
      throw "Manifest validation failed for $($file.Name): missing $pattern"
    }
  }
}

Write-Host "Manifest validation passed for $Directory"
