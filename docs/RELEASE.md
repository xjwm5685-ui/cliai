# Release Guide

本文件用于真实发布 `cliai` 到 GitHub Release、winget、Chocolatey、Homebrew tap，以及 Debian 包。

## 1. 需要的 GitHub Secrets

如果你想做已签名发布，可以提供以下 Secrets：

- `CLIAI_SIGN_PFX_BASE64`
- `CLIAI_SIGN_PFX_PASSWORD`
- `CLIAI_SIGN_TIMESTAMP_URL`
- `CLIAI_APT_GPG_KEY_FILE`
- `CLIAI_APT_GPG_KEY_ARMORED`
- `CLIAI_APT_GPG_KEY_ID`
- `CLIAI_APT_GPG_PASSPHRASE`

说明：

- `CLIAI_SIGN_PFX_BASE64`：将 `.pfx` 证书文件转为 Base64 后填入
- `CLIAI_SIGN_PFX_PASSWORD`：PFX 证书密码
- `CLIAI_SIGN_TIMESTAMP_URL`：时间戳服务地址，不提供时本地脚本默认使用 `http://timestamp.digicert.com`
- `CLIAI_APT_GPG_KEY_FILE`：apt 仓库签名私钥文件路径，适合自托管 runner
- `CLIAI_APT_GPG_KEY_ARMORED`：ASCII armored GPG 私钥内容，适合 GitHub Secrets
- `CLIAI_APT_GPG_KEY_ID`：签名使用的 key id 或 fingerprint
- `CLIAI_APT_GPG_PASSPHRASE`：GPG 私钥口令

## 2. 生成 Base64

PowerShell 示例：

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes("D:\certs\cliai-signing.pfx")) | Set-Clipboard
```

然后把剪贴板内容粘贴到 `CLIAI_SIGN_PFX_BASE64`。

## 3. 本地检查发布环境

无签名发布：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\check-release-env.ps1
```

需要签名时：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\check-release-env.ps1 -RequireSignature
```

Linux/macOS 无 apt 签名本地预检查：

```bash
./scripts/check-release-env.sh
```

Linux/macOS 需要 apt 签名时：

```bash
./scripts/check-release-env.sh --require-apt-signature
```

## 4. 本地构建

Windows 无签名本地发布：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\release-local.ps1 -Version 0.2.5
```

Windows 需要签名时：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\release-local.ps1 -Version 0.2.5 -RequireSignature
```

Linux/macOS 本地发布：

```bash
./scripts/check-release-env.sh
./scripts/release-local.sh 0.2.5
```

说明：

- `release-local.ps1` 生成 Windows zip、SHA256，并附带 `CliaiPredictor`
- `release-local.sh` 生成 Linux/macOS tar.gz、SHA256，并附带 `install-unix.sh`
- 在 Linux 且本机具备 `dpkg-deb` 时，还会额外生成 `.deb`、apt repo、公钥与 apt 校验结果

## 5. 推送正式 tag

```powershell
git tag v0.2.5
git push origin v0.2.5
```

Release workflow 会：

1. 在多平台 runner 上运行测试
2. 构建 `windows/amd64` 和 `windows/arm64` zip 包
3. 构建 `linux/amd64`、`linux/arm64`、`darwin/amd64`、`darwin/arm64` tar.gz 包
4. 额外构建 Linux `amd64` / `arm64` 的 `.deb`
5. 自动生成 apt 仓库元数据归档
6. 为所有二进制注入版本信息
7. 如提供证书则仅对 Windows 可执行文件执行代码签名
8. 如提供 GPG 私钥则为 apt 仓库生成 `Release.gpg` 和 `InRelease`
9. 为 Windows 包暂存 `CliaiPredictor` PowerShell 模块和 `scripts/install-powershell.ps1`
10. 为 Linux/macOS 包暂存 `scripts/install-unix.sh`
11. 生成 sha256
12. 自动校验 `.deb`、apt repo metadata、签名文件和公钥文件结构
13. 对可在当前 runner 上直接执行的发布包运行 smoke test
14. 上传 Release 资产

Windows 发布包解压后，用户可直接执行：

```powershell
.\cliai.exe shell install powershell
```

来一键启用实时灰字预测。

Linux/macOS 发布包解压后，用户可执行：

```bash
./scripts/install-unix.sh
```

如果机器上已经有 `pwsh`，再继续执行：

```bash
cliai shell install powershell
```

即可启用 PowerShell 实时预测。

## 6. 生成 winget manifest

```powershell
.\scripts\new-winget-manifest.ps1 `
  -Version 0.2.3 `
  -X64Url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.3/cliai_Windows_x86_64.zip `
  -X64Sha256 YOUR_X64_SHA256 `
  -Arm64Url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.3/cliai_Windows_ARM64.zip `
  -Arm64Sha256 YOUR_ARM64_SHA256
```

## 7. 校验 winget manifest

```powershell
.\scripts\check-winget-manifest.ps1 -Directory .\packaging\winget\0.2.3
```

