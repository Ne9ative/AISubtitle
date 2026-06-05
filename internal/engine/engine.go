// Package engine traduit des lots de lignes de sous-titres d'une langue source
// vers une langue cible, via un moteur local (llama-server) ou l'API Gemini.
package engine

import "context"

// Translator traduit des lignes de sous-titres d'une langue source vers une cible.
type Translator interface {
	// Translate renvoie EXACTEMENT len(lines) traductions, dans l'ordre.
	// ctxLines = lignes précédentes (déjà affichées) fournies comme contexte
	// non traduit, pour améliorer la cohérence des dialogues.
	Translate(ctx context.Context, lines, ctxLines []string, srcLang, tgtLang string) ([]string, error)
	// Close libère les ressources (arrête llama-server pour le moteur local).
	Close() error
}

// Vérifications à la compilation : les deux moteurs respectent l'interface.
var (
	_ Translator = (*Gemini)(nil)
	_ Translator = (*Local)(nil)
)
