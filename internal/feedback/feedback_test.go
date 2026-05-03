package feedback

import (
	"path/filepath"
	"testing"
)

func TestRecordAndBonus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "feedback.json")
	if err := Record(path, "安装 vscode", "winget install vscode"); err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if err := Record(path, "安装 vscode", "winget install vscode"); err != nil {
		t.Fatalf("Record returned error: %v", err)
	}

	entries, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(entries) != 1 || entries[0].Count != 2 {
		t.Fatalf("unexpected entries: %#v", entries)
	}

	bonuses := CommandBonuses("安装 vscode", entries)
	if bonuses["winget install vscode"] <= 0 {
		t.Fatalf("expected positive bonus, got %#v", bonuses)
	}
}
