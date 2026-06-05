package runtime

import (
	"archive/zip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadTo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("HELLO-MODEL"))
	}))
	defer srv.Close()
	dest := filepath.Join(t.TempDir(), "out.bin")
	if err := downloadTo(context.Background(), srv.URL, dest, "test", nil); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(dest); string(b) != "HELLO-MODEL" {
		t.Fatalf("contenu téléchargé = %q", b)
	}
}

func TestEnsureDefaultModelSkipsIfPresent(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, defaultModelName)
	if err := os.WriteFile(dest, []byte("present"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := EnsureDefaultModel(context.Background(), dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != dest {
		t.Fatalf("got %q, want %q", got, dest)
	}
}

func TestUnzipIntoFlattens(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "t.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	if w, _ := zw.Create("some/sub/llama-server.exe"); w != nil {
		_, _ = w.Write([]byte("FAKEEXE"))
	}
	if w, _ := zw.Create("ggml-cuda.dll"); w != nil {
		_, _ = w.Write([]byte("DLL"))
	}
	zw.Close()
	f.Close()

	dest := t.TempDir()
	if err := unzipInto(zipPath, dest); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(dest, "llama-server.exe")); err != nil || string(b) != "FAKEEXE" {
		t.Fatalf("llama-server.exe aplati manquant/incorrect: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "ggml-cuda.dll")); err != nil {
		t.Fatalf("ggml-cuda.dll manquant: %v", err)
	}
}

func TestEnsureLlamaServerSkipsIfPresent(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "llama-cuda-"+llamaBuild)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	server := filepath.Join(dir, "llama-server.exe")
	if err := os.WriteFile(server, []byte("present"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := ensureLlamaServerIn(context.Background(), base, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != server {
		t.Fatalf("got %q, want %q", got, server)
	}
}
