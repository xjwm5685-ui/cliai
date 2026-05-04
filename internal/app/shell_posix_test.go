package app

import (
	"strings"
	"testing"
)

func TestShellInitSnippetSupportsBashAndZsh(t *testing.T) {
	bash, err := shellInitSnippet("bash")
	if err != nil {
		t.Fatalf("shellInitSnippet(bash) returned error: %v", err)
	}
	if !strings.Contains(bash, `cliai predict --shell bash --command-only`) {
		t.Fatalf("bash snippet missing command-only predictor call")
	}

	zsh, err := shellInitSnippet("zsh")
	if err != nil {
		t.Fatalf("shellInitSnippet(zsh) returned error: %v", err)
	}
	if !strings.Contains(zsh, `region_highlight=`) {
		t.Fatalf("zsh snippet missing region_highlight integration")
	}
}

func TestShellIntegrationBlockUsesShellSpecificMarkers(t *testing.T) {
	bashBlock := shellIntegrationBlock("bash", "echo bash")
	if !strings.Contains(bashBlock, bashIntegrationMarker) {
		t.Fatalf("bash block missing bash marker")
	}

	zshBlock := shellIntegrationBlock("zsh", "echo zsh")
	if !strings.Contains(zshBlock, zshIntegrationMarker) {
		t.Fatalf("zsh block missing zsh marker")
	}
	if !strings.Contains(zshBlock, integrationEndMarker) {
		t.Fatalf("zsh block missing end marker")
	}
}
