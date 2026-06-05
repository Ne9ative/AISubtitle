package mkv

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const sampleMkvmergeJSON = `{
  "tracks": [
    {"id": 0, "type": "video", "codec": "AVC/H.264", "properties": {"language": "und"}},
    {"id": 1, "type": "audio", "codec": "AAC", "properties": {"language": "jpn"}},
    {"id": 2, "type": "subtitles", "codec": "SubRip/SRT", "properties": {"language": "eng", "track_name": "English", "codec_id": "S_TEXT/UTF8"}},
    {"id": 3, "type": "subtitles", "codec": "HDMV PGS", "properties": {"language": "eng", "codec_id": "S_HDMV/PGS"}},
    {"id": 4, "type": "subtitles", "codec": "SubStationAlpha", "properties": {"codec_id": "S_TEXT/ASS"}}
  ]
}`

func TestParseTracksJSON(t *testing.T) {
	tracks, err := parseTracksJSON([]byte(sampleMkvmergeJSON))
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 3 {
		t.Fatalf("attendu 3 pistes sous-titres, got %d", len(tracks))
	}
	if tracks[0].ID != 2 || tracks[0].Language != "eng" || tracks[0].IsImageBased {
		t.Fatalf("piste 0 inattendue: %+v", tracks[0])
	}
	if !tracks[1].IsImageBased {
		t.Fatalf("la piste PGS devrait être image: %+v", tracks[1])
	}
	if tracks[2].Language != "und" {
		t.Fatalf("langue manquante devrait être 'und', got %q", tracks[2].Language)
	}
}

func TestParseTracksJSONInvalid(t *testing.T) {
	if _, err := parseTracksJSON([]byte("{pas du json")); err == nil {
		t.Fatal("attendu une erreur pour JSON invalide")
	}
}

func TestParseTracksJSONNoSubs(t *testing.T) {
	tracks, err := parseTracksJSON([]byte(`{"tracks":[{"id":0,"type":"video","properties":{}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 0 {
		t.Fatalf("attendu 0 piste, got %d", len(tracks))
	}
}

// Test d'intégration : nécessite les binaires MKVToolNix.
// Définir AISUBTITLE_MKVTOOLNIX_DIR (dossier contenant mkvmerge.exe / mkvextract.exe).
func TestIntegrationScanExtract(t *testing.T) {
	dir := os.Getenv("AISUBTITLE_MKVTOOLNIX_DIR")
	if dir == "" {
		t.Skip("AISUBTITLE_MKVTOOLNIX_DIR non défini : test d'intégration ignoré")
	}
	tool := Tool{
		Mkvmerge:   filepath.Join(dir, "mkvmerge.exe"),
		Mkvextract: filepath.Join(dir, "mkvextract.exe"),
	}
	tmp := t.TempDir()
	srt := filepath.Join(tmp, "src.srt")
	if err := os.WriteFile(srt, []byte("1\n00:00:01,000 --> 00:00:03,000\nHello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mkvPath := filepath.Join(tmp, "test.mkv")
	// Fixture : créer un MKV à partir du SRT (mkvmerge peut renvoyer 1 = avertissements).
	if out, err := exec.Command(tool.Mkvmerge, "-o", mkvPath, srt).CombinedOutput(); err != nil {
		if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() >= 2 {
			t.Fatalf("création fixture: %v: %s", err, out)
		}
	}
	tracks, err := tool.ScanSubtitleTracks(context.Background(), mkvPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 {
		t.Fatalf("attendu 1 piste sous-titre, got %d", len(tracks))
	}
	if tracks[0].IsImageBased {
		t.Fatal("la piste SRT ne devrait pas être image")
	}
	outSrt := filepath.Join(tmp, "out.srt")
	if err := tool.ExtractTrack(context.Background(), mkvPath, tracks[0].ID, outSrt); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outSrt); err != nil {
		t.Fatalf("extraction sans fichier de sortie: %v", err)
	}
}
