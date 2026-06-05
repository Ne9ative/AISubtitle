package subs

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	assTagRe  = regexp.MustCompile(`\{[^}]*\}`)       // une balise d'override ASS
	leadTagRe = regexp.MustCompile(`^(?:\{[^}]*\})*`) // suite de balises en tête de ligne
)

// assRaw gère l'ASS par remplacement de texte BRUT : on conserve le fichier
// d'origine tel quel (en-tête [Script Info], [V4+ Styles], structure des lignes)
// et on ne remplace que le texte de chaque réplique. C'est bien plus fidèle
// qu'une réécriture via go-astisub, qui normalise les styles et altère le rendu.
type assRaw struct {
	lines  []string        // toutes les lignes du fichier (verbatim)
	dlgIdx []int           // indices des lignes "Dialogue:" dans lines
	starts []time.Duration // Start de chaque réplique (pour le mode Test)
	leads  []string        // balises de tête ({\pos}{\an7}…) de chaque réplique
	src    []string        // texte source (sans balises) de chaque réplique
	keep   []bool          // réplique active (LimitToDuration)
	trans  []string        // traductions (remplies par setTrans)
}

func parseASS(data []byte) *assRaw {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:] // retirer un BOM UTF-8 éventuel
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	a := &assRaw{}
	for _, line := range strings.Split(text, "\n") {
		idx := len(a.lines)
		a.lines = append(a.lines, line)
		if !strings.HasPrefix(line, "Dialogue:") {
			continue
		}
		parts := strings.SplitN(line, ",", 10)
		if len(parts) < 10 {
			continue // ligne Dialogue malformée : laissée verbatim
		}
		lead, plain := splitLeadTags(parts[9])
		a.dlgIdx = append(a.dlgIdx, idx)
		a.starts = append(a.starts, parseASSTime(parts[1]))
		a.leads = append(a.leads, lead)
		a.src = append(a.src, plain)
		a.keep = append(a.keep, true)
	}
	a.trans = make([]string, len(a.dlgIdx))
	return a
}

// splitLeadTags sépare les balises de tête (positionnement/style) du texte
// visible, qu'on nettoie de toute autre balise et des sauts de ligne ASS (\N).
func splitLeadTags(text string) (lead, plain string) {
	lead = leadTagRe.FindString(text)
	plain = assTagRe.ReplaceAllString(text[len(lead):], "")
	plain = strings.NewReplacer(`\N`, " ", `\n`, " ", `\h`, " ").Replace(plain)
	return lead, strings.TrimSpace(plain)
}

func parseASSTime(s string) time.Duration {
	var h, m, sec, cs int
	if _, err := fmt.Sscanf(strings.TrimSpace(s), "%d:%d:%d.%d", &h, &m, &sec, &cs); err != nil {
		return 0
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute +
		time.Duration(sec)*time.Second + time.Duration(cs)*10*time.Millisecond
}

func (a *assRaw) activeIdx() []int {
	var out []int
	for i, k := range a.keep {
		if k {
			out = append(out, i)
		}
	}
	return out
}

func (a *assRaw) activeTexts() []string {
	idx := a.activeIdx()
	out := make([]string, len(idx))
	for j, i := range idx {
		out[j] = a.src[i]
	}
	return out
}

func (a *assRaw) setTrans(tr []string) error {
	idx := a.activeIdx()
	if len(tr) != len(idx) {
		return fmt.Errorf("subs: %d traductions pour %d répliques", len(tr), len(idx))
	}
	for j, i := range idx {
		a.trans[i] = tr[j]
	}
	return nil
}

func (a *assRaw) limit(d time.Duration) {
	for i := range a.keep {
		a.keep[i] = a.starts[i] < d
	}
}

// render reconstruit l'ASS : lignes d'origine verbatim, sauf les répliques
// (texte remplacé par la traduction, balises de tête conservées) ; les répliques
// inactives (mode Test) sont retirées.
func (a *assRaw) render() string {
	pos := make(map[int]int, len(a.dlgIdx))
	for j, idx := range a.dlgIdx {
		pos[idx] = j
	}
	out := make([]string, 0, len(a.lines))
	for idx, line := range a.lines {
		j, isDlg := pos[idx]
		if !isDlg {
			out = append(out, line)
			continue
		}
		if !a.keep[j] {
			continue
		}
		if a.trans[j] == "" {
			out = append(out, line) // pas de traduction : ligne d'origine
			continue
		}
		parts := strings.SplitN(line, ",", 10)
		parts[9] = a.leads[j] + a.trans[j]
		out = append(out, strings.Join(parts, ","))
	}
	return strings.Join(out, "\n") + "\n"
}
