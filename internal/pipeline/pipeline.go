// Package pipeline orchestre l'extraction, la traduction par lots avec
// contexte, le réassemblage et le remux d'une piste de sous-titres.
package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Ne9ative/AISubtitle/internal/engine"
	"github.com/Ne9ative/AISubtitle/internal/subs"
)

// Extractor = sous-ensemble de mkv.Tool utilisé par le pipeline.
type Extractor interface {
	ExtractTrack(ctx context.Context, video string, trackID int, out string) error
	Mux(ctx context.Context, video, sub, out, lang, trackName string) error
}

// Reporter reçoit la progression et les messages de journal.
type Reporter interface {
	Progress(done, total int)
	Log(msg string)
}

// Options décrit un job de traduction.
type Options struct {
	VideoPath   string
	TrackID     int
	SrcLang     string
	BatchSize   int
	ContextSize int
	TestMode    bool
}

const (
	testDuration   = 20 * time.Second
	frenchTrackTag = "Français (IA)"
)

// Run exécute le pipeline complet et renvoie le chemin de la vidéo produite.
func Run(ctx context.Context, ex Extractor, tr engine.Translator, opts Options, rep Reporter) (string, error) {
	if opts.BatchSize < 1 {
		opts.BatchSize = 12
	}

	tmpSrt, err := tempFile("aisub-extract-*.srt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpSrt)

	rep.Log("Extraction de la piste de sous-titres…")
	if err := ex.ExtractTrack(ctx, opts.VideoPath, opts.TrackID, tmpSrt); err != nil {
		return "", err
	}

	sub, err := subs.Open(tmpSrt)
	if err != nil {
		return "", err
	}
	if opts.TestMode {
		sub.LimitToDuration(testDuration)
		rep.Log("Mode Test : limité aux 20 premières secondes.")
	}

	texts := sub.Texts()
	total := len(texts)
	if total == 0 {
		return "", fmt.Errorf("pipeline: aucun sous-titre à traduire")
	}

	rep.Log(fmt.Sprintf("Traduction de %d lignes…", total))
	batches := subs.MakeBatches(texts, opts.BatchSize, opts.ContextSize)
	translated := make([]string, 0, total)
	done := 0
	for _, b := range batches {
		if err := ctx.Err(); err != nil {
			return "", err // annulation
		}
		out, err := tr.Translate(ctx, b.Lines, b.Context, opts.SrcLang)
		if err != nil {
			rep.Log(fmt.Sprintf("Lot incertain (%v) — repli ligne par ligne.", err))
			out = translateLineByLine(ctx, tr, b, opts.SrcLang, rep)
		}
		translated = append(translated, out...)
		done += len(b.Lines)
		rep.Progress(done, total)
	}

	if err := sub.SetTexts(translated); err != nil {
		return "", err
	}
	outSrt, err := tempFile("aisub-final-*.srt")
	if err != nil {
		return "", err
	}
	defer os.Remove(outSrt)
	if err := sub.SaveSRT(outSrt); err != nil {
		return "", err
	}

	outPath := OutputPath(opts.VideoPath, opts.TestMode)
	rep.Log("Fusion dans la vidéo (remux)…")
	if err := ex.Mux(ctx, opts.VideoPath, outSrt, outPath, "fre", frenchTrackTag); err != nil {
		return "", err
	}
	return outPath, nil
}

// translateLineByLine traduit chaque ligne séparément ; si une ligne échoue,
// on conserve son texte original pour ne pas perdre tout le job.
func translateLineByLine(ctx context.Context, tr engine.Translator, b subs.Batch, srcLang string, rep Reporter) []string {
	out := make([]string, len(b.Lines))
	for i, line := range b.Lines {
		res, err := tr.Translate(ctx, []string{line}, b.Context, srcLang)
		if err != nil || len(res) != 1 {
			rep.Log(fmt.Sprintf("Ligne non traduite, original conservé : %q", line))
			out[i] = line
			continue
		}
		out[i] = res[0]
	}
	return out
}

// OutputPath calcule le nom du fichier de sortie (suffixe _PRO_FR ou _TEST_20s).
func OutputPath(video string, testMode bool) string {
	dir := filepath.Dir(video)
	base := strings.TrimSuffix(filepath.Base(video), filepath.Ext(video))
	base = stripKnownSuffixes(base)
	suffix := "_PRO_FR"
	if testMode {
		suffix = "_TEST_20s"
	}
	return filepath.Join(dir, base+suffix+".mkv")
}

func stripKnownSuffixes(base string) string {
	for _, s := range []string{"_PRO_FR", "_TEST_20s"} {
		base = strings.ReplaceAll(base, s, "")
	}
	return base
}

func tempFile(pattern string) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	name := f.Name()
	_ = f.Close()
	return name, nil
}
