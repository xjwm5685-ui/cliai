# cliai

`cliai` 是一个跨平台的高准确度命令预测 CLI，在 Windows、Linux、macOS 上都能运行，并对 PowerShell 实时灰字预测做了优先优化。

它的目标不是只做传统的前缀补全，而是尽量结合自然语言、项目上下文、历史命令、个性化反馈和可选云端重排，给出更接近真实工作流的命令建议。

当前 GitHub 仓库地址：

- 项目主页：[xjwm5685-ui/cliai](https://github.com/xjwm5685-ui/cliai)
- 预期 winget 包名：`Sanqiu.Cliai`
- 预期 Chocolatey 包名：`sanqiu-cliai`
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
- 历史增强：从 PowerShell、bash、zsh、fish 历史和本地缓存中提取高频、近期命令
- 个性化学习：记录用户接受的候选，对后续相似查询加权
- 风险分级：对候选命令标记 `safe`、`caution`、`danger`
- 交互模式：支持交互式选择、复制到剪贴板、只输出最佳命令
- 云端重排：支持 OpenAI 兼容接口做候选排序增强
- 安全收敛：云端只能重排现有本地候选，不能发明新命令
- 发布就绪：内置 CI、Release、签名脚本、winget/Chocolatey 包生成与校验辅助脚本

## 支持矩阵

| 能力 | Windows | Linux | macOS |
| --- | --- | --- | --- |
| 核心 CLI 构建与运行 | 已支持 | 已支持 | 已支持 |
| `history import` 默认路径 | PowerShell | bash/zsh/fish/pwsh | zsh/bash/pwsh |
| `predict` / `selftest` / `config` | 已支持 | 已支持 | 已支持 |
| GitHub Release 产物 | zip | tar.gz | tar.gz |
| 一键系统安装脚本 | `install-powershell.ps1` | `install-unix.sh` | `install-unix.sh` |
| PowerShell 实时灰字预测 | 已支持 | 已支持，需本机安装 `pwsh` | 已支持，需本机安装 `pwsh` |
| 原生包管理分发 | winget / Chocolatey | 已补 `.deb` 与 apt repo 脚本 | 已补 Homebrew Formula 脚本 |

## 当前边界

虽然这版已经比初版完整很多，但仍建议明确这些边界：

- CLI 核心已支持 `powershell`、`bash`、`zsh`、`fish`
- 实时灰字预测当前依赖 PowerShell `7.2+` 与 `PSReadLine`
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
- [支持矩阵](#支持矩阵)
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
- [Chocolatey 发布](#chocolatey-发布)
- [常见问题](#常见问题)

## 快速开始

### 1. 构建

```powershell
git clone https://github.com/xjwm5685-ui/cliai.git
cd "cli ai"
go build -o .\bin\cliai.exe .
```

### 2. 查看版本

```powershell
.\bin\cliai.exe version
```

### 3. 导入 PowerShell 历史

```powershell
.\bin\cliai.exe history import
```

### 4. 预测命令

```powershell
.\bin\cliai.exe predict "安装 vscode"
.\bin\cliai.exe predict "git st"
.\bin\cliai.exe predict --json "run tests"
```

### 5. 打开交互模式

```powershell
.\bin\cliai.exe predict --interactive "进入 src"
```

### 6. 启用 PowerShell 实时预测

```powershell
.\bin\cliai.exe shell install powershell
```

### 7. 初始化 PowerShell 助手

```powershell
.\bin\cliai.exe shell init powershell
```

## 安装

### 本地源码安装

要求：

- Windows、Linux 或 macOS
- Go `1.25` 或兼容版本

Windows PowerShell：

```powershell
go build -o .\bin\cliai.exe .
```

Linux / macOS：

```bash
go build -o ./bin/cliai .
```

如果你想全局使用，把生成的二进制放到 `PATH` 中。

### GitHub Release 安装

Windows：

推荐一键安装：

```powershell
iwr -useb https://raw.githubusercontent.com/xjwm5685-ui/cliai/main/install.ps1 | iex
```

说明：

- 该脚本会自动识别 `amd64` / `arm64`
- 自动下载最新 Windows Release zip 与 `.sha256`
- 自动校验校验和、解压到用户目录并写入用户 `PATH`
- 自动接入 PowerShell Profile 与实时灰字预测

也可以手动下载安装包：

1. 下载 `cliai_Windows_x86_64.zip` 或 `cliai_Windows_ARM64.zip`
2. 解压后运行：

```powershell
.\cliai.exe shell install powershell
```

Linux / macOS：

1. 下载对应的 `cliai_Linux_*.tar.gz` 或 `cliai_macOS_*.tar.gz`
2. 解压后运行：

```bash
tar -xzf cliai_Linux_x86_64.tar.gz
cd <extract-dir>
./scripts/install-unix.sh
```

如果系统里已安装 `pwsh`，安装脚本会提示你继续执行：

```bash
cliai shell install powershell
```

来启用 PowerShell 实时灰字预测。

### 未来通过 winget 安装

预期安装命令：

```powershell
winget install Sanqiu.Cliai
```

说明：

- 当前仓库已经准备好 Release 与 winget manifest 生成链路
- 但是否能直接通过 `winget install` 使用，仍取决于 manifest 是否已提交并合并到 `microsoft/winget-pkgs`

### 未来通过 Chocolatey 安装

预期安装命令：

```powershell
choco install sanqiu-cliai
```

说明：

- 当前仓库已经补齐 Chocolatey 包生成脚本与包目录结构
- 正式可安装仍取决于包是否已推送并通过 Chocolatey 社区仓库审核

### 未来通过 Homebrew 安装

预期安装命令：

```bash
brew install <your-tap>/cliai
```

说明：

- 当前仓库已补 `scripts/new-homebrew-formula.sh`
- 它会根据 GitHub Release 的 macOS/Linux tar.gz 资产生成 Homebrew Formula
- 仍需要单独维护并发布一个 Homebrew tap 仓库

### 未来通过 apt 安装

预期安装命令：

```bash
sudo apt install cliai
```

说明：

- 当前仓库已补 `scripts/new-deb-package.sh`
- 它可以生成 Debian 包 staging 目录，并在 Linux 上调用 `dpkg-deb` 构建 `.deb`
- 当前仓库也已补 `scripts/new-apt-repo.sh`，可生成 `pool/`、`dists/`、`Packages.gz`、`Release`
- 当前仓库已补 `scripts/sign-apt-repo.sh`，可生成 `Release.gpg` 和 `InRelease`
- 当前 Release workflow 已自动构建 `.deb`、apt repo 元数据、可选签名、公钥和 apt 校验
- 还缺真实公开的 apt 源托管地址，以及最终面向用户的软件源配置入口

## 命令总览

```text
cliai predict <query>
cliai predictor serve
cliai history import
cliai config show
cliai config set <key> <value>
cliai feedback show
cliai feedback accept --query <query> <command>
cliai shell init powershell
cliai shell install powershell
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

- Windows PowerShell：`%USERPROFILE%\AppData\Roaming\Microsoft\Windows\PowerShell\PSReadLine\ConsoleHost_history.txt`
- Linux/macOS PowerShell：`~/.local/share/powershell/PSReadLine/ConsoleHost_history.txt`
- bash：`~/.bash_history`
- zsh：`~/.zsh_history`
- fish：`~/.local/share/fish/fish_history`

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
  - 支持 `powershell`、`pwsh`、`bash`、`zsh`、`fish`
  - `pwsh` 内部会统一归一化为 `powershell`
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
cliai shell install powershell
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
cliai shell install powershell
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

### 实时灰字预测

从这一版开始，`cliai` 不再只提供手动 `cliai predict ...`。

现在可以通过 PowerShell Predictive IntelliSense 插件提供真正的实时灰字预测：

- 你输入 `git st` 时，后面会直接出现灰色 `git status`
- 你输入自然语言片段时，会由 `CliaiPredictor` 调用本地 `cliai predictor serve`
- 预测结果来自本地历史、项目上下文、模板和反馈学习
- 为了保证实时体验，插件默认关闭云端重排，只使用本地召回

### 前置要求

- PowerShell `7.2+`
- PSReadLine `2.2.2+`
- `.NET SDK 8`，仅在没有 bundled predictor 模块、需要本地编译 `CliaiPredictor` 时必需

当前开发环境已验证：

- PowerShell `7.6.1`
- PSReadLine `2.4.5`
- `.NET SDK 8.0.420`

### 方式一：只加载 helper

```powershell
cliai shell init powershell
```

把输出复制到 PowerShell Profile 中。

这会提供：

- `csg`
- `csi`
- `csc`

但这不是实时灰字预测。

### 方式二：一键启用 helper + 实时灰字预测

推荐直接执行：

```powershell
.\bin\cliai.exe shell install powershell
```

如果你使用的是 Linux/macOS 且系统里安装了 `pwsh`，也可以在完成 `install-unix.sh` 后执行同一条命令。

这个命令会自动寻找：

- 安装目录里的 `scripts\install-powershell.ps1`
- 安装目录里的 `modules\CliaiPredictor\<version>`
- 当前 `cliai.exe`

然后自动完成 profile 写入和 predictor 安装。

### 方式三：手动执行安装脚本

```powershell
.\bin\cliai.exe version
powershell -ExecutionPolicy Bypass -File .\scripts\install-powershell.ps1 -ExeName .\bin\cliai.exe
```

安装脚本会做这些事情：

- 如有需要则编译 `predictor\CliaiPredictor\CliaiPredictor.csproj`
- 把二进制模块复制到 `Documents\PowerShell\Modules\CliaiPredictor\0.2.1`
- 往 PowerShell 7 的 `Documents\PowerShell\Profile.ps1` 写入 helper 片段
- 设置 `$env:CLIAI_EXE`
- 自动执行 `Import-Module CliaiPredictor`
- 默认设置 `Set-PSReadLineOption -PredictionSource Plugin`
- 自动绑定 `Alt+RightArrow` 和 `Alt+Shift+RightArrow`

### 手动验证预测器是否注册

```powershell
Import-Module CliaiPredictor
(Get-PSSubsystem -Kind CommandPredictor).Implementations |
  Select-Object Id, Name, Description
```

如果安装成功，你应该能看到：

- `Name = CliaiPredictor`

### 安装后可直接使用

```powershell
csg "安装 vscode"
csi "git st"
csc "run tests"
```

同时也可以直接在命令行输入时看到灰字预测。

默认推荐按键：

- `RightArrow`：接受当前 inline 预测
- `Alt+RightArrow`：接受整条预测
- `Alt+Shift+RightArrow`：接受下一个预测词

### 关于 Shift 补全

`PSReadLine` 不能绑定“单独按 Shift”。

因此当前默认改用上面这组 `Alt` 组合键，而不是继续模拟“单独按 Shift 补全”。

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
├─ packaging/
│  ├─ chocolatey/
│  ├─ deb/
│  ├─ apt/
│  ├─ homebrew/
│  └─ winget/
├─ scripts/
│  ├─ install-powershell.ps1
│  ├─ install-unix.sh
│  ├─ check-release-env.sh
│  ├─ new-apt-repo.sh
│  ├─ new-deb-package.sh
│  ├─ new-homebrew-formula.sh
│  ├─ sign-apt-repo.sh
│  ├─ release-local.sh
│  ├─ sign-windows.ps1
│  ├─ validate-release.ps1
│  ├─ new-chocolatey-package.ps1
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

- 在 `Windows`、`Linux`、`macOS` 上执行 `gofmt` 检查
- 在 `Windows`、`Linux`、`macOS` 上执行 `go test ./...`
- 运行预测器基准冒烟
- 构建带版本元信息的本机二进制
- 运行 `selftest --json`

### Release

文件：

- `.github/workflows/release.yml`

触发条件：

- 推送 `v*` tag

Release 会：

1. 运行测试
2. 为 `windows/amd64`、`windows/arm64` 构建 zip 发布包
3. 为 `linux/amd64`、`linux/arm64`、`darwin/amd64`、`darwin/arm64` 构建 tar.gz 发布包
4. 为 Linux 额外构建 `amd64` / `arm64` 的 `.deb`
5. 自动生成 apt 仓库元数据归档
6. 注入 `Version`、`Commit`、`BuildDate`
7. Windows 包额外暂存 `CliaiPredictor` 和 `install-powershell.ps1`
8. Unix 包额外暂存 `scripts/install-unix.sh`
9. 如果提供证书则对 Windows 可执行文件执行代码签名
10. 如果提供 GPG 私钥则为 apt 仓库生成 `Release.gpg` 和 `InRelease`
11. 生成 SHA256
12. 自动校验 `.deb`、apt repo metadata、签名文件和公钥文件结构
13. 对可在当前 runner 上执行的发布包运行 smoke test
14. 上传 Release 资产

正式发布前的 Secrets、证书和实际提交流程见 [RELEASE.md](file:///d:/sanqiu/cli%20ai/docs/RELEASE.md)，发布核对项见 [RELEASE_CHECKLIST.md](file:///d:/sanqiu/cli%20ai/docs/RELEASE_CHECKLIST.md)。

## 代码签名

仓库内已补齐签名流程支持，但默认发布并不强制要求签名。

### 脚本

- `scripts/sign-windows.ps1`

### 支持的环境变量

- `CLIAI_SIGN_PFX_BASE64`
- `CLIAI_SIGN_PFX_PASSWORD`
- `CLIAI_SIGN_TIMESTAMP_URL`
- `CLIAI_APT_GPG_KEY_FILE`
- `CLIAI_APT_GPG_KEY_ARMORED`
- `CLIAI_APT_GPG_KEY_ID`
- `CLIAI_APT_GPG_PASSPHRASE`

### 工作方式

- 如果没有提供签名密钥，脚本会跳过签名
- 如果提供了 PFX 和密码，脚本会导入证书并对 `cliai.exe` 执行 `Set-AuthenticodeSignature`

### 本地 Windows 签名发布

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\release-local.ps1 -Version 0.2.0 -RequireSignature
```

### 本地 Windows 无签名发布

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\release-local.ps1 -Version 0.2.0
```

### 本地 Linux/macOS 发布

```bash
./scripts/check-release-env.sh
./scripts/release-local.sh 0.2.0
```

说明：

- `release-local.ps1` 会生成 Windows zip、SHA256、predictor 模块和安装脚本
- `release-local.sh` 会生成 Linux/macOS tar.gz、SHA256 和 `install-unix.sh`
- 如果本机有 `dpkg-deb`，`release-local.sh` 还会额外生成 `.deb`、apt repo、签名文件、公钥和 apt 校验结果

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

## Chocolatey 发布

### 1. 先准备 GitHub Release 产物

当前 Chocolatey 包默认使用 x64 便携 zip 资产：

- `https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.1/cliai_Windows_x86_64.zip`

### 2. 生成 Chocolatey 包目录

```powershell
.\scripts\new-chocolatey-package.ps1 `
  -Version 0.2.1 `
  -X64Url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.1/cliai_Windows_x86_64.zip `
  -X64Sha256 YOUR_X64_SHA256
```

生成后目录类似：

- `packaging/chocolatey/0.2.1/sanqiu-cliai.nuspec`
- `packaging/chocolatey/0.2.1/tools/chocolateyinstall.ps1`
- `packaging/chocolatey/0.2.1/tools/chocolateyuninstall.ps1`
- `packaging/chocolatey/0.2.1/tools/VERIFICATION.txt`

### 3. 本地打包

```powershell
cd .\packaging\chocolatey\0.2.1
choco pack
```

### 4. 推送到 Chocolatey 社区源

```powershell
choco push .\sanqiu-cliai.0.2.1.nupkg --source https://push.chocolatey.org/ --api-key YOUR_API_KEY
```

最终用户安装方式：

```powershell
choco install sanqiu-cliai
```

## Homebrew 发布

### 1. 先准备 GitHub Release 产物

预期资产链接：

- `https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.1/cliai_macOS_x86_64.tar.gz`
- `https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.1/cliai_macOS_ARM64.tar.gz`
- 可选：Linux 对应的 `cliai_Linux_x86_64.tar.gz` 和 `cliai_Linux_ARM64.tar.gz`

### 2. 生成 Formula

```bash
./scripts/new-homebrew-formula.sh \
  --version 0.2.1 \
  --darwin-amd64-url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.1/cliai_macOS_x86_64.tar.gz \
  --darwin-amd64-sha256 YOUR_MACOS_X64_SHA256 \
  --darwin-arm64-url https://github.com/xjwm5685-ui/cliai/releases/download/v0.2.1/cliai_macOS_ARM64.tar.gz \
  --darwin-arm64-sha256 YOUR_MACOS_ARM64_SHA256
```

### 3. 提交到 Tap 仓库

- 将生成的 `packaging/homebrew/0.2.1/cliai.rb` 提交到你的 Homebrew tap
- 常见仓库名是 `homebrew-tap`
- 用户安装命令通常是 `brew install <owner>/tap/cliai`

## Debian 发布

### 1. 先准备 Linux Release 二进制

例如：

- `dist/linux-amd64/cliai`
- `dist/linux-arm64/cliai`

### 2. 生成 Debian 包目录或 `.deb`

```bash
./scripts/new-deb-package.sh --version 0.2.1 --arch amd64
./scripts/new-deb-package.sh --version 0.2.1 --arch arm64
```

如果当前机器没有 `dpkg-deb`，也可以先只生成 staging 目录：

```bash
./scripts/new-deb-package.sh --version 0.2.1 --arch amd64 --stage-only
```

### 3. 输出位置

- `packaging/deb/0.2.1/amd64/stage/cliai_0.2.1_amd64/`
- `packaging/deb/0.2.1/amd64/cliai_0.2.1_amd64.deb`

### 4. 生成 apt 仓库元数据

```bash
./scripts/new-apt-repo.sh \
  --repo-root ./packaging/apt/0.2.1 \
  --deb ./packaging/deb/0.2.1/amd64/cliai_0.2.1_amd64.deb \
  --deb ./packaging/deb/0.2.1/arm64/cliai_0.2.1_arm64.deb
```

默认会生成：

- `packaging/apt/0.2.1/pool/main/c/cliai/*.deb`
- `packaging/apt/0.2.1/dists/stable/main/binary-amd64/Packages`
- `packaging/apt/0.2.1/dists/stable/main/binary-amd64/Packages.gz`
- `packaging/apt/0.2.1/dists/stable/Release`

### 5. 后续 apt 源托管

先生成签名文件：

```bash
./scripts/sign-apt-repo.sh \
  --repo-root ./packaging/apt/0.2.1 \
  --require-signature
```

脚本支持以下环境变量：

- `CLIAI_APT_GPG_KEY_FILE`
- `CLIAI_APT_GPG_KEY_ARMORED`
- `CLIAI_APT_GPG_KEY_ID`
- `CLIAI_APT_GPG_PASSPHRASE`

签名后会得到：

- `packaging/apt/0.2.1/dists/stable/Release.gpg`
- `packaging/apt/0.2.1/dists/stable/InRelease`
- 发布流程配置了 apt GPG key 后，还会额外上传 `cliai-archive-keyring.asc`

如果你已经把 apt 仓库发布到某个静态地址，例如 `https://example.com/cliai/apt`，用户侧安装命令可以写成：

```bash
sudo install -d -m 0755 /etc/apt/keyrings
curl -fsSL https://example.com/cliai/apt/cliai-archive-keyring.asc | sudo tee /etc/apt/keyrings/cliai-archive-keyring.asc >/dev/null
echo "deb [signed-by=/etc/apt/keyrings/cliai-archive-keyring.asc] https://example.com/cliai/apt stable main" | sudo tee /etc/apt/sources.list.d/cliai.list >/dev/null
sudo apt update
sudo apt install cliai
```

若要支持真实的 `apt install cliai`，还需要继续补：

- 可公开访问的 Debian/Ubuntu 软件源托管地址
- 用户侧的软件源添加说明与安装命令

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
