param(
  [Parameter(Mandatory = $true)]
  [string]$Version,

  [switch]$RequireSignature
)

$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root "dist"
New-Item -ItemType Directory -Path $dist -Force | Out-Null

$buildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$commit = (git rev-parse --short HEAD) 2>$null
if (-not $commit) {
  $commit = "local"
}

function New-Sha256File {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Path
  )

  $artifactName = Split-Path -Leaf $Path
  $hash = Get-FileHash $Path -Algorithm SHA256
  "$($hash.Hash)  $artifactName" | Set-Content "$Path.sha256"
}

function New-AssetDirectory {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Path
  )

  if (Test-Path $Path) {
    Remove-Item $Path -Recurse -Force
  }

  New-Item -ItemType Directory -Path $Path -Force | Out-Null
}

Push-Location $root
try {
  $checkReleaseArgs = @(
    "-NoProfile",
    "-ExecutionPolicy", "Bypass",
    "-File", ".\scripts\check-release-env.ps1"
  )
  if ($RequireSignature) {
    $checkReleaseArgs += "-RequireSignature"
  }
  powershell @checkReleaseArgs
  go test ./...

  foreach ($target in @(
    @{ Arch = "amd64"; AssetDir = "windows-amd64"; Artifact = "cliai_Windows_x86_64.zip"; Validate = $true },
    @{ Arch = "arm64"; AssetDir = "windows-arm64"; Artifact = "cliai_Windows_ARM64.zip"; Validate = $false }
  )) {
    $arch = $target.Arch
    $env:GOOS = "windows"
    $env:GOARCH = $arch
    $env:CGO_ENABLED = "0"
    $assetRoot = Join-Path $dist $target.AssetDir
    $artifactPath = Join-Path $dist $target.Artifact
    New-AssetDirectory -Path $assetRoot
    if (Test-Path $artifactPath) {
      Remove-Item $artifactPath -Force
    }
    if (Test-Path "$artifactPath.sha256") {
      Remove-Item "$artifactPath.sha256" -Force
    }

    $exe = Join-Path $assetRoot "cliai.exe"
    go build -trimpath -ldflags "-s -w -X github.com/sanqiu/cliai/internal/app.Version=$Version -X github.com/sanqiu/cliai/internal/app.Commit=$commit -X github.com/sanqiu/cliai/internal/app.BuildDate=$buildDate" -o $exe .
    $signArgs = @(
      "-NoProfile",
      "-ExecutionPolicy", "Bypass",
      "-File", ".\scripts\sign-windows.ps1",
      "-FilePath", $exe
    )
    if ($RequireSignature) {
      $signArgs += "-RequireSignature"
    }
    powershell @signArgs
    powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\stage-predictor-module.ps1 -DestinationRoot $assetRoot -ModuleVersion $Version

    Compress-Archive -Path (Join-Path $assetRoot "*") -DestinationPath $artifactPath -Force
    New-Sha256File -Path $artifactPath

    if ($target.Validate) {
      powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-release.ps1 -ExePath $exe
    }
  }

  Write-Host "Local release artifacts are ready in $dist"
}
finally {
  Pop-Location
}
