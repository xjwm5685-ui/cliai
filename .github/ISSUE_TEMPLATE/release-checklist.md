---
name: Release Checklist
about: Track a real GitHub Release + winget release for cliai
title: "release: v"
labels: ["release"]
assignees: []
---

## Version

- [ ] Confirm release version
- [ ] Update `CHANGELOG.md`

## Validation

- [ ] Run `go test ./...`
- [ ] Run `go test ./internal/predict -run ^$ -bench BenchmarkPredict -benchtime=1x`
- [ ] Run `cliai selftest --json`

## Signing (Optional)

- [ ] If you want a signed release, confirm `CLIAI_SIGN_PFX_BASE64` secret exists
- [ ] If you want a signed release, confirm `CLIAI_SIGN_PFX_PASSWORD` secret exists
- [ ] If you want a signed release, confirm `CLIAI_SIGN_TIMESTAMP_URL` secret exists or default is acceptable
- [ ] If you want a signed release, run `scripts/check-release-env.ps1 -RequireSignature`

## GitHub Release

- [ ] Create and push tag `vX.Y.Z`
- [ ] Verify Release workflow succeeded
- [ ] Verify `cliai_Windows_x86_64.zip` uploaded
- [ ] Verify `cliai_Windows_ARM64.zip` uploaded
- [ ] Verify `.sha256` files uploaded
- [ ] Verify `cliai version` shows injected metadata

## winget

- [ ] Compute SHA256 for x64 asset
- [ ] Compute SHA256 for ARM64 asset
- [ ] Run `scripts/new-winget-manifest.ps1`
- [ ] Run `scripts/check-winget-manifest.ps1`
- [ ] Submit manifest PR to `microsoft/winget-pkgs`

## Post-release

- [ ] Update README examples if version changed
- [ ] Verify `winget install Sanqiu.Cliai` after merge
