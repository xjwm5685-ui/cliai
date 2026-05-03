# Release Guide

本文件用于真实发布 `cliai` 到 GitHub Release 与 winget。

## 1. 需要的 GitHub Secrets

如果你想做已签名发布，可以提供以下 Secrets：

- `CLIAI_SIGN_PFX_BASE64`
- `CLIAI_SIGN_PFX_PASSWORD`
- `CLIAI_SIGN_TIMESTAMP_URL`

说明：

- `CLIAI_SIGN_PFX_BASE64`：将 `.pfx` 证书文件转为 Base64 后填入
- `CLIAI_SIGN_PFX_PASSWORD`：PFX 证书密码
- `CLIAI_SIGN_TIMESTAMP_URL`：时间戳服务地址，不提供时本地脚本默认使用 `http://timestamp.digicert.com`

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

## 4. 本地构建签名版本

无签名本地发布：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\release-local.ps1 -Version 0.2.0
```

需要签名时：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\release-local.ps1 -Version 0.2.0 -RequireSignature
```

## 5. 推送正式 tag

```powershell
git tag v0.2.0
git push origin v0.2.0
```

Release workflow 会：

1. 运行测试
2. 构建 `amd64` 和 `arm64`
3. 注入版本信息
4. 如提供证书则执行代码签名
5. 打包 zip
6. 生成 sha256
7. 运行 release smoke test
8. 上传 Release 资产

## 6. 生成 winget manifest

```powershell
.\scripts\new-winget-manifest.ps1 `
  -Version 0.2.0 `
  -X64Url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.0/cliai_Windows_x86_64.zip `
  -X64Sha256 YOUR_X64_SHA256 `
  -Arm64Url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.0/cliai_Windows_ARM64.zip `
  -Arm64Sha256 YOUR_ARM64_SHA256
```

## 7. 校验 winget manifest

```powershell
.\scripts\check-winget-manifest.ps1 -Directory .\packaging\winget\0.2.0
```

## 8. 提交到 winget-pkgs

将生成的以下文件提交到 `microsoft/winget-pkgs`：

- `Sanqiu.Cliai.yaml`
- `Sanqiu.Cliai.installer.yaml`
- `Sanqiu.Cliai.locale.zh-CN.yaml`
