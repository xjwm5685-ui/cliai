# cliai

`cliai` is a cross-platform command prediction CLI for Windows, Linux, and macOS, with first-class PowerShell inline prediction support.

It combines:

- natural-language intent matching
- local shell history from PowerShell, bash, zsh, and fish
- project-context detection
- lightweight feedback learning
- optional cloud reranking with OpenAI-compatible APIs

The goal is not just prefix completion, but high-quality command suggestions for real workflows.

## Highlights

- Cross-platform CLI core for Windows, Linux, and macOS
- PowerShell real-time grey inline predictions through `CliaiPredictor`
- Context-aware suggestions for Go, Node, Python, Docker, and Git projects
- Feedback learning from accepted suggestions
- Risk labels for suggested commands
- Interactive selection mode and clipboard copy support
- GitHub Actions CI on Windows, Linux, and macOS
- Release assets for Windows zip packages, Unix tarballs, Debian packages, and apt repository archives

## Support Matrix

| Capability | Windows | Linux | macOS |
| --- | --- | --- | --- |
| Core CLI build and run | Supported | Supported | Supported |
| Default `history import` paths | PowerShell | bash/zsh/fish/pwsh | zsh/bash/pwsh |
| `predict` / `selftest` / `config` | Supported | Supported | Supported |
| GitHub Release package | zip | tar.gz | tar.gz |
| Install helper | `install-powershell.ps1` | `install-unix.sh` | `install-unix.sh` |
| PowerShell inline prediction | Supported | Supported when `pwsh` is installed | Supported when `pwsh` is installed |
| Native package channel | winget / Chocolatey | `.deb` and apt repo scripts included | Homebrew formula script included |

## Quick Start

Windows PowerShell:

```powershell
go build -o .\bin\cliai.exe .
.\bin\cliai.exe history import
.\bin\cliai.exe predict "install vscode"
.\bin\cliai.exe shell install powershell
```

Linux / macOS:

```bash
go build -o ./bin/cliai .
./bin/cliai history import
./bin/cliai predict "install vscode"
```

## Installation

Windows one-liner:

```powershell
iwr -useb https://raw.githubusercontent.com/xjwm5685-ui/cliai/main/install.ps1 | iex
```

The bootstrap installer detects `amd64` vs `arm64`, downloads the latest Windows release zip and `.sha256`, verifies the checksum, adds `cliai` to the user `PATH`, and enables the PowerShell predictor integration.

From GitHub Release:

- Windows: download `cliai_Windows_x86_64.zip` or `cliai_Windows_ARM64.zip`, then run `.\cliai.exe shell install powershell`
- Linux/macOS: download the matching `cliai_Linux_*.tar.gz` or `cliai_macOS_*.tar.gz`, extract it, then run `./scripts/install-unix.sh`

If `pwsh` is available on Linux/macOS, `install-unix.sh` will remind you to run:

```bash
cliai shell install powershell
```

Planned Homebrew install command:

```bash
brew install <your-tap>/cliai
```

Planned apt install command:

```bash
sudo apt install cliai
```

Current apt status:

- `release.yml` already builds `.deb` packages, apt repository metadata, optional signatures, optional public key export, and apt asset validation
- end-user `apt install cliai` still depends on publishing the repository at a real public URL

## Local Release Builds

- Windows precheck: `powershell -ExecutionPolicy Bypass -File .\scripts\check-release-env.ps1`
- Windows local release: `powershell -ExecutionPolicy Bypass -File .\scripts\release-local.ps1 -Version 0.2.1`
- Linux/macOS precheck: `./scripts/check-release-env.sh`
- Linux/macOS local release: `./scripts/release-local.sh 0.2.1`

If you require apt signing locally, run:

```bash
./scripts/check-release-env.sh --require-apt-signature
./scripts/release-local.sh 0.2.1 --require-apt-signature
```

## Real-Time PowerShell Predictions

`cliai` supports real-time grey inline predictions in PowerShell through the binary predictor module `CliaiPredictor`.

