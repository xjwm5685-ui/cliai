# Release Checklist

本清单用于在首次跨平台正式发布 `cliai` 前后做快速核对。

## 发布前

- 确认 `README.md`、`README_EN.md`、`docs/RELEASE.md` 已反映当前版本能力
- 确认 `go test ./...` 通过
- 确认 Windows predictor 相关改动已本地验证
- 确认 GitHub Release 所需 Secrets 已配置
- 确认 apt GPG key 已准备好，至少包含：
  - `CLIAI_APT_GPG_KEY_ARMORED` 或 `CLIAI_APT_GPG_KEY_FILE`
  - `CLIAI_APT_GPG_KEY_ID`
  - 如有口令则配置 `CLIAI_APT_GPG_PASSPHRASE`

## 本地预检查

Windows：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\check-release-env.ps1
```

Windows 需要签名时：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\check-release-env.ps1 -RequireSignature
```

Linux/macOS：

```bash
./scripts/check-release-env.sh
```

Linux/macOS 需要 apt 签名时：

```bash
./scripts/check-release-env.sh --require-apt-signature
```

建议本地再补一轮产物预演：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\release-local.ps1 -Version 0.2.2
```

```bash
./scripts/release-local.sh 0.2.2
```

## 打 tag

```powershell
git tag v0.2.2
git push origin v0.2.2
```

## Release 产物核对

- Windows:
  - `cliai_Windows_x86_64.zip`
  - `cliai_Windows_ARM64.zip`
- Unix:
  - `cliai_Linux_x86_64.tar.gz`
  - `cliai_Linux_ARM64.tar.gz`
  - `cliai_macOS_x86_64.tar.gz`
  - `cliai_macOS_ARM64.tar.gz`
- Debian / apt:
  - `cliai_<version>_amd64.deb`
  - `cliai_<version>_arm64.deb`
  - `cliai_apt_repo_<version>.tar.gz`
  - 可选：`cliai-archive-keyring.asc`

## Workflow 核对

- `build-windows` 成功
- `build-unix` 全矩阵成功
- `build-apt-repo` 成功
- Release 页面存在对应 `.sha256` 文件
- 如配置了 apt GPG key，则 `build-apt-repo` 日志中包含签名、公钥导出和 apt 校验成功输出

## apt 仓库核对

- apt repo tar.gz 解压后包含：
  - `pool/`
  - `dists/stable/Release`
  - `dists/stable/main/binary-amd64/Packages`
  - `dists/stable/main/binary-arm64/Packages`
- 如果配置了 GPG key：
  - `dists/stable/Release.gpg`
  - `dists/stable/InRelease`
  - `cliai-archive-keyring.asc`

## 发布后

- 用 Windows PowerShell 7 验证 `cliai shell install powershell`
- 用 Linux 或 WSL 验证 `.deb` 安装或 apt repo 添加流程
- 验证 `cliai-archive-keyring.asc`、`Release.gpg`、`InRelease` 与 README 中的 apt 示例一致
- 验证 README 中提到的安装命令与实际资产名一致
