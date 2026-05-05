package app

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	powerShellHelpersStartMarker = "# >>> cliai helpers >>>"
	powerShellHelpersEndMarker   = "# <<< cliai helpers <<<"
)

func runShellInstallPowerShell(stdout io.Writer, stderr io.Writer) int {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(stderr, "resolve executable path: %v\n", err)
		return 1
	}

	scriptPath := findInstallPowerShellScript(exePath)
	if scriptPath == "" {
		fmt.Fprintln(stderr, "install-powershell.ps1 not found next to the installed cliai files")
		fmt.Fprintln(stderr, "expected packaged assets under scripts/ and modules/")
		return 1
	}

	host, err := powerShellHost()
	if err != nil {
		fmt.Fprintf(stderr, "find PowerShell host: %v\n", err)
		return 1
	}

	cmd := exec.Command(host, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath, "-ExeName", exePath)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(stderr, "run PowerShell installer: %v\n", err)
		return 1
	}
	return 0
}

func runShellInstallPowerShellHelpers(stdout io.Writer, stderr io.Writer) int {
	profilePath, err := powerShellProfilePath(os.UserHomeDir)
	if err != nil {
		fmt.Fprintf(stderr, "resolve PowerShell profile path: %v\n", err)
		return 1
	}
	if err := installPowerShellHelpers(profilePath); err != nil {
		fmt.Fprintf(stderr, "install PowerShell helpers: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Installed cliai PowerShell helpers to %s\n", profilePath)
	fmt.Fprintln(stdout, "Open a new pwsh session, or reload your profile with:")
	fmt.Fprintf(stdout, "  . %s\n", profilePath)
	fmt.Fprintln(stdout, "Helper aliases: csg, csi, csc")
	return 0
}

func findInstallPowerShellScript(exePath string) string {
	exeDir := filepath.Dir(exePath)
	cwd, _ := os.Getwd()

	candidates := []string{
		filepath.Join(exeDir, "scripts", "install-powershell.ps1"),
		filepath.Join(filepath.Dir(exeDir), "scripts", "install-powershell.ps1"),
		filepath.Join(cwd, "scripts", "install-powershell.ps1"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

func powerShellProfilePath(userHomeDir func() (string, error)) (string, error) {
	home, err := userHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Documents", "PowerShell", "Profile.ps1"), nil
}

func installPowerShellHelpers(profilePath string) error {
	return upsertMarkedBlock(profilePath, powerShellHelpersStartMarker, powerShellHelpersEndMarker, powershellSnippet())
}

func upsertMarkedBlock(filePath string, startMarker string, endMarker string, block string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}

	existing, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	content := string(existing)
	markedBlock := startMarker + "\n" + block + "\n" + endMarker + "\n"
	startIndex := strings.Index(content, startMarker)
	if startIndex >= 0 {
		endIndex := strings.Index(content[startIndex:], endMarker)
		if endIndex >= 0 {
			endIndex = startIndex + endIndex + len(endMarker)
			updated := content[:startIndex] + markedBlock
			if endIndex < len(content) {
				updated += strings.TrimLeft(content[endIndex:], "\r\n")
			}
			return os.WriteFile(filePath, []byte(updated), 0o644)
		}
	}

	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if content != "" {
		content += "\n"
	}
	content += markedBlock
	return os.WriteFile(filePath, []byte(content), 0o644)
}

func powerShellHost() (string, error) {
	for _, name := range []string{"pwsh", "powershell"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("pwsh or powershell not found in PATH")
}
