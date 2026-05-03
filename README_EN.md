# cliai

`cliai` is a PowerShell-first command prediction CLI for Windows.

It combines:

- natural-language intent matching
- local PowerShell history
- project-context detection
- lightweight feedback learning
- optional cloud reranking with OpenAI-compatible APIs

The goal is not just prefix completion, but better command suggestions for real workflows.

## Highlights

- PowerShell-first command prediction
- Context-aware suggestions for Go, Node, Python, Docker, and Git projects
- Feedback learning from accepted suggestions
- Risk labels for suggested commands
- Interactive selection mode
- Clipboard copy support
- CI, Release, signing, and winget helper scripts included

## Quick Start

```powershell
go build -o cliai.exe .
.\cliai.exe history import
.\cliai.exe predict "install vscode"
.\cliai.exe predict --interactive "git st"
.\cliai.exe shell init powershell
```

## Main Commands

```text
cliai predict <query>
cliai history import
cliai config show
cliai config set <key> <value>
cliai feedback show
cliai feedback accept --query <query> <command>
cliai shell init powershell
cliai selftest
cliai version
```

## Release Docs

- Detailed Chinese README: [README.md](file:///d:/sanqiu/cli%20ai/README.md)
- Release guide: [RELEASE.md](file:///d:/sanqiu/cli%20ai/docs/RELEASE.md)

## Repository

- GitHub: [xjwm5685-ui/cliai](https://github.com/xjwm5685-ui/cliai)
- Expected winget package: `Sanqiu.Cliai`
