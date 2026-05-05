# cliai

`cliai` is a cross-platform, local-first command prediction and completion tool for Windows, Linux, and macOS.

It is not a command runner. It is a command suggester that tries to understand your shell context:

- combines local history, built-in rules, natural-language templates, and project context
- learns from accepted suggestions over time
- labels risky commands instead of hiding them behind autocomplete
- stays local and no longer depends on external AI services

Project links:

- GitHub repository: [xjwm5685-ui/cliai](https://github.com/xjwm5685-ui/cliai)
- Community link: [Linux.do](https://linux.do)
- Chinese README: [README.md](file:///d:/sanqiu/cliai/README.md)
- Release guide: [RELEASE.md](file:///d:/sanqiu/cliai/docs/RELEASE.md)

## Highlights

- Local prediction with history, built-in command knowledge, templates, and project context
- Natural-language input for common command intents
- Project-aware ranking for `Go`, `Node`, `Python`, `Docker`, and `Git`
- Feedback learning from accepted suggestions
- Risk labels: `safe`, `caution`, `danger`
- Table, JSON, command-only, interactive, and clipboard-friendly output modes
- Shell integration for PowerShell, zsh, and bash, with native inline prediction in PowerShell and zsh

## Support Matrix

| Capability | Windows | Linux | macOS |
| --- | --- | --- | --- |
| Core CLI build and run | Supported | Supported | Supported |
| Default `history import` paths | PowerShell | bash/zsh/fish/pwsh | zsh/bash/pwsh |
| `predict` / `selftest` / `config` | Supported | Supported | Supported |
| GitHub Release artifacts | zip | tar.gz / `.deb` | tar.gz |
| Install scripts | `install.ps1` | `install.sh` / `install-unix.sh` | `install-unix.sh` |
| Real-time prediction | Native in PowerShell | Native in `zsh`, quick-accept in `bash`, also works in `pwsh` | Native in `zsh`, quick-accept in `bash`, also works in `pwsh` |

## Quick Start

Clone and build:

```powershell
git clone https://github.com/xjwm5685-ui/cliai.git
cd cliai
go build -o .\bin\cliai.exe .
```

Windows PowerShell:

```powershell
.\bin\cliai.exe history import
.\bin\cliai.exe predict "install vscode"
.\bin\cliai.exe shell install powershell
```

Linux / macOS:

```bash
go build -o ./bin/cliai .
./bin/cliai history import
./bin/cliai predict "run tests"
./bin/cliai shell install zsh
```

## Installation

### Windows one-liner

```powershell
iwr -useb https://raw.githubusercontent.com/xjwm5685-ui/cliai/main/install.ps1 | iex
```

This installer:

- detects `amd64` vs `arm64`
- downloads the latest Windows release and `.sha256`
- verifies the checksum
- adds `cliai` to the user `PATH`
- enables the PowerShell predictor integration

### Linux apt install

Add the apt source:

```bash
curl -fsSL https://raw.githubusercontent.com/xjwm5685-ui/cliai/main/install.sh | bash
```

Install the package:

```bash
sudo apt update
sudo apt install cliai
```

Then enable shell integration:

```bash
cliai shell install zsh
```

For `bash`:

```bash
cliai shell install bash
```

Default keybindings:

- `Alt+RightArrow`: accept the whole prediction
- `Alt+Shift+RightArrow`: accept one word
- `Alt+f`: more reliable word-accept fallback in many Linux and WSL2 terminals

If `pwsh` is installed:

```bash
cliai shell install powershell
```

### GitHub Release install

- Windows: download `cliai_Windows_x86_64.zip` or `cliai_Windows_ARM64.zip`, then run `.\cliai.exe shell install powershell`
- Linux/macOS: download the matching `cliai_Linux_*.tar.gz` or `cliai_macOS_*.tar.gz`, extract it, then run `./scripts/install-unix.sh`

### Distribution Status

- Available now: GitHub Release assets, the Windows bootstrap installer, and the Linux apt package `cliai`
- Not publicly distributed yet: winget `Sanqiu.Cliai` and Chocolatey `sanqiu-cliai`
- Homebrew still requires a separately maintained tap before `brew install` is available

## Main Commands

```text
cliai predict <query>
cliai predictor serve
cliai history import
cliai config show
cliai config set <key> <value>
cliai feedback show
cliai feedback accept --query <query> <command>
cliai shell init <powershell|bash|zsh>
cliai shell install <powershell|bash|zsh>
cliai shell init powershell-helpers
cliai shell install powershell-helpers
cliai selftest
cliai version
```

## `predict`

Usage:

```powershell
cliai predict [flags] <query>
```

Flags:

- `--limit <N>`
- `--shell <name>`
- `--json`
- `--cwd <path>`
- `--debug`
- `--copy`
- `--command-only`
- `--interactive`

Examples:

```powershell
cliai predict "git st"
cliai predict --limit 3 "install vscode"
cliai predict --json "search docker"
cliai predict --cwd D:\code\myapp "start"
cliai predict --debug "enter src"
cliai predict --command-only "run tests"
cliai predict --interactive --copy "enter src"
```

Default columns:

- `COMMAND`
- `SOURCE`
- `RISK`
- `WHY`

Typical `source` values:

- `template`
- `builtin`
- `context`
- `powershell-history`

## `feedback`

```powershell
cliai feedback show
cliai feedback show --json
cliai feedback accept --query "install vscode" winget install vscode
```

Use it to inspect accepted suggestions and manually reinforce preferred commands.

## `history`

```powershell
cliai history import [--file path]
```

Default history sources:

- Windows PowerShell: `%USERPROFILE%\AppData\Roaming\Microsoft\Windows\PowerShell\PSReadLine\ConsoleHost_history.txt`
- Linux/macOS PowerShell: `~/.local/share/powershell/PSReadLine/ConsoleHost_history.txt`
- bash: `~/.bash_history`
- zsh: `~/.zsh_history`
- fish: `~/.local/share/fish/fish_history`

## `config`

Show config:

```powershell
cliai config show
```

Set config values:

```powershell
cliai config set shell powershell
cliai config set history_path D:\custom\ConsoleHost_history.txt
cliai config set local.max_history 5000
```

Supported keys:

- `shell`
- `history_path`
- `local.max_history`

Example config:

```json
{
  "shell": "powershell",
  "history_path": "C:\\Users\\YOUR_NAME\\AppData\\Roaming\\Microsoft\\Windows\\PowerShell\\PSReadLine\\ConsoleHost_history.txt",
  "local": {
    "max_history": 4000
  }
}
```

Supported environment variables:

- `CLIAI_SHELL`
- `CLIAI_HISTORY_PATH`

## `shell`

Usage:

```powershell
cliai shell init powershell
cliai shell install powershell
cliai shell init powershell-helpers
cliai shell install powershell-helpers
cliai shell init zsh
cliai shell install zsh
cliai shell init bash
cliai shell install bash
```

If you only want the PowerShell helper aliases `csg` / `csi` / `csc` without the full predictor integration, you can install them separately:

```powershell
cliai shell install powershell-helpers
```

PowerShell helper aliases:

- `csg`: standard suggestions
- `csi`: interactive selection with clipboard copy
- `csc`: print only the top command

## Real-Time Shell Predictions

### PowerShell

`cliai shell install powershell` installs `CliaiPredictor` and enables real-time grey inline predictions.

Requirements:

- PowerShell `7.2+`
- PSReadLine `2.2.2+`
- `.NET SDK 8` only if the installer needs to build the predictor locally

Verify registration:

```powershell
Import-Module CliaiPredictor
(Get-PSSubsystem -Kind CommandPredictor).Implementations |
  Select-Object Id, Name, Description
```

### zsh

`zsh` gets native inline ghost text:

```bash
cliai shell install zsh
```

### bash

`bash` gets quick-accept bindings:

```bash
cliai shell install bash
```

Common bindings:

- `Alt+RightArrow`: accept the whole prediction
- `Alt+Shift+RightArrow`: accept one word
- `Alt+f`: fallback one-word accept shortcut

## How It Works

The prediction pipeline is fully local:

1. read the current working directory, shell, history cache, and feedback records
2. detect project markers such as `go.mod`, `package.json`, `pyproject.toml`, `Dockerfile`, and `.git`
3. recall candidates from built-in rules, templates, project context, and shell history
4. rank them using prefix matches, keywords, frequency, recency, and feedback bonuses
5. print results or enter interactive mode and write feedback after acceptance

## Project Layout

```text
cliai/
├─ .github/
├─ docs/
├─ internal/
│  ├─ app/
│  ├─ config/
│  ├─ feedback/
│  ├─ history/
│  ├─ predict/
│  └─ project/
├─ packaging/
├─ predictor/
├─ scripts/
├─ README.md
├─ README_EN.md
├─ go.mod
└─ main.go
```

## Development

Run locally:

```powershell
go run . version
go run . selftest --json
go run . predict "install vscode"
go run . predict --debug "run tests"
```

Build:

```powershell
go build -o cliai.exe .
```

Run tests:

```powershell
go test ./...
```

Prediction benchmark:

```powershell
go test ./internal/predict -run ^$ -bench BenchmarkPredict
```

## Release

- CI workflow: `.github/workflows/ci.yml`
- Release workflow: `.github/workflows/release.yml`

When you push a `v*` tag, the release workflow builds Windows zip packages, Linux/macOS tarballs, Linux `.deb` packages, apt metadata, and SHA256 files.

See [RELEASE.md](file:///d:/sanqiu/cliai/docs/RELEASE.md) for details.
