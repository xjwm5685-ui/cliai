package predict

type CommandSpec struct {
	Command      string
	Description  string
	Keywords     []string
	ProjectTypes []string
}

func Builtins() []CommandSpec {
	return []CommandSpec{
		{Command: "winget search", Description: "Search for a package in winget", Keywords: []string{"search", "find", "package", "软件", "搜索", "查找"}},
		{Command: "winget install", Description: "Install a package with winget", Keywords: []string{"install", "package", "app", "软件", "安装"}},
		{Command: "winget upgrade --all", Description: "Upgrade all packages with winget", Keywords: []string{"upgrade", "update", "升级", "更新"}},
		{Command: "winget uninstall", Description: "Uninstall a package with winget", Keywords: []string{"uninstall", "remove", "卸载", "删除"}},
		{Command: "Get-ChildItem", Description: "List files in the current directory", Keywords: []string{"ls", "dir", "list", "文件", "目录", "查看"}},
		{Command: "Get-ChildItem -Recurse", Description: "List files recursively", Keywords: []string{"search file", "递归", "遍历"}},
		{Command: "Set-Location", Description: "Change directory", Keywords: []string{"cd", "folder", "directory", "进入", "切换目录"}},
		{Command: "Get-Content", Description: "Read a file", Keywords: []string{"cat", "read", "open file", "读取文件"}},
		{Command: "Select-String", Description: "Search inside files", Keywords: []string{"grep", "find text", "文本搜索", "查内容"}},
		{Command: "git status", Description: "Show git working tree status", Keywords: []string{"git", "status", "仓库状态"}, ProjectTypes: []string{"git"}},
		{Command: "git pull", Description: "Pull latest changes", Keywords: []string{"git", "pull", "更新代码"}, ProjectTypes: []string{"git"}},
		{Command: "git checkout -b", Description: "Create and switch to a branch", Keywords: []string{"branch", "new branch", "新分支"}, ProjectTypes: []string{"git"}},
		{Command: "git log --oneline -n 10", Description: "Show a concise git log", Keywords: []string{"history", "commit", "提交记录"}, ProjectTypes: []string{"git"}},
		{Command: "npm install", Description: "Install npm dependencies", Keywords: []string{"npm", "install", "依赖安装"}, ProjectTypes: []string{"node"}},
		{Command: "npm run dev", Description: "Start npm development server", Keywords: []string{"dev server", "run dev", "启动前端"}, ProjectTypes: []string{"node"}},
		{Command: "pnpm install", Description: "Install pnpm dependencies", Keywords: []string{"pnpm", "install", "依赖安装"}, ProjectTypes: []string{"node"}},
		{Command: "pnpm dev", Description: "Start pnpm development server", Keywords: []string{"pnpm", "dev", "启动前端"}, ProjectTypes: []string{"node"}},
		{Command: "go test ./...", Description: "Run Go tests", Keywords: []string{"go", "test", "测试", "单元测试"}, ProjectTypes: []string{"go"}},
		{Command: "go build ./...", Description: "Build Go packages", Keywords: []string{"go", "build", "编译"}, ProjectTypes: []string{"go"}},
		{Command: "go run .", Description: "Run the current Go project", Keywords: []string{"go", "run", "启动", "运行项目"}, ProjectTypes: []string{"go"}},
		{Command: "python -m venv .venv", Description: "Create a Python virtual environment", Keywords: []string{"python", "venv", "虚拟环境"}, ProjectTypes: []string{"python"}},
		{Command: "pip install -r requirements.txt", Description: "Install Python dependencies", Keywords: []string{"pip", "install", "python deps"}, ProjectTypes: []string{"python"}},
		{Command: "docker ps", Description: "List running containers", Keywords: []string{"docker", "container", "容器"}, ProjectTypes: []string{"docker"}},
		{Command: "docker compose up -d", Description: "Start services in detached mode", Keywords: []string{"docker compose", "启动服务"}, ProjectTypes: []string{"docker"}},
		{Command: "code .", Description: "Open current folder in VS Code", Keywords: []string{"vscode", "open editor", "打开项目"}},
	}
}
