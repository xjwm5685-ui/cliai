param(
  [Parameter(Mandatory = $true)]
  [string]$Version,

  [switch]$RequireSignature
)

$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root "dist"
New-Item -ItemType Directory -Path $dist -Force | Out-Null

$buildDate = Get-Date -Format s
$commit = (git rev-parse --short HEAD) 2>$null
if (-not $commit) {
  $commit = "local"
}

Push-Location $root
try {
  powershell -ExecutionPolicy Bypass -File .\scripts\check-release-env.ps1 -RequireSignature:$RequireSignature
  go test ./...

  foreach ($arch in @("amd64", "arm64")) {
    $env:GOOS = "windows"
    $env:GOARCH = $arch
    $env:CGO_ENABLED = "0"
    $exe = Join-Path $dist "cliai-$arch.exe"
    go build -trimpath -ldflags "-s -w -X github.com/sanqiu/cliai/internal/app.Version=$Version -X github.com/sanqiu/cliai/internal/app.Commit=$commit -X github.com/sanqiu/cliai/internal/app.BuildDate=$buildDate" -o $exe .
    powershell -ExecutionPolicy Bypass -File .\scripts\sign-windows.ps1 -FilePath $exe -RequireSignature:$RequireSignature
  }

  Write-Host "Local release artifacts are ready in $dist"
}
finally {
  Pop-Location
}
