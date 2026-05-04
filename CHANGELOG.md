# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog and this project follows Semantic Versioning.

## [Unreleased]

### Added
- Chocolatey package generation script and package layout for publishing `cliai` as `sanqiu-cliai`

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
- Cloud reranking now only reorders existing local candidates
- Release pipeline now prepares signed Windows binaries and checksums
- README expanded with detailed usage, safety and release documentation

## [0.1.0] - 2026-05-03

### Added
- Initial PowerShell-first hybrid command prediction CLI
- Local history-based ranking and OpenAI-compatible cloud reranking
- Basic winget manifest generation and release workflow