Requirements:

- PowerShell `7.2+`
- PSReadLine `2.2.2+`
- `.NET SDK 8` only when the installer needs to build the predictor locally

Install from source:

```powershell
go build -o .\bin\cliai.exe .
.\bin\cliai.exe shell install powershell
```

The installer:

- uses the bundled predictor module when it exists
- otherwise builds `predictor\CliaiPredictor`
- installs the module to your PowerShell user module path
- imports `CliaiPredictor` from your profile
- enables `Plugin` prediction source by default
- adds `Alt+RightArrow` and `Alt+Shift+RightArrow` key handlers

Verify registration:

```powershell
Import-Module CliaiPredictor
(Get-PSSubsystem -Kind CommandPredictor).Implementations |
  Select-Object Id, Name, Description
```

## Main Commands

```text
cliai predict <query>
cliai predictor serve
cliai history import
cliai config show
cliai config set <key> <value>
cliai feedback show
cliai feedback accept --query <query> <command>
cliai shell init powershell
cliai shell install powershell
cliai selftest
cliai version
```

## Package Channels

- Expected winget package: `Sanqiu.Cliai`
- Expected Chocolatey package: `sanqiu-cliai`
- Planned Homebrew formula: `cliai`
- Planned apt package name: `cliai`

Chocolatey packaging files can be generated with:

```powershell
.\scripts\new-chocolatey-package.ps1 `
  -Version 0.2.1 `
  -X64Url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.1/cliai_Windows_x86_64.zip `
  -X64Sha256 YOUR_X64_SHA256
```

Homebrew formula files can be generated with:

```bash
./scripts/new-homebrew-formula.sh \
  --version 0.2.1 \
  --darwin-amd64-url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.1/cliai_macOS_x86_64.tar.gz \
  --darwin-amd64-sha256 YOUR_MACOS_X64_SHA256 \
  --darwin-arm64-url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.1/cliai_macOS_ARM64.tar.gz \
  --darwin-arm64-sha256 YOUR_MACOS_ARM64_SHA256
```

Debian package staging trees and `.deb` files can be generated with:

```bash
./scripts/new-deb-package.sh --version 0.2.1 --arch amd64
./scripts/new-deb-package.sh --version 0.2.1 --arch arm64
```

APT repository metadata can be generated with:

```bash
./scripts/new-apt-repo.sh \
  --repo-root ./packaging/apt/0.2.1 \
  --deb ./packaging/deb/0.2.1/amd64/cliai_0.2.1_amd64.deb \
  --deb ./packaging/deb/0.2.1/arm64/cliai_0.2.1_arm64.deb
```

APT repository signing files can be generated with:

```bash
./scripts/sign-apt-repo.sh \
  --repo-root ./packaging/apt/0.2.1 \
  --require-signature
```

In the tagged GitHub Release workflow, these steps are already automated when the apt GPG key secrets are provided.

The public apt key can be exported with:

```bash
./scripts/export-apt-public-key.sh --output ./dist/cliai-archive-keyring.asc
```

If you publish the apt repository at a URL such as `https://example.com/cliai/apt`, end users can install with:

```bash
sudo install -d -m 0755 /etc/apt/keyrings
curl -fsSL https://example.com/cliai/apt/cliai-archive-keyring.asc | sudo tee /etc/apt/keyrings/cliai-archive-keyring.asc >/dev/null
echo "deb [signed-by=/etc/apt/keyrings/cliai-archive-keyring.asc] https://example.com/cliai/apt stable main" | sudo tee /etc/apt/sources.list.d/cliai.list >/dev/null
sudo apt update
sudo apt install cliai
```

## Release Docs

- Chinese README: [README.md](file:///d:/sanqiu/cli%20ai/README.md)
- Release guide: [RELEASE.md](file:///d:/sanqiu/cli%20ai/docs/RELEASE.md)
- GitHub repository: [xjwm5685-ui/cliai](https://github.com/xjwm5685-ui/cliai)
