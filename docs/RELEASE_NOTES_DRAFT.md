# Release Notes Draft

## Highlights

- Strong command-prefix queries are now much cleaner and more reliable.
- Natural-language queries keep context-aware explanations and safer defaults.
- Predictor debug output now explains why low-quality candidates were filtered out.

## What's New

- Added a lightweight intent layer for file search, package install, directory navigation, file reading, project start, and test-running queries.
- Added candidate gating for explicit command-prefix queries such as `git st`, `git cl`, `docker ps`, and `go test`.
- Added richer candidate explanations so the top result can explain the current recommendation context, not just its historical frequency.
- Added debug reporting for gated candidates in `cliai predict --debug`.

## Ranking Improvements

- `git st` now stays focused on `git status` and no longer leaks unrelated `git clone`, `go install`, `winget install`, or bare URL history into top results.
- `git cl` still allows `git clone ...` history as expected.
- `docker ps` now prioritizes `docker ps` cleanly and keeps non-docker history out of the top results.
- `go test` now prefers `go test ./...` and keeps the source focused on current Go project context even when the same command also exists in local history.
- Development-oriented file search queries such as `查找 TODO` now prefer `Select-String` / `grep` ahead of package search.

## History Quality

- Rejected bare URLs such as `https://github.com/foo/bar.git`.
- Rejected concatenated commands such as `winget install starshipwinget install starship`.
- Rejected obvious shell output text and low-quality non-command lines before they can affect ranking.

## PowerShell

- PowerShell helper snippets now pass through the current working directory, support remaining arguments, resolve `cliai` more safely, and avoid noisy profile output.
- Predictor debug output now helps explain both accepted candidates and gated candidates.

## Recommended Demo Queries

```powershell
.\cliai.exe predict "git st"
.\cliai.exe predict "git cl"
.\cliai.exe predict "docker ps"
.\cliai.exe predict "go test"
.\cliai.exe predict "查找 TODO"
.\cliai.exe predict "打开 README"
.\cliai.exe predict "安装一下 vscode"
.\cliai.exe predict --debug "git st"
```
