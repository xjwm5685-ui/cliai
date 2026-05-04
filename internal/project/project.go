package project

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Context struct {
	CWD            string   `json:"cwd"`
	ProjectTypes   []string `json:"project_types"`
	PackageManager string   `json:"package_manager,omitempty"`
	IsGitRepo      bool     `json:"is_git_repo"`
	Files          []string `json:"files"`
	Directories    []string `json:"directories"`
}

func Detect(cwd string) (Context, error) {
	ctx := Context{CWD: cwd}
	if strings.TrimSpace(cwd) == "" {
		return ctx, nil
	}

	entries, err := os.ReadDir(cwd)
	if err != nil {
		return ctx, err
	}

	typeSet := map[string]struct{}{}
	markCurrentDirContext(&ctx, typeSet, entries)
	if err := markAncestorProjectContext(&ctx, typeSet, cwd); err != nil {
		return ctx, err
	}

	for kind := range typeSet {
		ctx.ProjectTypes = append(ctx.ProjectTypes, kind)
	}

	sort.Strings(ctx.ProjectTypes)
	sort.Strings(ctx.Files)
	sort.Strings(ctx.Directories)

	if ctx.PackageManager == "" && contains(ctx.ProjectTypes, "node") {
		ctx.PackageManager = "npm"
	}

	return ctx, nil
}

func markCurrentDirContext(ctx *Context, typeSet map[string]struct{}, entries []os.DirEntry) {
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			ctx.Directories = append(ctx.Directories, name)
			if name == ".git" {
				ctx.IsGitRepo = true
				typeSet["git"] = struct{}{}
			}
			continue
		}

		ctx.Files = append(ctx.Files, name)
		applyProjectMarker(ctx, typeSet, name)
	}
}

func markAncestorProjectContext(ctx *Context, typeSet map[string]struct{}, cwd string) error {
	current := filepath.Clean(cwd)
	parent := filepath.Dir(current)
	for parent != current {
		entries, err := os.ReadDir(parent)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			applyProjectMarker(ctx, typeSet, entry.Name())
		}
		current = parent
		parent = filepath.Dir(current)
	}
	return nil
}

func applyProjectMarker(ctx *Context, typeSet map[string]struct{}, name string) {
	switch strings.ToLower(name) {
	case "go.mod":
		typeSet["go"] = struct{}{}
	case "package.json":
		typeSet["node"] = struct{}{}
	case "pnpm-lock.yaml":
		typeSet["node"] = struct{}{}
		if ctx.PackageManager == "" {
			ctx.PackageManager = "pnpm"
		}
	case "package-lock.json":
		typeSet["node"] = struct{}{}
		if ctx.PackageManager == "" {
			ctx.PackageManager = "npm"
		}
	case "yarn.lock":
		typeSet["node"] = struct{}{}
		if ctx.PackageManager == "" {
			ctx.PackageManager = "yarn"
		}
	case "requirements.txt", "pyproject.toml":
		typeSet["python"] = struct{}{}
	case "dockerfile", "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml":
		typeSet["docker"] = struct{}{}
	case ".git":
		ctx.IsGitRepo = true
		typeSet["git"] = struct{}{}
	}
}

func MatchDirectory(ctx Context, token string) string {
	token = normalize(token)
	if token == "" {
		return ""
	}
	for _, name := range ctx.Directories {
		if strings.Contains(normalize(name), token) {
			return filepath.Clean(name)
		}
	}
	return ""
}

func MatchFile(ctx Context, token string) string {
	token = normalize(token)
	if token == "" {
		return ""
	}
	for _, name := range ctx.Files {
		if strings.Contains(normalize(name), token) {
			return filepath.Clean(name)
		}
	}
	return ""
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func normalize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer("\\", " ", "/", " ", "_", " ", "-", " ").Replace(value)
	return strings.Join(strings.Fields(value), " ")
}
