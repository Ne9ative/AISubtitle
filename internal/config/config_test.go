package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	c := Default()
	if c.Mode != "Local" || c.SourceLang != "ANGLAIS" || c.BatchSize != 12 || c.ContextSize != 2 {
		t.Fatalf("defaults inattendus: %+v", c)
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	in := Config{Mode: "Gemini", Model: "gemini-2.0-flash", APIKey: "k", SourceLang: "JAPONAIS", BatchSize: 5, ContextSize: 1}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("roundtrip: got %+v want %+v", out, in)
	}
}

func TestLoadMissingReturnsDefault(t *testing.T) {
	out, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatalf("fichier absent: erreur inattendue %v", err)
	}
	if out != Default() {
		t.Fatalf("attendu défaut, got %+v", out)
	}
}

func TestLoadInvalidReturnsDefaultAndError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{pas du json"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := Load(path)
	if err == nil {
		t.Fatal("attendu une erreur pour JSON invalide")
	}
	if out != Default() {
		t.Fatalf("attendu défaut sur invalide, got %+v", out)
	}
}
