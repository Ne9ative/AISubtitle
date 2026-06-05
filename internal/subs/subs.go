package subs

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	astisub "github.com/asticode/go-astisub"
)

// Subtitle encapsule un document de sous-titres pour la traduction.
//   - ASS  : géré en BRUT (assRaw) pour préserver en-tête et styles à l'identique.
//   - SRT/VTT : géré via go-astisub.
type Subtitle struct {
	doc    *astisub.Subtitles
	ass    *assRaw
	format string // "srt", "ass" ou "vtt"
}

// Open lit un fichier de sous-titres ; le format est détecté au CONTENU
// (l'extension du fichier extrait par mkvextract n'est pas fiable).
func Open(path string) (*Subtitle, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("subs: lecture %q: %w", path, err)
	}
	format := detectFormat(data)
	if format == "ass" {
		return &Subtitle{format: "ass", ass: parseASS(data)}, nil
	}
	doc, err := parseNonASS(data, format)
	if err != nil {
		return nil, fmt.Errorf("subs: analyse %q: %w", path, err)
	}
	return &Subtitle{format: format, doc: doc}, nil
}

func detectFormat(data []byte) string {
	trimmed := bytes.TrimLeft(data, "\xef\xbb\xbf \t\r\n")
	head := strings.ToLower(string(trimmed[:min(len(trimmed), 300)]))
	switch {
	case strings.Contains(head, "[script info]"), strings.Contains(head, "[v4"):
		return "ass"
	case strings.HasPrefix(head, "webvtt"):
		return "vtt"
	default:
		return "srt"
	}
}

func parseNonASS(data []byte, format string) (*astisub.Subtitles, error) {
	if format == "vtt" {
		return astisub.ReadFromWebVTT(bytes.NewReader(data))
	}
	return astisub.ReadFromSRT(bytes.NewReader(data))
}

// Len renvoie le nombre de répliques actives.
func (s *Subtitle) Len() int {
	if s.ass != nil {
		return len(s.ass.activeIdx())
	}
	return len(s.doc.Items)
}

// Texts renvoie le texte source de chaque réplique, dans l'ordre.
func (s *Subtitle) Texts() []string {
	if s.ass != nil {
		return s.ass.activeTexts()
	}
	out := make([]string, len(s.doc.Items))
	for i, it := range s.doc.Items {
		out[i] = itemText(it)
	}
	return out
}

// SetTexts remplace le texte de chaque réplique par sa traduction.
func (s *Subtitle) SetTexts(translated []string) error {
	if s.ass != nil {
		return s.ass.setTrans(translated)
	}
	if len(translated) != len(s.doc.Items) {
		return fmt.Errorf("subs: %d traductions pour %d sous-titres", len(translated), len(s.doc.Items))
	}
	for i, it := range s.doc.Items {
		setItemText(it, translated[i])
	}
	return nil
}

// LimitToDuration ne conserve que les répliques commençant avant d (mode Test 20 s).
func (s *Subtitle) LimitToDuration(d time.Duration) {
	if s.ass != nil {
		s.ass.limit(d)
		return
	}
	kept := s.doc.Items[:0]
	for _, it := range s.doc.Items {
		if it.StartAt < d {
			kept = append(kept, it)
		}
	}
	s.doc.Items = kept
}

// Ext renvoie l'extension du format source (.ass / .vtt / .srt).
func (s *Subtitle) Ext() string {
	switch s.format {
	case "ass":
		return ".ass"
	case "vtt":
		return ".vtt"
	default:
		return ".srt"
	}
}

// Save écrit le document dans son format d'origine (path doit porter Ext()).
func (s *Subtitle) Save(path string) error {
	if s.ass != nil {
		if err := os.WriteFile(path, []byte(s.ass.render()), 0o644); err != nil {
			return fmt.Errorf("subs: écriture %q: %w", path, err)
		}
		return nil
	}
	if err := s.doc.Write(path); err != nil {
		return fmt.Errorf("subs: écriture %q: %w", path, err)
	}
	return nil
}

// SaveSRT est conservé pour compatibilité (SRT/VTT) ; délègue à Save.
func (s *Subtitle) SaveSRT(path string) error { return s.Save(path) }

// --- helpers go-astisub (SRT/VTT) ---

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
	return strings.TrimSpace(strings.ReplaceAll(sb.String(), "\n", " "))
}

func setItemText(it *astisub.Item, text string) {
	it.Lines = []astisub.Line{{Items: []astisub.LineItem{{Text: text}}}}
}
