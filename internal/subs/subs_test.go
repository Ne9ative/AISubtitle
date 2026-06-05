package subs

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
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

const sampleASS = `[Script Info]
Title: Test
ScriptType: v4.00+

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Arial,20,&H00FFFFFF,&H000000FF,&H00000000,&H00000000,0,0,0,0,100,100,0,0,1,2,2,2,10,10,10,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
Dialogue: 0,0:00:01.00,0:00:04.00,Default,,0,0,0,,{\pos(300,900)}Hello {\i1}world{\i0}
Dialogue: 0,0:00:05.00,0:00:07.00,Default,,0,0,0,,Second line
`

// Un fichier nommé .srt mais au contenu ASS doit être détecté au contenu.
func TestOpenASSByContent(t *testing.T) {
	s, err := Open(writeTemp(t, "track.srt", sampleASS))
	if err != nil {
		t.Fatal(err)
	}
	if s.Len() != 2 {
		t.Fatalf("attendu 2 cues ASS, got %d", s.Len())
	}
	want := []string{"Hello world", "Second line"}
	if got := s.Texts(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Texts() ASS = %v, attendu %v", got, want)
	}
}

// Une source ASS doit ressortir en ASS, avec son positionnement {\pos} intact.
func TestASSPreservesPositioning(t *testing.T) {
	s, err := Open(writeTemp(t, "in.ass", sampleASS))
	if err != nil {
		t.Fatal(err)
	}
	if s.Ext() != ".ass" {
		t.Fatalf("Ext() = %q, attendu .ass", s.Ext())
	}
	if err := s.SetTexts([]string{"Bonjour le monde", "Deuxieme ligne"}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.ass")
	if err := s.Save(out); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), `\pos(300,900)`) {
		t.Fatalf("positionnement \\pos perdu:\n%s", data)
	}
	if !strings.Contains(string(data), "Bonjour le monde") {
		t.Fatalf("traduction absente:\n%s", data)
	}
	s2, err := Open(out)
	if err != nil {
		t.Fatal(err)
	}
	if got := s2.Texts(); got[0] != "Bonjour le monde" {
		t.Fatalf("relecture: %v", got)
	}
}

func TestLimitToDuration(t *testing.T) {
	s, _ := Open(writeTemp(t, "in.srt", sampleSRT)) // cues à 1s et 5s
	s.LimitToDuration(3 * time.Second)
	if s.Len() != 1 {
		t.Fatalf("attendu 1 cue après limite 3s, got %d", s.Len())
	}
}
