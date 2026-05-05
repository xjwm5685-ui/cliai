# cliai

`cliai` 是一个跨平台、本地优先的命令预测与补全工具，运行在 Windows、Linux、macOS，并为 PowerShell、zsh、bash 提供不同程度的补全体验。

它不是命令执行器，而是“更懂当前终端上下文”的命令建议器：

- 结合本地历史、内置规则、自然语言模板和项目上下文生成候选
- 记录你实际接受过的候选，让后续排序逐渐更贴近个人习惯
- 标记候选风险级别，避免把高风险命令伪装成普通补全
- 保持本地工作流，不依赖云端模型或外部 AI 服务

当前仓库：

- 项目主页：[xjwm5685-ui/cliai](https://github.com/xjwm5685-ui/cliai)
- 社区友链：[Linux.do](https://linux.do)
- 英文说明：[README_EN.md](file:///d:/sanqiu/cliai/README_EN.md)
- 发布说明：[RELEASE.md](file:///d:/sanqiu/cliai/docs/RELEASE.md)
- 发布说明草稿：[RELEASE_NOTES_DRAFT.md](file:///d:/sanqiu/cliai/docs/RELEASE_NOTES_DRAFT.md)
- 发布清单：[RELEASE_CHECKLIST.md](file:///d:/sanqiu/cliai/docs/RELEASE_CHECKLIST.md)

## 典型效果

```powershell
cliai predict "git st"
cliai predict "安装 vscode"
cliai predict "启动"
cliai predict --cwd D:\code\my-go-app "run tests"
cliai predict --interactive "进入 src"
```

常见结果：

- `git st` -> `git status`
- `安装 vscode` -> `winget install vscode`
- Node 项目里的 `启动` -> `pnpm dev` 或 `npm run dev`
- Go 项目里的 `run tests` -> `go test ./...`
- 当前目录存在 `src` 时，`进入 src` -> `Set-Location .\src`

## 主要特性

- 本地预测：结合历史、内置知识库、意图模板和项目上下文
- 自然语言输入：支持常见中英文命令意图
- 项目感知：识别 `Go`、`Node`、`Python`、`Docker`、`Git`
- 历史增强：读取 PowerShell、bash、zsh、fish 历史和本地缓存
- 反馈学习：对被你接受过的命令持续加权
- 风险提示：标记 `safe`、`caution`、`danger`
- 多种输出：表格、JSON、仅输出命令、交互式选择、复制到剪贴板
- 多 shell 集成：PowerShell 实时灰字预测，zsh 原生灰字预测，bash 快捷接受预测

## 支持矩阵

| 能力 | Windows | Linux | macOS |
| --- | --- | --- | --- |
| 核心 CLI 构建与运行 | 已支持 | 已支持 | 已支持 |
| `history import` 默认路径 | PowerShell | bash/zsh/fish/pwsh | zsh/bash/pwsh |
| `predict` / `selftest` / `config` | 已支持 | 已支持 | 已支持 |
| GitHub Release 产物 | zip | tar.gz / `.deb` | tar.gz |
| 安装脚本 | `install.ps1` | `install.sh` / `install-unix.sh` | `install-unix.sh` |
| 实时预测 | PowerShell 原生支持 | `zsh` 原生灰字，`bash` 快捷接受，`pwsh` 可用 | `zsh` 原生灰字，`bash` 快捷接受，`pwsh` 可用 |

## 当前边界

- 工具只负责建议、展示、复制和学习，不会自动执行命令
- Linux/macOS/WSL2 推荐使用 `zsh` 获得最接近 PowerShell 的灰字体验
- `bash` 当前重点是“快速接受预测”，不是完整灰字渲染
- 项目上下文识别是轻量规则检测，不是完整语义分析器
- 反馈学习依赖本地历史和接受记录，第一次使用时效果会更基础

## 快速开始

### 源码构建

```powershell
git clone https://github.com/xjwm5685-ui/cliai.git
cd cliai
go build -o .\bin\cliai.exe .
```

Windows PowerShell：

```powershell
.\bin\cliai.exe history import
.\bin\cliai.exe predict "安装 vscode"
.\bin\cliai.exe shell install powershell
```

Linux / macOS：

```bash
go build -o ./bin/cliai .
./bin/cliai history import
./bin/cliai predict "run tests"
./bin/cliai shell install zsh
```

## 安装

### Windows 一键安装

```powershell
iwr -useb https://raw.githubusercontent.com/xjwm5685-ui/cliai/main/install.ps1 | iex
```

这个安装脚本会：

- 自动识别 `amd64` / `arm64`
- 下载最新 Windows Release 与 `.sha256`
- 校验校验和并写入用户 `PATH`
- 自动安装 PowerShell 预测集成

如果你只想安装 `csg` / `csi` / `csc` helper，而不安装完整预测器模块：

```powershell
$env:CLIAI_SHELL_INTEGRATION = "HelpersOnly"
iwr -useb https://raw.githubusercontent.com/xjwm5685-ui/cliai/main/install.ps1 | iex
```

### Linux 通过 apt 安装

先添加软件源：

```bash
curl -fsSL https://raw.githubusercontent.com/xjwm5685-ui/cliai/main/install.sh | bash
```

再安装：

```bash
sudo apt update
sudo apt install cliai
```

如果你想一步完成 apt 源、`cliai`、`zsh` 和 zsh 集成：

```bash
curl -fsSL https://raw.githubusercontent.com/xjwm5685-ui/cliai/main/install.sh | \
  env CLIAI_INSTALL_PACKAGE=1 CLIAI_INSTALL_ZSH=1 CLIAI_ENABLE_ZSH=1 bash
exec zsh
```

安装完成后，建议继续执行 shell 集成：

```bash
cliai shell install zsh
```

如果你使用 `bash`：

```bash
cliai shell install bash
```

默认快捷键：

- `Alt+RightArrow`：接受整条预测
- `Alt+Shift+RightArrow`：按词接受预测
- `Alt+f`：很多 Linux/WSL2 终端下更稳的“按词接受”兜底按键

如果系统里装了 `pwsh`，也可以启用 PowerShell 预测：

```bash
cliai shell install powershell
```

如果你只想启用 PowerShell helper：

```bash
cliai shell install powershell-helpers
```

### Linux / macOS 通用一键安装

如果你不想使用 apt，或者你在 macOS 上，可以直接安装最新 GitHub Release：

```bash
curl -fsSL https://raw.githubusercontent.com/xjwm5685-ui/cliai/main/install-unix.sh | bash
```

安装后如果你主要使用 `zsh`，推荐继续执行：

```bash
cliai shell install zsh
```

想一步完成下载、安装和 zsh 集成：

```bash
curl -fsSL https://raw.githubusercontent.com/xjwm5685-ui/cliai/main/install-unix.sh | \
  env CLIAI_ENABLE_ZSH=1 bash
exec zsh
```

### GitHub Release 手动安装

Windows：

- 下载 `cliai_Windows_x86_64.zip` 或 `cliai_Windows_ARM64.zip`
- 解压后执行 `.\cliai.exe shell install powershell`
- 如果只想要 helper，可执行 `.\cliai.exe shell install powershell-helpers`

Linux / macOS：

- 下载对应的 `cliai_Linux_*.tar.gz` 或 `cliai_macOS_*.tar.gz`
- 解压后执行 `./scripts/install-unix.sh`

### 分发状态

- 当前可用：GitHub Release、Windows 安装脚本、Linux apt 包 `cliai`
- 待公开分发：winget `Sanqiu.Cliai`、Chocolatey `sanqiu-cliai`
- 未提供现成 Homebrew tap：如需 `brew install` 仍需单独维护 tap

## 命令总览

```text
cliai predict <query>
cliai predictor serve
cliai history import
cliai config show
cliai config set <key> <value>
cliai feedback show
cliai feedback accept --query <query> <command>
cliai shell init <powershell|bash|zsh>
cliai shell install <powershell|bash|zsh>
cliai shell init powershell-helpers
cliai shell install powershell-helpers
cliai selftest
cliai version
```

## `predict` 命令

用法：

```powershell
cliai predict [flags] <query>
```

支持参数：

- `--limit <N>`：限制返回候选数，默认 `5`
- `--shell <name>`：指定 shell
- `--json`：输出 JSON
- `--cwd <path>`：指定工作目录，用于项目上下文检测
- `--debug`：输出调试信息到标准错误
- `--copy`：复制最佳命令或交互选中的命令
- `--command-only`：只输出最优命令
- `--interactive`：进入交互式选择模式

示例：

```powershell
cliai predict "git st"
cliai predict --limit 3 "安装 vscode"
cliai predict --json "搜索 docker"
cliai predict --cwd D:\code\myapp "启动"
cliai predict --debug "进入 src"
cliai predict --command-only "run tests"
cliai predict --interactive --copy "进入 src"
```

默认输出列：

- `COMMAND`
- `SOURCE`
- `RISK`
- `WHY`

风险级别：

- `safe`：低风险建议
- `caution`：安装、升级、切换分支、启动服务等
- `danger`：删除、关机、格式化等高风险命令

JSON 示例：

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

`source` 常见取值：

- `template`
- `builtin`
- `context`
- `powershell-history`

## `feedback` 命令

```powershell
cliai feedback show
cliai feedback show --json
cliai feedback accept --query "安装 vscode" winget install vscode
```

作用：

- 查看反馈记录
- 手动记录某次查询最终接受的命令
- 配合 `predict --interactive` 自动学习

## `history` 命令

```powershell
cliai history import [--file path]
```

默认历史路径：

- Windows PowerShell：`%USERPROFILE%\AppData\Roaming\Microsoft\Windows\PowerShell\PSReadLine\ConsoleHost_history.txt`
- Linux/macOS PowerShell：`~/.local/share/powershell/PSReadLine/ConsoleHost_history.txt`
- bash：`~/.bash_history`
- zsh：`~/.zsh_history`
- fish：`~/.local/share/fish/fish_history`

## `config` 命令

查看配置：

```powershell
cliai config show
```

设置配置：

```powershell
cliai config set shell powershell
cliai config set history_path D:\custom\ConsoleHost_history.txt
cliai config set local.max_history 5000
```

当前支持的配置项：

- `shell`
- `history_path`
- `local.max_history`

配置示例：

```json
{
  "shell": "powershell",
  "history_path": "C:\\Users\\YOUR_NAME\\AppData\\Roaming\\Microsoft\\Windows\\PowerShell\\PSReadLine\\ConsoleHost_history.txt",
  "local": {
    "max_history": 4000
  }
}
```

环境变量：

- `CLIAI_SHELL`
- `CLIAI_HISTORY_PATH`

## `shell` 命令

用法：

```powershell
cliai shell init powershell
cliai shell install powershell
cliai shell init powershell-helpers
cliai shell install powershell-helpers
cliai shell init zsh
cliai shell install zsh
cliai shell init bash
cliai shell install bash
```

如果你只想安装 `csg` / `csi` / `csc` 这些 PowerShell helper，而不处理完整预测器集成，也可以单独执行：

```powershell
cliai shell install powershell-helpers
```

PowerShell helper 别名：

- `csg`：普通建议
- `csi`：交互式选择并复制
- `csc`：只输出最佳命令

PowerShell 示例：

```powershell
csg "安装 vscode"
csi "git st"
csc "run tests"
```

PowerShell helper 安装与卸载：

```powershell
cliai shell install powershell-helpers
. $PROFILE
```

- helper 只安装 `csg` / `csi` / `csc`，不改动完整预测器模块
- helper 代码会写入 `$PROFILE` 中由 `# >>> cliai helpers >>>` 和 `# <<< cliai helpers <<<` 包裹的区块
- 如需卸载，删除这段标记区块并重新打开 PowerShell

## 实时预测

### PowerShell

`cliai shell install powershell` 会安装 `CliaiPredictor` 并启用实时灰字预测。

前置要求：

- PowerShell `7.2+`
- PSReadLine `2.2.2+`
- `.NET SDK 8` 仅在需要本地编译预测器模块时必需

验证方法：

```powershell
Import-Module CliaiPredictor
(Get-PSSubsystem -Kind CommandPredictor).Implementations |
  Select-Object Id, Name, Description
```

卸载或回滚：

- 删除 `$PROFILE` 中 cliai 写入的 PowerShell 集成区块
- 删除 `~/Documents/PowerShell/Modules/CliaiPredictor/<version>` 目录
- 重新打开 PowerShell，必要时再执行 `Import-Module PSReadLine`

### zsh

`zsh` 提供原生灰字预测，推荐在 Linux、macOS、WSL2 上优先使用：

```bash
cliai shell install zsh
```

### bash

`bash` 提供快捷接受预测：

```bash
cliai shell install bash
```

常用按键：

- `Alt+RightArrow`：接受整条预测
- `Alt+Shift+RightArrow`：接受一个词
- `Alt+f`：更稳定的按词接受后备方案

## 工作原理

预测流程主要分为 5 步：

1. 读取当前工作目录、shell、历史缓存和反馈记录
2. 检测项目类型，如 `go.mod`、`package.json`、`pyproject.toml`、`Dockerfile`、`.git`
3. 从内置规则、自然语言模板、项目上下文和历史命令中召回候选
4. 根据前缀、关键词、频率、近期性和反馈记录做本地排序
5. 输出表格、JSON 或进入交互式选择，并在接受后回写反馈

## 项目结构

```text
cliai/
├─ .github/
├─ docs/
├─ internal/
│  ├─ app/
│  ├─ config/
│  ├─ feedback/
│  ├─ history/
│  ├─ predict/
│  └─ project/
├─ packaging/
├─ predictor/
├─ scripts/
├─ CHANGELOG.md
├─ README.md
├─ README_EN.md
├─ go.mod
└─ main.go
```

## 开发与测试

本地运行：

```powershell
go run . version
go run . selftest --json
go run . predict "安装 vscode"
go run . predict --debug "run tests"
```

构建：

```powershell
go build -o cliai.exe .
```

如果你希望直接在仓库根目录验证最新行为，重新构建当前目录二进制：

```powershell
go build -o .\cliai.exe .
```

运行全部测试：

```powershell
go test ./...
```

预测器基准：

```powershell
go test ./internal/predict -run ^$ -bench BenchmarkPredict
```

## 发布

CI 文件：

- `.github/workflows/ci.yml`

Release 文件：

- `.github/workflows/release.yml`

推送 `v*` tag 后，Release 工作流会自动构建：

- Windows zip
- Linux/macOS tar.gz
- Linux `.deb`
- apt 仓库元数据
- SHA256

更多细节见 [RELEASE.md](file:///d:/sanqiu/cliai/docs/RELEASE.md)。

## 常见问题

### 为什么不会直接执行命令

这是刻意设计：

- 工具负责召回、排序、解释和交互选择
- 最终执行权仍然在用户手里

### 为什么历史里有些命令没被学进去

历史导入会过滤：

- 空命令
- 超长命令
- 疑似带敏感字段的命令

例如包含 `Authorization`、`Bearer`、`password`、`api_key`、`token` 的内容会被跳过。

### 为什么中文有时会乱码

通常是终端编码问题。Windows 下可尝试：

```powershell
chcp 65001
```

### `Set-PSReadLineOption -PredictionSource` 报错怎么办

如果安装 PowerShell 预测器时看到 `PredictionSource` 参数不支持、取值无效，通常是 PowerShell 或 PSReadLine 版本偏旧。

可按下面顺序排查：

```powershell
$PSVersionTable.PSVersion
Get-Module PSReadLine -ListAvailable | Select-Object Name, Version, Path
```

- 推荐使用 PowerShell `7.2+`
- 推荐使用 PSReadLine `2.2.2+`
- 如果只想先恢复 helper 工作流，可执行 `cliai shell install powershell-helpers`
- 升级 PowerShell / PSReadLine 后，重新执行 `cliai shell install powershell`

如果你是源码用户，也可以先重新构建最新可执行文件再安装：

```powershell
go build -o .\cliai.exe .
.\cliai.exe shell install powershell
```

## 许可证

本项目采用 [MIT License](./LICENSE)。
