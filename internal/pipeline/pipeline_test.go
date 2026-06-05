package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type mockExtractor struct {
	srtContent string
	muxCalled  bool
	muxContent string
}

func (m *mockExtractor) ExtractTrack(ctx context.Context, video string, trackID int, out string) error {
	return os.WriteFile(out, []byte(m.srtContent), 0o644)
}
func (m *mockExtractor) Mux(ctx context.Context, video, sub, out, lang, trackName string) error {
	m.muxCalled = true
	// Capturer le contenu pendant l'appel : le pipeline supprime le temp ensuite (defer).
	data, _ := os.ReadFile(sub)
	m.muxContent = string(data)
	return os.WriteFile(out, []byte("fake mkv"), 0o644)
}

// mockTranslator préfixe chaque ligne par "FR:".
type mockTranslator struct{}

func (mockTranslator) Translate(ctx context.Context, lines, ctxLines []string, srcLang string) ([]string, error) {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = "FR:" + l
	}
	return out, nil
}
func (mockTranslator) Close() error { return nil }

// pickyTranslator échoue sur les lots multi-lignes, réussit en solo (teste le repli).
type pickyTranslator struct{}

func (pickyTranslator) Translate(ctx context.Context, lines, ctxLines []string, srcLang string) ([]string, error) {
	if len(lines) > 1 {
		return nil, fmt.Errorf("lot refusé")
	}
	return []string{"FR:" + lines[0]}, nil
}
func (pickyTranslator) Close() error { return nil }

type mockReporter struct {
	lastDone, lastTotal, logs int
}

func (m *mockReporter) Progress(done, total int) { m.lastDone, m.lastTotal = done, total }
func (m *mockReporter) Log(string)               { m.logs++ }

const testSRT = `1
00:00:01,000 --> 00:00:04,000
Hello

2
00:00:05,000 --> 00:00:07,000
World
`

func TestRunHappyPath(t *testing.T) {
	ex := &mockExtractor{srtContent: testSRT}
	rep := &mockReporter{}
	video := filepath.Join(t.TempDir(), "episode.mkv")
	out, err := Run(context.Background(), ex, mockTranslator{},
		Options{VideoPath: video, TrackID: 0, SrcLang: "ANGLAIS", BatchSize: 10, ContextSize: 2}, rep)
	if err != nil {
		t.Fatal(err)
	}
	if !ex.muxCalled {
		t.Fatal("Mux n'a pas été appelé")
	}
	if rep.lastDone != 2 || rep.lastTotal != 2 {
		t.Fatalf("progression = %d/%d, attendu 2/2", rep.lastDone, rep.lastTotal)
	}
	if !strings.Contains(ex.muxContent, "FR:Hello") || !strings.Contains(ex.muxContent, "FR:World") {
		t.Fatalf("SRT traduit inattendu:\n%s", ex.muxContent)
	}
	if filepath.Base(out) != "episode_PRO_FR.mkv" {
		t.Fatalf("sortie = %s", out)
	}
}

func TestRunFallbackLineByLine(t *testing.T) {
	ex := &mockExtractor{srtContent: testSRT}
	rep := &mockReporter{}
	video := filepath.Join(t.TempDir(), "ep.mkv")
	out, err := Run(context.Background(), ex, pickyTranslator{},
		Options{VideoPath: video, TrackID: 0, SrcLang: "ANGLAIS", BatchSize: 10}, rep)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ex.muxContent, "FR:Hello") || !strings.Contains(ex.muxContent, "FR:World") {
		t.Fatalf("le repli ligne par ligne a échoué:\n%s", ex.muxContent)
	}
	_ = out
}

func TestRunTestModeOutputName(t *testing.T) {
	ex := &mockExtractor{srtContent: testSRT}
	video := filepath.Join(t.TempDir(), "ep.mp4")
	out, err := Run(context.Background(), ex, mockTranslator{},
		Options{VideoPath: video, SrcLang: "ANGLAIS", BatchSize: 10, TestMode: true}, &mockReporter{})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(out) != "ep_TEST_20s.mkv" {
		t.Fatalf("sortie test = %s", out)
	}
}

func TestOutputPath(t *testing.T) {
	cases := []struct {
		video    string
		testMode bool
		want     string
	}{
		{"/x/y/Show.mkv", false, "Show_PRO_FR.mkv"},
		{"/x/y/Show.mp4", true, "Show_TEST_20s.mkv"},
		{"/x/Show_PRO_FR.mkv", false, "Show_PRO_FR.mkv"}, // pas de double suffixe
	}
	for _, c := range cases {
		if got := filepath.Base(OutputPath(c.video, c.testMode)); got != c.want {
			t.Fatalf("OutputPath(%q,%v)=%q, attendu %q", c.video, c.testMode, got, c.want)
		}
	}
}
