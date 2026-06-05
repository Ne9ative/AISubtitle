package engine

import (
	"encoding/json"
	"fmt"
	"strings"
)

// buildPrompt construit l'instruction de traduction vers le français,
// demandant une sortie JSON stricte alignée 1:1 sur les lignes d'entrée.
func buildPrompt(lines, ctxLines []string, srcLang, tgtLang string) string {
	linesJSON, _ := json.Marshal(lines)
	var sb strings.Builder
	sb.WriteString("Tu es un traducteur professionnel de sous-titres. ")
	sb.WriteString("Traduis CHAQUE élément du tableau JSON ci-dessous depuis le ")
	sb.WriteString(srcLang)
	sb.WriteString(" vers le ")
	sb.WriteString(tgtLang)
	sb.WriteString(", en gardant le ton, le registre et la concision d'un sous-titre.\n")
	sb.WriteString("Traduis tout le texte, mais NE touche pas à la numérotation : si une ligne commence par un chiffre (ex. \"1.\"), garde ce chiffre tel quel, et n'ajoute jamais de numéro qui n'existe pas. ")
	sb.WriteString("Ne fusionne pas, ne supprime pas, n'invente rien.\n")
	if len(ctxLines) > 0 {
		ctxJSON, _ := json.Marshal(ctxLines)
		sb.WriteString("Contexte précédent (NE PAS traduire, seulement pour la cohérence) : ")
		sb.Write(ctxJSON)
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("Réponds UNIQUEMENT par un objet JSON {\"translations\": [...]} contenant EXACTEMENT %d traductions, dans le même ordre que l'entrée.\n", len(lines)))
	sb.WriteString("À traduire : ")
	sb.Write(linesJSON)
	return sb.String()
}

// parseTranslations extrait le tableau "translations" d'une réponse modèle,
// même entourée de texte ou de balises markdown, et valide le nombre d'éléments.
func parseTranslations(text string, expected int) ([]string, error) {
	jsonStr := extractJSONObject(text)
	if jsonStr == "" {
		return nil, fmt.Errorf("engine: aucune réponse JSON trouvée (%q)", truncate(text, 160))
	}
	var payload struct {
		Translations []string `json:"translations"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &payload); err != nil {
		return nil, fmt.Errorf("engine: JSON de traduction invalide: %w", err)
	}
	if len(payload.Translations) != expected {
		return nil, fmt.Errorf("engine: %d traductions reçues, %d attendues", len(payload.Translations), expected)
	}
	return payload.Translations, nil
}

// extractJSONObject renvoie la sous-chaîne du 1er '{' au dernier '}'.
func extractJSONObject(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < 0 || end < start {
		return ""
	}
	return text[start : end+1]
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
