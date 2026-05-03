# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog and this project follows Semantic Versioning.

## [Unreleased]

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
