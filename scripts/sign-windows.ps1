param(
  [Parameter(Mandatory = $true)]
  [string]$FilePath,

  [switch]$RequireSignature
)

if (-not (Test-Path $FilePath)) {
  throw "File not found: $FilePath"
}

if (-not $env:CLIAI_SIGN_PFX_BASE64 -or -not $env:CLIAI_SIGN_PFX_PASSWORD) {
  if ($RequireSignature) {
    throw "Code signing secrets are missing."
  }
  Write-Host "No code-signing secrets provided. Skipping signing."
  exit 0
}

$tempDir = Join-Path $env:TEMP "cliai-sign"
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
$pfxPath = Join-Path $tempDir "signing-cert.pfx"
[IO.File]::WriteAllBytes($pfxPath, [Convert]::FromBase64String($env:CLIAI_SIGN_PFX_BASE64))

$password = ConvertTo-SecureString $env:CLIAI_SIGN_PFX_PASSWORD -AsPlainText -Force
$cert = Import-PfxCertificate -FilePath $pfxPath -CertStoreLocation Cert:\CurrentUser\My -Password $password
if (-not $cert) {
  throw "Failed to import signing certificate."
}

$timestampServer = $env:CLIAI_SIGN_TIMESTAMP_URL
if (-not $timestampServer) {
  $timestampServer = "http://timestamp.digicert.com"
}

$signature = Set-AuthenticodeSignature -FilePath $FilePath -Certificate $cert -TimestampServer $timestampServer
if ($signature.Status -ne "Valid") {
  throw "Signing failed: $($signature.Status) $($signature.StatusMessage)"
}

Write-Host "Successfully signed $FilePath"
