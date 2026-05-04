package app

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

func powerShellHost() (string, error) {
	for _, name := range []string{"pwsh", "powershell"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("pwsh or powershell not found in PATH")
}
