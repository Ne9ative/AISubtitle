# Plan 3 — Moteurs de traduction — Implementation Plan

> **Statut : IMPLÉMENTÉ** — 10 tests verts (prompt, parsing JSON robuste, Gemini & Local via httptest).

**Goal :** Fournir une interface `Translator` commune et deux implémentations : `Gemini` (API) et `Local` (llama-server), avec un parsing JSON tolérant aux réponses imparfaites des modèles.

**Architecture :** Paquet `internal/engine`. L'interface est **découplée de `subs`** : `Translate(ctx, lines, ctxLines, srcLang)` prend des `[]string` bruts (le pipeline du Plan 4 passera `batch.Lines` / `batch.Context`). Logique de prompt et de parsing partagée ; appels HTTP avec URL de base injectable (testée via `httptest`).

**Tech Stack :** Go (net/http), llama-server (OpenAI-compatible), API Gemini.

---

## Fichiers créés

```
internal/engine/
  engine.go       interface Translator + assertions de conformité
  prompt.go       buildPrompt + parseTranslations + extractJSONObject
  gemini.go       Gemini{APIKey,Model,baseURL,client} : /v1beta/models/{m}:generateContent
  local.go        Local{serverBin,modelPath,...} : Start/waitHealthy/Translate/Close
  prompt_test.go  prompt + 6 cas de parseTranslations
  gemini_test.go  httptest (succès + erreur HTTP 429)
  local_test.go   httptest (/v1/chat/completions)
```

## Interface

```go
type Translator interface {
    Translate(ctx context.Context, lines, ctxLines []string, srcLang string) ([]string, error)
    Close() error
}
```

## Tâches (réalisées)

- **`buildPrompt`** : instruction FR stricte, contexte listé séparément (non traduit), demande un JSON `{"translations":[...]}` aligné 1:1, lignes numérotées.
- **`parseTranslations`** : extrait du 1er `{` au dernier `}` (robuste aux fences markdown et au texte autour), valide le **nombre** d'éléments (erreur si écart → le pipeline pourra réessayer / replier).
- **`Gemini.Translate`** : POST `generateContent` avec `responseMimeType: application/json` ; gère les erreurs HTTP (quota, clé) avec message clair.
- **`Local`** : `Start` choisit un port libre, lance `llama-server -m <gguf> --host 127.0.0.1 --port N -ngl 99 --ctx-size 4096` (caché via `winproc`), attend `/health` ; `Translate` POST `/v1/chat/completions` ; `Close` tue le serveur.

## Décisions

- **Interface découplée de `subs`** (prend des `[]string`) : évite tout couplage, simplifie les tests.
- **Parsing tolérant** : les modèles locaux ajoutent souvent des balises/prose → on extrait l'objet JSON et on valide le compte, plutôt que d'exiger une sortie parfaite (fini le nettoyage regex de l'ancienne app).
- **`baseURL` injectable** : permet de tester toute la logique HTTP avec `httptest`, sans vrai serveur.
- **`response_format: json_object`** envoyé à llama-server (supporté par les versions récentes) en plus de la consigne dans le prompt.

## Limite connue / suite

- `Local.Start` (lancement réel de llama-server) sera **validé en intégration au Plan 4**, une fois le binaire `llama-server` (Vulkan) disponible via `runtime`. La logique de traduction HTTP est, elle, déjà testée.
- Le Plan 4 (`app`) assemblera `subs` + `mkv` + `engine` en un pipeline complet (events, annulation), et `runtime` embarquera les binaires.
