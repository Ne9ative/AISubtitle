package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheDirContainsAppName(t *testing.T) {
	dir, err := CacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dir, appName) {
		t.Fatalf("cache dir %q devrait contenir %q", dir, appName)
	}
	if filepath.Base(dir) != "bin" {
		t.Fatalf("cache dir devrait finir par 'bin', got %q", dir)
	}
}

func TestEnsureCacheDirCreates(t *testing.T) {
	dir, err := EnsureCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("le dossier devrait exister: %v", err)
	}
	want, err := CacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != want {
		t.Fatalf("EnsureCacheDir=%q != CacheDir=%q", dir, want)
	}
}
