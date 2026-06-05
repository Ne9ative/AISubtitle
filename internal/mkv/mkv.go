// Package mkv enveloppe les outils MKVToolNix (mkvmerge / mkvextract)
// pour scanner, extraire et remuxer des pistes de sous-titres.
package mkv

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Ne9ative/AISubtitle/internal/winproc"
)

// Track décrit une piste de sous-titres d'un conteneur.
type Track struct {
	ID           int
	Type         string
	Codec        string
	CodecID      string
	Language     string
	Name         string
	IsImageBased bool // PGS/VobSub... : non traduisible en texte
}

// Tool référence les binaires MKVToolNix à utiliser.
type Tool struct {
	Mkvmerge   string
	Mkvextract string
}

// codecs de sous-titres image (non traduisibles sans OCR).
var imageCodecIDs = map[string]bool{
	"S_HDMV/PGS":    true,
	"S_HDMV/TEXTST": true,
	"S_VOBSUB":      true,
	"S_DVBSUB":      true,
}

// structure partielle de la sortie `mkvmerge -J`.
type mkvmergeJSON struct {
	Tracks []struct {
		ID         int    `json:"id"`
		Type       string `json:"type"`
		Codec      string `json:"codec"`
		Properties struct {
			Language  string `json:"language"`
			TrackName string `json:"track_name"`
			CodecID   string `json:"codec_id"`
		} `json:"properties"`
	} `json:"tracks"`
}

func parseTracksJSON(data []byte) ([]Track, error) {
	var doc mkvmergeJSON
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("mkv: JSON mkvmerge illisible: %w", err)
	}
	var tracks []Track
	for _, t := range doc.Tracks {
		if t.Type != "subtitles" {
			continue
		}
		lang := t.Properties.Language
		if lang == "" {
			lang = "und"
		}
		tracks = append(tracks, Track{
			ID:           t.ID,
			Type:         t.Type,
			Codec:        t.Codec,
			CodecID:      t.Properties.CodecID,
			Language:     lang,
			Name:         t.Properties.TrackName,
			IsImageBased: imageCodecIDs[strings.ToUpper(t.Properties.CodecID)],
		})
	}
	return tracks, nil
}

// ScanSubtitleTracks liste les pistes de sous-titres d'un conteneur.
func (t Tool) ScanSubtitleTracks(ctx context.Context, video string) ([]Track, error) {
	out, err := run(ctx, t.Mkvmerge, "-J", video)
	if err != nil {
		return nil, err
	}
	return parseTracksJSON(out)
}

// ExtractTrack extrait la piste trackID vers le fichier out.
func (t Tool) ExtractTrack(ctx context.Context, video string, trackID int, out string) error {
	_, err := run(ctx, t.Mkvextract, video, "tracks", fmt.Sprintf("%d:%s", trackID, out))
	return err
}

// Mux crée out = video + la piste sub (langue lang, marquée par défaut).
func (t Tool) Mux(ctx context.Context, video, sub, out, lang, trackName string) error {
	_, err := run(ctx, t.Mkvmerge,
		"-o", out,
		video,
		"--language", "0:"+lang,
		"--track-name", "0:"+trackName,
		"--default-track", "0:yes",
		sub,
	)
	return err
}

// run lance un binaire MKVToolNix sans fenêtre console. Les outils
// MKVToolNix renvoient 0 (succès), 1 (avertissements, sortie produite)
// ou >=2 (vraie erreur) : on ne considère échec que >=2.
func run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "" {
		return nil, fmt.Errorf("mkv: binaire non configuré")
	}
	cmd := exec.CommandContext(ctx, name, args...)
	winproc.HideWindow(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return stdout.Bytes(), nil // avertissements uniquement
		}
		return stdout.Bytes(), fmt.Errorf("mkv: %s: %v: %s",
			filepath.Base(name), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}
