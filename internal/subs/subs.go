package subs

import (
	"fmt"
	"strings"
	"time"

	astisub "github.com/asticode/go-astisub"
)

// Subtitle encapsule un document de sous-titres pour la traduction.
type Subtitle struct {
	doc *astisub.Subtitles
}

// Open lit un fichier de sous-titres. Le format est détecté par
// l'extension (.srt, .ass/.ssa, .vtt...).
func Open(path string) (*Subtitle, error) {
	doc, err := astisub.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("subs: ouverture %q: %w", path, err)
	}
	return &Subtitle{doc: doc}, nil
}

// Len renvoie le nombre de sous-titres (cues).
func (s *Subtitle) Len() int { return len(s.doc.Items) }

// Texts renvoie le texte source de chaque cue, dans l'ordre. Les retours
// à la ligne internes sont aplatis en espaces (texte propre pour le modèle).
func (s *Subtitle) Texts() []string {
	out := make([]string, len(s.doc.Items))
	for i, it := range s.doc.Items {
		out[i] = itemText(it)
	}
	return out
}

// SetTexts remplace le texte de chaque cue par sa traduction.
// len(translated) doit être égal à Len().
func (s *Subtitle) SetTexts(translated []string) error {
	if len(translated) != len(s.doc.Items) {
		return fmt.Errorf("subs: %d traductions pour %d sous-titres", len(translated), len(s.doc.Items))
	}
	for i, it := range s.doc.Items {
		it.Lines = textToLines(translated[i])
	}
	return nil
}

// SaveSRT écrit le document au format SRT (path doit finir par .srt).
func (s *Subtitle) SaveSRT(path string) error {
	if err := s.doc.Write(path); err != nil {
		return fmt.Errorf("subs: écriture %q: %w", path, err)
	}
	return nil
}

// LimitToDuration ne conserve que les cues commençant avant d (mode Test 20 s).
func (s *Subtitle) LimitToDuration(d time.Duration) {
	kept := s.doc.Items[:0]
	for _, it := range s.doc.Items {
		if it.StartAt < d {
			kept = append(kept, it)
		}
	}
	s.doc.Items = kept
}

// itemText reconstruit le texte d'une cue à partir des LineItem (runs
// inline concaténés, lignes jointes par une espace), puis l'aplatit.
func itemText(it *astisub.Item) string {
	var sb strings.Builder
	for li, ln := range it.Lines {
		if li > 0 {
			sb.WriteString(" ")
		}
		for _, item := range ln.Items {
			sb.WriteString(item.Text)
		}
	}
	return flatten(sb.String())
}

func flatten(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
}

func textToLines(text string) []astisub.Line {
	parts := strings.Split(text, "\n")
	lines := make([]astisub.Line, 0, len(parts))
	for _, p := range parts {
		lines = append(lines, astisub.Line{Items: []astisub.LineItem{{Text: p}}})
	}
	return lines
}
