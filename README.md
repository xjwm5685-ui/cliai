# cliai

`cliai` 是一个面向 Windows PowerShell 的高准确度命令预测 CLI。

它的目标不是只做传统的前缀补全，而是尽量结合自然语言、项目上下文、历史命令、个性化反馈和可选云端重排，给出更接近真实工作流的命令建议。

当前 GitHub 仓库地址：

- 项目主页：[xjwm5685-ui/cliai](https://github.com/xjwm5685-ui/cliai)
- 预期 winget 包名：`Sanqiu.Cliai`
- 正式发布说明：[RELEASE.md](file:///d:/sanqiu/cli%20ai/docs/RELEASE.md)
- 英文简介：[README_EN.md](file:///d:/sanqiu/cli%20ai/README_EN.md)

## 你可以把它理解成什么

`cliai` 当前更接近下面这个定位：

- 它是“命令建议器”，不是命令执行器
- 它会先在本地召回候选，再决定是否交给云端模型重排
- 它会利用当前目录里的项目类型来提升预测准确率
- 它会记录你最终接受了哪个候选，让后续排序逐渐更懂你
- 它会标记候选风险级别，帮助你在执行前做判断

## 典型效果

```powershell
cliai predict "git st"
cliai predict "安装 vscode"
cliai predict "启动"
cliai predict --cwd D:\code\my-go-app "run tests"
cliai predict --interactive "进入 src"
```

你可以期待的输出包括：

- `git st` -> `git status`
- `安装 vscode` -> `winget install vscode`
- 在 Node 项目中输入 `启动` -> `pnpm dev` 或 `npm run dev`
- 在 Go 项目中输入 `run tests` -> `go test ./...`
- 在当前目录有 `src` 文件夹时输入 `进入 src` -> `Set-Location .\src`

## 主要特性

- 混合预测：结合历史、内置命令知识库、意图模板和项目上下文
- 自然语言支持：支持中英文常见意图表达
- 项目上下文感知：识别 `Go`、`Node`、`Python`、`Docker`、`Git`
- 历史增强：从 PowerShell 历史和本地缓存中提取高频、近期命令
- 个性化学习：记录用户接受的候选，对后续相似查询加权
- 风险分级：对候选命令标记 `safe`、`caution`、`danger`
- 交互模式：支持交互式选择、复制到剪贴板、只输出最佳命令
- 云端重排：支持 OpenAI 兼容接口做候选排序增强
- 安全收敛：云端只能重排现有本地候选，不能发明新命令
- 发布就绪：内置 CI、Release、签名脚本、winget manifest 生成与校验脚本

## 当前边界

虽然这版已经比初版完整很多，但仍建议明确这些边界：

- 当前正式支持的 shell 仍然只有 `powershell`
- 程序不会直接执行候选命令，只负责建议、选择、复制和学习
- 项目上下文识别目前是轻量级静态检测，不是完整语义代理
- 历史学习主要来自“命令出现次数”和“用户明确接受的候选”
- 中文显示如果乱码，通常是终端编码问题，不是内部逻辑错误
- 代码签名是可选能力，不是默认发布前提

## 目录

- [快速开始](#快速开始)
- [安装](#安装)
- [命令总览](#命令总览)
- [predict 命令](#predict-命令)
- [feedback 命令](#feedback-命令)
- [history 命令](#history-命令)
- [config 命令](#config-命令)
- [shell 命令](#shell-命令)
- [selftest 命令](#selftest-命令)
- [输出格式](#输出格式)
- [工作原理](#工作原理)
- [项目上下文感知](#项目上下文感知)
- [反馈学习](#反馈学习)
- [云端重排与安全边界](#云端重排与安全边界)
- [配置说明](#配置说明)
- [环境变量](#环境变量)
- [PowerShell 集成](#powershell-集成)
- [项目结构](#项目结构)
- [开发](#开发)
- [测试与基准](#测试与基准)
- [CI 与发布](#ci-与发布)
- [代码签名](#代码签名)
- [winget 发布](#winget-发布)
- [常见问题](#常见问题)

## 快速开始

### 1. 构建

```powershell
git clone https://github.com/xjwm5685-ui/cliai.git
cd "cli ai"
go build -o cliai.exe .
```

### 2. 查看版本

```powershell
.\cliai.exe version
```

### 3. 导入 PowerShell 历史

```powershell
.\cliai.exe history import
```

### 4. 预测命令

```powershell
.\cliai.exe predict "安装 vscode"
.\cliai.exe predict "git st"
.\cliai.exe predict --json "run tests"
```

### 5. 打开交互模式

```powershell
.\cliai.exe predict --interactive "进入 src"
```

### 6. 初始化 PowerShell 助手

```powershell
.\cliai.exe shell init powershell
```

## 安装

### 本地源码安装

要求：

- Windows
- Go 1.25 或兼容版本

构建：

```powershell
go build -o cliai.exe .
```

如果你想全局使用，把生成的 `cliai.exe` 放到 `PATH` 中。

### 未来通过 winget 安装

预期安装命令：

```powershell
winget install Sanqiu.Cliai
```

说明：

- 当前仓库已经准备好 Release 与 winget manifest 生成链路
- 但是否能直接通过 `winget install` 使用，仍取决于 manifest 是否已提交并合并到 `microsoft/winget-pkgs`

## 命令总览

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

## predict 命令

### 用法

```powershell
cliai predict [flags] <query>
```

### 支持的参数

- `--limit <N>`：限制返回候选数，默认 `5`
- `--shell <name>`：指定 shell，默认读取配置中的 `shell`
- `--json`：输出 JSON
- `--no-cloud`：禁用云端重排
- `--cwd <path>`：指定工作目录，用于项目上下文检测
- `--debug`：输出调试信息到标准错误
- `--copy`：把最佳命令或交互选中的命令复制到剪贴板
- `--command-only`：只输出最优命令，不输出表格
- `--interactive`：进入交互式选择模式

### 示例

```powershell
cliai predict "git st"
cliai predict --limit 3 "安装 vscode"
cliai predict --json "搜索 docker"
cliai predict --no-cloud "run tests"
cliai predict --cwd D:\code\myapp "启动"
cliai predict --debug "进入 src"
cliai predict --command-only "run tests"
cliai predict --interactive --copy "进入 src"
```

### 参数解析特性

`predict` 做了一层额外处理：

- 支持前置 flag：`cliai predict --json "安装 vscode"`
- 支持尾置 flag：`cliai predict "安装 vscode" --json`
- 不会错误吞掉查询中间的字面量 `--json`、`--limit`

### 默认输出

默认输出表格包含 4 列：

- `COMMAND`
- `SOURCE`
- `RISK`
- `WHY`

示例：

```text
COMMAND                   SOURCE    RISK     WHY
winget install vscode     template  caution  install package from natural-language intent
winget search vscode      template  safe     search likely package first
```

### 风险级别

- `safe`：低风险建议
- `caution`：可能修改环境、安装、卸载、升级、切分分支或启动服务
- `danger`：删除、关机、格式化等高风险命令

### JSON 输出字段

```json
[
  {
    "command": "winget install vscode",
    "reason": "install package from natural-language intent",
    "source": "template",
    "score": 128,
    "risk": "caution"
  }
]
```

字段说明：

- `command`：候选命令
- `reason`：排序原因
- `source`：来源，可能是 `template`、`builtin`、`context`、`powershell-history`、`cloud`
- `score`：内部评分
- `risk`：风险级别

## feedback 命令

### 用法

```powershell
cliai feedback show
cliai feedback show --json
cliai feedback accept --query "安装 vscode" winget install vscode
```

### 作用

反馈模块用于记录“用户最终接受了哪个候选”，从而形成个性化加权。

### 目前支持

- 查看反馈记录
- 手动记录某次查询被接受的命令
- `predict --interactive` 选中候选后自动写入反馈

### 存储内容

反馈记录主要保存：

- 原始查询
- 归一化查询键
- 接受的命令
- 接受次数
- 最近一次接受时间

## history 命令

### 用法

```powershell
cliai history import [--file path]
```

### 作用

- 从 PowerShell 历史文件导入命令
- 去重并统计频率
- 清洗空行、超长内容和包含敏感字段的命令
- 写入本地缓存供后续排序使用

### 默认历史路径

```text
%USERPROFILE%\AppData\Roaming\Microsoft\Windows\PowerShell\PSReadLine\ConsoleHost_history.txt
```

### 示例

```powershell
cliai history import
cliai history import --file D:\temp\ConsoleHost_history.txt
```

## config 命令

### 查看配置

```powershell
cliai config show
```

### 设置配置

```powershell
cliai config set shell powershell
cliai config set history_path D:\custom\ConsoleHost_history.txt
cliai config set local.max_history 5000
cliai config set openai.enabled true
cliai config set openai.base_url https://api.openai.com/v1
cliai config set openai.model gpt-4.1-mini
cliai config set openai.api_key YOUR_KEY
cliai config set openai.timeout_seconds 20
```

### 配置项说明

- `shell`
  - 当前只支持 `powershell` 或 `pwsh`
  - 内部统一归一化为 `powershell`
- `history_path`
  - PowerShell 历史文件路径
- `local.max_history`
  - 导入历史时保留的最大命令数
- `openai.enabled`
  - 是否启用云端重排
- `openai.base_url`
  - OpenAI 兼容接口地址
- `openai.api_key`
  - API Key
- `openai.model`
  - 模型名
- `openai.timeout_seconds`
  - 云端请求超时秒数

## shell 命令

### 用法

```powershell
cliai shell init powershell
```

### 输出内容

当前会输出 PowerShell 助手函数和别名：

- `Invoke-CliaiSuggestion`
- `Invoke-CliaiInteractiveSuggestion`
- `Get-CliaiTopCommand`
- `csg`
- `csi`
- `csc`

### 助手别名含义

- `csg`：普通建议模式
- `csi`：交互式选择并复制
- `csc`：只输出最佳命令

### 用法示例

```powershell
csg "安装 vscode"
csi "git st"
csc "run tests"
```

### 自动写入 Profile

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-powershell.ps1
```

## selftest 命令

### 用法

```powershell
cliai selftest
cliai selftest --json
```

### 检查内容

- 版本元信息是否可读
- 配置是否可加载
- 缓存路径是否可解析
- 反馈路径是否可解析
- 当前目录项目上下文是否可识别

这个命令适合：

- 本地安装后做 smoke test
- Release 流程中做产物验证
- CI 中确认二进制可运行

## 输出格式

### 表格模式

适合人直接阅读。

### JSON 模式

适合脚本或其他工具消费。

### 只输出命令模式

```powershell
cliai predict --command-only "git st"
```

适合：

- 粘贴到其他脚本
- PowerShell 补全
- 快捷键绑定

## 工作原理

`cliai` 的预测流程现在大致分为 6 层。

### 1. 读取本地上下文

程序会读取：

- 当前工作目录
- shell 类型
- PowerShell 历史
- 本地缓存
- 用户反馈记录

### 2. 检测项目类型

当前会检测：

- `go.mod` -> `go`
- `package.json` / `pnpm-lock.yaml` / `package-lock.json` / `yarn.lock` -> `node`
- `requirements.txt` / `pyproject.toml` -> `python`
- `Dockerfile` / `docker-compose.yml` / `compose.yaml` -> `docker`
- `.git` -> `git`

### 3. 本地召回候选

候选来源包括：

- 内置命令知识库
- 自然语言模板
- 项目上下文模板
- 历史命令

### 4. 本地打分

本地排序综合考虑：

- 前缀命中
- 子串命中
- token 匹配
- 关键词匹配
- 历史频率
- 近期性
- 项目上下文加权
- 用户反馈加权

### 5. 可选云端重排

如果启用 OpenAI 兼容接口：

- 本地先生成候选
- 候选列表和查询会发给云端
- 云端只能返回“选择第几个候选”
- 本地按照索引重排，并保留未选中的本地候选

### 6. 交互与学习

如果你使用 `--interactive`：

- 程序展示候选编号
- 你选择其中一个
- 该选择会自动记录到反馈存储
- 后续相似查询会为该命令增加额外分数

## 项目上下文感知

这是当前版本提升准确率最重要的增强点之一。

### 例子

在 Go 项目目录：

```powershell
cliai predict "run tests"
```

更容易得到：

```text
go test ./...
```

在 Node + pnpm 项目目录：

```powershell
cliai predict "启动"
```

更容易得到：

```text
pnpm dev
```

在当前目录有 `src` 文件夹时：

```powershell
cliai predict "进入 src"
```

更容易得到：

```text
Set-Location .\src
```

## 反馈学习

当前反馈学习是轻量级但实用的。

### 学习来源

- 手动执行 `feedback accept`
- 交互式选择自动记录

### 学习方式

如果某条查询经常对应某条命令：

- 完全相同查询会获得更高加权
- 相近查询也会有一定加权
- 最近接受的命令会获得额外奖励

### 适合的用法

如果你经常对同样的自然语言使用同一条命令，这个功能会越来越有效。

## 云端重排与安全边界

### 什么时候值得开

适合这些情况：

- 自然语言表达比较模糊
- 候选很多且相似
- 你希望排序更贴近语义理解

### 安全边界

当前实现的关键约束：

- 云端不能返回新命令
- 云端只能从本地候选列表中选择索引
- 云端解析失败时自动退回本地结果
- 本地仍然保留未被云端选中的候选作为兜底

这比“完全让 LLM 生成要执行的命令”更安全。

### 启用方式

```powershell
cliai config set openai.enabled true
cliai config set openai.base_url https://api.openai.com/v1
cliai config set openai.model gpt-4.1-mini
cliai config set openai.api_key YOUR_API_KEY
```

### 临时禁用

```powershell
cliai predict --no-cloud "安装 vscode"
```

## 配置说明

### 配置文件位置

通常位于：

```text
%AppData%\cliai\config.json
```

历史缓存通常位于：

```text
%AppData%\cliai\history_cache.json
```

反馈记录通常位于：

```text
%AppData%\cliai\feedback.json
```

### 配置示例

```json
{
  "shell": "powershell",
  "history_path": "C:\\Users\\YOUR_NAME\\AppData\\Roaming\\Microsoft\\Windows\\PowerShell\\PSReadLine\\ConsoleHost_history.txt",
  "local": {
    "max_history": 4000
  },
  "openai": {
    "enabled": false,
    "base_url": "https://api.openai.com/v1",
    "api_key": "",
    "model": "gpt-4.1-mini",
    "timeout_seconds": 20
  }
}
```

## 环境变量

环境变量会覆盖配置文件值。

支持：

- `CLIAI_SHELL`
- `CLIAI_HISTORY_PATH`
- `CLIAI_OPENAI_API_KEY`
- `CLIAI_OPENAI_BASE_URL`
- `CLIAI_OPENAI_MODEL`

示例：

```powershell
$env:CLIAI_SHELL="powershell"
$env:CLIAI_HISTORY_PATH="D:\history\ConsoleHost_history.txt"
$env:CLIAI_OPENAI_API_KEY="YOUR_API_KEY"
$env:CLIAI_OPENAI_BASE_URL="https://api.openai.com/v1"
$env:CLIAI_OPENAI_MODEL="gpt-4.1-mini"
```

## PowerShell 集成

### 方式一：手动输出

```powershell
cliai shell init powershell
```

把输出复制到 PowerShell Profile 中。

### 方式二：自动安装

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-powershell.ps1
```

### 安装后可直接使用

```powershell
csg "安装 vscode"
csi "git st"
csc "run tests"
```

## 项目结构

```text
cli ai/
├─ .github/
│  └─ workflows/
│     ├─ ci.yml
│     └─ release.yml
├─ internal/
│  ├─ app/
│  ├─ cloud/
│  ├─ config/
│  ├─ feedback/
│  ├─ history/
│  ├─ predict/
│  └─ project/
├─ scripts/
│  ├─ install-powershell.ps1
│  ├─ sign-windows.ps1
│  ├─ validate-release.ps1
│  ├─ new-winget-manifest.ps1
│  ├─ check-winget-manifest.ps1
│  └─ release-local.ps1
├─ CHANGELOG.md
├─ LICENSE
├─ README.md
├─ go.mod
└─ main.go
```

## 开发

### 本地运行

```powershell
go run . version
go run . selftest --json
go run . predict "安装 vscode"
go run . predict --debug "run tests"
```

### 构建

```powershell
go build -o cliai.exe .
```

### 带版本元信息构建

```powershell
go build -ldflags "-X github.com/sanqiu/cliai/internal/app.Version=0.2.0 -X github.com/sanqiu/cliai/internal/app.Commit=local -X github.com/sanqiu/cliai/internal/app.BuildDate=2026-05-03T12:00:00" -o cliai.exe .
```

### 格式化

```powershell
gofmt -w main.go .\internal\...\*.go
```

## 测试与基准

运行全部测试：

```powershell
go test ./...
```

运行预测器基准：

```powershell
go test ./internal/predict -run ^$ -bench BenchmarkPredict
```

当前测试覆盖包括：

- 参数解析
- 云端重排安全边界
- shell 配置校验
- PowerShell 专用命令过滤
- 项目上下文识别
- 反馈学习
- 历史敏感命令清洗
- 预测器基准性能

## CI 与发布

### CI

文件：

- `.github/workflows/ci.yml`

CI 会执行：

- `gofmt` 检查
- `go test ./...`
- 基准冒烟
- 构建带版本元信息的二进制
- `selftest --json`

### Release

文件：

- `.github/workflows/release.yml`

触发条件：

- 推送 `v*` tag

Release 会：

1. 运行测试
2. 为 `windows/amd64` 和 `windows/arm64` 构建二进制
3. 注入 `Version`、`Commit`、`BuildDate`
4. 如果提供证书则执行代码签名
5. 打包为 zip
6. 生成 SHA256
7. 执行本地 release smoke test
8. 上传 Release 资产

正式发布前的 Secrets、证书和实际提交流程见 [RELEASE.md](file:///d:/sanqiu/cli%20ai/docs/RELEASE.md)。

## 代码签名

仓库内已补齐签名流程支持，但默认发布并不强制要求签名。

### 脚本

- `scripts/sign-windows.ps1`

### 支持的环境变量

- `CLIAI_SIGN_PFX_BASE64`
- `CLIAI_SIGN_PFX_PASSWORD`
- `CLIAI_SIGN_TIMESTAMP_URL`

### 工作方式

- 如果没有提供签名密钥，脚本会跳过签名
- 如果提供了 PFX 和密码，脚本会导入证书并对 `cliai.exe` 执行 `Set-AuthenticodeSignature`

### 本地签名发布

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\release-local.ps1 -Version 0.2.0 -RequireSignature
```

### 本地无签名发布

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\release-local.ps1 -Version 0.2.0
```

## winget 发布

### 1. 先准备 Release 产物

预期资产链接：

- `https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.0/cliai_Windows_x86_64.zip`
- `https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.0/cliai_Windows_ARM64.zip`

### 2. 生成 manifest

```powershell
.\scripts\new-winget-manifest.ps1 `
  -Version 0.2.0 `
  -X64Url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.0/cliai_Windows_x86_64.zip `
  -X64Sha256 YOUR_X64_SHA256 `
  -Arm64Url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.0/cliai_Windows_ARM64.zip `
  -Arm64Sha256 YOUR_ARM64_SHA256
```

### 3. 校验 manifest

```powershell
.\scripts\check-winget-manifest.ps1 -Directory .\packaging\winget\0.2.0
```

### 4. 提交到 winget-pkgs

把生成的这些文件提交到 [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs)：

- `Sanqiu.Cliai.yaml`
- `Sanqiu.Cliai.installer.yaml`
- `Sanqiu.Cliai.locale.zh-CN.yaml`

最终用户安装方式：

```powershell
winget install Sanqiu.Cliai
```

## 常见问题

### 为什么中文有时会乱码

通常是终端编码问题。可尝试：

```powershell
chcp 65001
```

或者使用较新的 Windows Terminal / PowerShell 7。

### 为什么我开了云端却感觉没效果

检查：

- `openai.enabled`
- `openai.api_key`
- `openai.base_url`
- `openai.model`

如果云端返回空数据、非法索引或解析失败，程序会自动退回本地结果。

### 为什么不会直接执行命令

这是刻意设计：

- 程序负责召回、排序、解释、交互选择
- 最终执行权仍然在用户手里

这样更安全，也更适合逐步提高准确率。

### 为什么历史里有些命令没被学进去

历史导入会主动过滤：

- 空命令
- 超长命令
- 疑似包含敏感信息的命令

例如带 `Authorization`、`Bearer`、`password`、`api_key`、`token` 的内容会被跳过。

## 许可证

本项目采用 [MIT License](./LICENSE)。

## 使用建议

- 把它当作“高质量命令建议器”，不是“无脑执行器”
- 对 `caution` 和 `danger` 级别命令保持人工确认
- 在启用云端模型时，优先使用你信任的 OpenAI 兼容服务
- 想要长期提升准确率时，建议同时使用 `history import` 和交互式反馈学习
