# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog and this project follows Semantic Versioning.

## [Unreleased]

### Added
- A repository-root `install-unix.sh` bootstrap script so Linux and macOS users can install the latest GitHub Release directly with `curl | bash`
- Lightweight intent classification in `internal/predict`, covering file search, package install, change-directory, read-file, run-tests, open-editor, and start-project scenarios
- Candidate eligibility gating for strong command-prefix queries such as `git st`, `docker ps`, and `go test`
- Predictor debug reporting for rejected candidates, including gate reasons in `predict --debug`
- Regression coverage for intent classification, family gating, history sanitization, richer candidate reasons, and PowerShell helper integration

### Changed
- Windows bootstrap install now supports `CLIAI_SHELL_INTEGRATION=HelpersOnly` to install only the `csg` / `csi` / `csc` PowerShell helpers
- Linux apt bootstrap install now supports optional package and shell-integration autorun through environment variables such as `CLIAI_INSTALL_PACKAGE`, `CLIAI_INSTALL_ZSH`, and `CLIAI_ENABLE_ZSH`
- Unix package-local installer now supports optional shell-integration autorun, including `zsh`, `bash`, `powershell`, and `powershell-helpers`
- History ranking now penalizes noisy long commands more aggressively and keeps unrelated high-frequency history from crowding out context/template suggestions
- Candidate explanations now prefer current-context reasons over raw history hits when the same command is recommended from multiple sources
- PowerShell helper snippets now resolve `cliai` more safely, pass through `--cwd`, support remaining arguments, and avoid noisy profile output
- File-search intent in development contexts now prefers `Select-String` / `grep` over package search for queries such as `查找 TODO`

### Fixed
- README and release documentation now point to a real top-level Unix one-liner installer instead of implying that `install-unix.sh` exists only after extracting a release archive
- History import now rejects bare URLs, concatenated commands, obvious shell output text, and other low-quality lines before they can pollute ranking
- Strong command-prefix queries such as `git st`, `git cl`, `docker ps`, and `go test` now gate out unrelated command families from top results
- Identical commands coming from both history and project context now keep the more helpful context/template source while retaining history as supporting explanation

## [0.2.9] - 2026-05-04

### Added
- Parent-directory project marker detection so predictions still recognize repository context from nested subdirectories
- Regression coverage for parent project detection and cross-shell history help text

### Fixed
- PowerShell predictor bridge now uses a more forgiving cold-start timeout and restarts after repeated read timeouts
- CLI help now describes `history import` as shell history import instead of PowerShell-only import

## [0.2.8] - 2026-05-04

### Added
- Regression tests to keep the local-first CLI help text and predictor bridge usage stable after removing remote reranking
- Updated package metadata and README positioning for the local-first product direction

### Removed
- OpenAI-compatible remote reranking implementation and related configuration

### Fixed
- PowerShell predictor bridge no longer passes the removed `--no-cloud` flag to `cliai predictor serve`

## [0.2.6] - 2026-05-04

### Fixed
- Release workflow now invokes Debian and apt helper scripts through `bash`, so Linux runners no longer depend on the executable bit being preserved in git metadata

## [0.2.5] - 2026-05-04

### Fixed
- Cross-platform release tests now validate the platform-appropriate package manager template instead of assuming `winget` on Linux and macOS runners

## [0.2.4] - 2026-05-04

### Fixed
- Cross-platform tests now pin the shell explicitly where Windows-specific install templates are expected, avoiding Linux/macOS failures caused by platform-default shell selection
- `scripts/stage-predictor-module.ps1` now uses ASCII-only status and error messages so the Windows release workflow can parse the script reliably on hosted runners

## [0.2.3] - 2026-05-04

### Fixed
- GitHub Actions `setup-go` cache now keys off `go.mod`, so CI and Release workflows work correctly in repositories that do not have a `go.sum` file
- Release retry version advanced to `v0.2.3` after the failed `v0.2.2` workflow exposed the cache configuration issue

## [0.2.2] - 2026-05-04

### Added
- Cross-platform GitHub Actions CI across Windows, Linux, and macOS
- Linux and macOS release assets, local Unix release script, and `install-unix.sh`
- Homebrew formula generation script and packaging docs
- Debian package generation, apt repository metadata generation, apt signing, public-key export, and apt asset validation scripts
- Release checklist and Unix release-environment precheck script

### Changed
- Release workflow now publishes Windows zip packages, Unix tarballs, Debian packages, apt repository archives, optional apt signatures, and optional public key assets
- Release and README documentation now describe the cross-platform install, release, and validation flow in both Chinese and English

### Fixed
- Local release scripts now use a PowerShell-compatible UTC timestamp format
- Unix local release flow now handles `go.exe` path conversion and `dpkg-deb` permission quirks in mixed Windows/bash environments
- Release documentation now reflects that apt build, signing, public-key export, and validation are part of the automated workflow

## [0.2.1] - 2026-05-03

### Fixed
- Release workflow now only runs the executable smoke test for `windows/amd64`, avoiding ARM64 execution failures on the hosted runner

## [0.2.0] - 2026-05-03

### Added
- Project context detection for Go, Node, Python, Docker and Git repositories
- Feedback learning store and accepted-command ranking bonuses
- Interactive suggestion picking, clipboard copy and command-only output modes
- PowerShell helper aliases for standard, interactive and top-command workflows
- CI workflow, release smoke checks, optional Windows code-signing support
- Self-test command, benchmarks and broader unit test coverage

### Changed
- Candidate ranking now remains fully local and only reorders existing local results
- Release pipeline now prepares signed Windows binaries and checksums
- README expanded with detailed usage, safety and release documentation

## [0.1.0] - 2026-05-03

### Added
- Initial PowerShell-first hybrid command prediction CLI
- Local history-based ranking with built-in command prediction rules
- Basic winget manifest generation and release workflow
