param(
  [switch]$RequireSignature
)

$errors = @()

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  $errors += "Go is not installed or not in PATH."
}

if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
  $errors += "Git is not installed or not in PATH."
}

if ($RequireSignature) {
  if (-not $env:CLIAI_SIGN_PFX_BASE64) {
    $errors += "Missing CLIAI_SIGN_PFX_BASE64."
  }
  if (-not $env:CLIAI_SIGN_PFX_PASSWORD) {
    $errors += "Missing CLIAI_SIGN_PFX_PASSWORD."
  }
}

if ($errors.Count -gt 0) {
  $errors | ForEach-Object { Write-Error $_ }
  exit 1
}

Write-Host "Release environment looks good."
