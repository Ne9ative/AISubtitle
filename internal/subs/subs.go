package subs

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	astisub "github.com/asticode/go-astisub"
)

// Subtitle encapsule un document de sous-titres pour la traduction.
type Subtitle struct {
	doc    *astisub.Subtitles
	format string // "srt", "ass" ou "vtt"
}

// Open lit un fichier de sous-titres en détectant le format au CONTENU
// (l'extension du fichier extrait par mkvextract n'est pas fiable : il écrit
// le codec natif — SRT, ASS/SSA ou VTT — quel que soit le nom du fichier).
func Open(path string) (*Subtitle, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("subs: lecture %q: %w", path, err)
	}
	doc, format, err := parse(data)
	if err != nil {
		return nil, fmt.Errorf("subs: analyse %q: %w", path, err)
	}
	return &Subtitle{doc: doc, format: format}, nil
}

func parse(data []byte) (*astisub.Subtitles, string, error) {
	trimmed := bytes.TrimLeft(data, "\xef\xbb\xbf \t\r\n") // BOM + espaces de tête
	head := strings.ToLower(string(trimmed[:min(len(trimmed), 300)]))
	switch {
	case strings.Contains(head, "[script info]"), strings.Contains(head, "[v4"):
		doc, err := astisub.ReadFromSSA(bytes.NewReader(data))
		return doc, "ass", err
	case strings.HasPrefix(head, "webvtt"):
		doc, err := astisub.ReadFromWebVTT(bytes.NewReader(data))
		return doc, "vtt", err
	default:
		doc, err := astisub.ReadFromSRT(bytes.NewReader(data))
		return doc, "srt", err
	}
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
		setItemText(it, translated[i])
	}
	return nil
}

// setItemText remplace le texte affiché en CONSERVANT le 1er LineItem, qui
// porte les balises de positionnement ASS ({\pos}, {\an}…) dans son SSAEffect.
// Sans ça, l'ASS perdrait son placement à l'écran (lignes empilées en vrac).
func setItemText(it *astisub.Item, text string) {
	text = strings.ReplaceAll(text, "\n", " ")
	if len(it.Lines) > 0 && len(it.Lines[0].Items) > 0 {
		first := it.Lines[0].Items[0]
		first.Text = text
		it.Lines = it.Lines[:1]
		it.Lines[0].Items = []astisub.LineItem{first}
		return
	}
	it.Lines = []astisub.Line{{Items: []astisub.LineItem{{Text: text}}}}
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
	if err := s.doc.Write(path); err != nil {
		return fmt.Errorf("subs: écriture %q: %w", path, err)
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

// assTagRe capture les balises d'override ASS, ex. {\i1}, {\an8}, {\pos(...)}.
var assTagRe = regexp.MustCompile(`\{[^}]*\}`)

func flatten(s string) string {
	s = assTagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