## 8. 提交到 winget-pkgs

将生成的以下文件提交到 `microsoft/winget-pkgs`：

- `Sanqiu.Cliai.yaml`
- `Sanqiu.Cliai.installer.yaml`
- `Sanqiu.Cliai.locale.zh-CN.yaml`

## 9. 生成 Chocolatey 包目录

```powershell
.\scripts\new-chocolatey-package.ps1 `
  -Version 0.2.3 `
  -X64Url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.3/cliai_Windows_x86_64.zip `
  -X64Sha256 YOUR_X64_SHA256
```

## 10. 打包 Chocolatey 包

```powershell
cd .\packaging\chocolatey\0.2.3
choco pack
```

## 11. 推送到 Chocolatey 社区源

```powershell
choco push .\sanqiu-cliai.0.2.3.nupkg --source https://push.chocolatey.org/ --api-key YOUR_API_KEY
```

## 12. 生成 Homebrew Formula

```bash
./scripts/new-homebrew-formula.sh \
  --version 0.2.3 \
  --darwin-amd64-url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.3/cliai_macOS_x86_64.tar.gz \
  --darwin-amd64-sha256 YOUR_MACOS_X64_SHA256 \
  --darwin-arm64-url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.3/cliai_macOS_ARM64.tar.gz \
  --darwin-arm64-sha256 YOUR_MACOS_ARM64_SHA256 \
  --linux-amd64-url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.3/cliai_Linux_x86_64.tar.gz \
  --linux-amd64-sha256 YOUR_LINUX_X64_SHA256 \
  --linux-arm64-url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.3/cliai_Linux_ARM64.tar.gz \
  --linux-arm64-sha256 YOUR_LINUX_ARM64_SHA256
```

脚本默认会生成：

- `packaging/homebrew/0.2.3/cliai.rb`

后续步骤：

- 将生成的 Formula 提交到独立的 Homebrew tap 仓库
- 常见命名是 `<owner>/homebrew-tap`
- 用户安装方式通常是 `brew install <owner>/tap/cliai`

## 13. 生成 Debian 包

```bash
./scripts/new-deb-package.sh --version 0.2.3 --arch amd64
./scripts/new-deb-package.sh --version 0.2.3 --arch arm64
```

如果当前环境没有 `dpkg-deb`，可以先生成 staging 目录：

```bash
./scripts/new-deb-package.sh --version 0.2.3 --arch amd64 --stage-only
```

默认输出：

- `packaging/deb/0.2.3/amd64/stage/cliai_0.2.3_amd64/`
- `packaging/deb/0.2.3/amd64/cliai_0.2.3_amd64.deb`

说明：

- 这些脚本主要用于本地预演、手工发布或自定义托管
- 正式 `v*` tag 的 GitHub Release workflow 已自动构建 `.deb`、apt repo 元数据、可选签名、公钥和校验结果
- 若要让终端用户真正执行 `apt install cliai`，仍需要公开托管 apt 仓库地址

## 14. 生成 apt 仓库元数据

```bash
./scripts/new-apt-repo.sh \
  --repo-root ./packaging/apt/0.2.3 \
  --deb ./packaging/deb/0.2.3/amd64/cliai_0.2.3_amd64.deb \
  --deb ./packaging/deb/0.2.3/arm64/cliai_0.2.3_arm64.deb
```

脚本会生成：

- `pool/main/c/cliai/*.deb`
- `dists/stable/main/binary-amd64/Packages`
- `dists/stable/main/binary-amd64/Packages.gz`
- `dists/stable/Release`

说明：

- 本脚本只负责生成 apt repo 元数据，不负责签名
- 正式 Release workflow 会在提供 GPG key 时继续执行签名、公钥导出和结构校验

## 15. 为 apt 仓库生成签名

```bash
./scripts/sign-apt-repo.sh \
  --repo-root ./packaging/apt/0.2.3 \
  --require-signature
```

支持的环境变量：

- `CLIAI_APT_GPG_KEY_FILE`
- `CLIAI_APT_GPG_KEY_ARMORED`
- `CLIAI_APT_GPG_KEY_ID`
- `CLIAI_APT_GPG_PASSPHRASE`

脚本会生成：

- `dists/stable/Release.gpg`
- `dists/stable/InRelease`

如果发布流程里配置了 apt GPG key，workflow 还会额外导出并上传：

- `cliai-archive-keyring.asc`

完成后仍需：

- 将 apt 仓库上传到可公开访问的 Debian/Ubuntu 软件源地址
- 给用户提供软件源添加方式和安装命令
- 用 [RELEASE_CHECKLIST.md](file:///d:/sanqiu/cli%20ai/docs/RELEASE_CHECKLIST.md) 核对本地预检查、Release 产物和用户侧安装步骤

发布前后建议再对照 [RELEASE_CHECKLIST.md](file:///d:/sanqiu/cli%20ai/docs/RELEASE_CHECKLIST.md) 做一次人工核对。
