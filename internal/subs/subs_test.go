package subs

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const sampleSRT = `1
00:00:01,000 --> 00:00:04,000
Hello world

2
00:00:05,000 --> 00:00:07,000
Second line
with two rows
`

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOpenAndTexts(t *testing.T) {
	s, err := Open(writeTemp(t, "in.srt", sampleSRT))
	if err != nil {
		t.Fatal(err)
	}
	if s.Len() != 2 {
		t.Fatalf("attendu 2 cues, got %d", s.Len())
	}
	want := []string{"Hello world", "Second line with two rows"}
	if got := s.Texts(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Texts() = %v, attendu %v", got, want)
	}
}

func TestSetTextsCountMismatch(t *testing.T) {
	s, _ := Open(writeTemp(t, "in.srt", sampleSRT))
	if err := s.SetTexts([]string{"un seul"}); err == nil {
		t.Fatal("attendu une erreur pour un mauvais nombre de traductions")
	}
}

func TestSetTextsAndSaveRoundtrip(t *testing.T) {
	s, _ := Open(writeTemp(t, "in.srt", sampleSRT))
	if err := s.SetTexts([]string{"Bonjour le monde", "Deuxieme ligne"}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.srt")
	if err := s.SaveSRT(out); err != nil {
		t.Fatal(err)
	}
	s2, err := Open(out)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Bonjour le monde", "Deuxieme ligne"}
	if got := s2.Texts(); !reflect.DeepEqual(got, want) {
		t.Fatalf("apres sauvegarde, Texts() = %v, attendu %v", got, want)
	}
}
