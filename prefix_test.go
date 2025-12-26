package wine

import (
	"os"
	"path/filepath"
	"testing"
)

var testPfx = New(filepath.Join(os.TempDir(), "wine_prefix_test"), "")

func TestMain(m *testing.M) {
	code := m.Run()
	_ = testPfx.Kill()
	os.Exit(code)
}

func TestPrefixInit(t *testing.T) {
	if testPfx.Exists() {
		t.SkipNow()
	}

	if err := os.Mkdir(testPfx.Dir(), 0o755); err != nil {
		t.Errorf("unexpected dir error: %v", err)
	}

	if err := testPfx.Init().Run(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrefixRunning(t *testing.T) {
	if testPfx.Running() {
		t.Log("wineserver running on system, killing")
		if err := testPfx.Kill(); err != nil {
			t.Errorf("unexpected running kill error: %v", err)
		}
	}
	if testPfx.Running() {
		t.Fatal("expected wineserver death")
	}
	if err := testPfx.Start(); err != nil {
		t.Errorf("unexpected start error: %v", err)
	}
	if !testPfx.Running() {
		t.Fatal("expected wineserver alive")
	}

	t.Run("GUI application", func(t *testing.T) {
		if err := testPfx.Wine("regedit").Run(); err != nil {
			t.Errorf("unexpected start error: %v", err)
		}
	})

	if err := testPfx.Kill(); err != nil {
		t.Errorf("unexpected kill error: %v", err)
	}
}
