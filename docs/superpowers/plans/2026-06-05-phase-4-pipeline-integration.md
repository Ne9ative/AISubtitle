# Plan 4 — Pipeline & Intégration — Implementation Plan

> **Statut : partiel.** Le `pipeline` est IMPLÉMENTÉ et testé. L'embarquage des binaires (`runtime`) et le branchement Wails (`app`) attendent les binaires (point de décision utilisateur).

**Goal :** Assembler `subs` + `mkv` + `engine` en un pipeline complet, embarquer les binaires dans l'exe, et brancher le tout à l'UI via Wails (events, annulation).

---

## Partie A — Pipeline (FAIT)

**Fichiers :**
```
internal/subs/subs.go     + LimitToDuration(d) pour le mode Test 20 s
internal/pipeline/pipeline.go      Run, OutputPath, repli ligne-par-ligne
internal/pipeline/pipeline_test.go mocks Extractor/Translator/Reporter
```

- **`pipeline.Run(ctx, ex, tr, opts, rep)`** : extrait la piste → (mode Test : tronque à 20 s) → `MakeBatches` → traduit chaque lot → **repli ligne-par-ligne** si un lot échoue (et conserve l'original si une ligne échoue) → `SetTexts` → `SaveSRT` → `Mux`. Émet progression + journal via `Reporter`. Respecte `ctx` (annulation).
- **`OutputPath`** : `{nom}_PRO_FR.mkv` ou `{nom}_TEST_20s.mkv`, sans double suffixe.
- Interfaces `Extractor` (sous-ensemble de `mkv.Tool`) et `Reporter` → testable avec des mocks, sans binaires.

**Tests verts :** chemin nominal (traduction appliquée, progression 2/2, remux appelé), repli ligne-par-ligne, nom de sortie en mode Test, `OutputPath`.

---

## Partie B — runtime (embed) — À FAIRE (nécessite les binaires)

- `go:embed` des binaires dans `assets/bin/` (non versionnés : voir `.gitignore`) :
  - `llama-server.exe` + DLL (build **Vulkan** de llama.cpp)
  - `mkvmerge.exe`, `mkvextract.exe` (MKVToolNix)
- `runtime.EnsureBinaries()` : extrait vers `%LOCALAPPDATA%\AISubtitlePro\bin\<hash>\` si absent, renvoie les chemins (`LlamaServerPath`, `MkvmergePath`, `MkvextractPath`).
- Réutilise `CacheDir`/`EnsureCacheDir` (Plan 1).

## Partie C — app (Wails) — À FAIRE

- Méthodes liées : `ScanTracks(path)`, `ListModels()`, `StartTranslation(opts)` (goroutine + `context.Context`), `Cancel()`, `GetConfig`/`SaveConfig`.
- `App` implémente `pipeline.Reporter` en émettant les events Wails `progress` / `log` ; events `done` / `error` en fin de job.
- Câblage : `runtime.EnsureBinaries()` → `mkv.Tool{...}` + `engine.NewLocal(...)`/`NewGemini(...)` → `pipeline.Run(...)`.

## Intégration finale

- Test réel de `engine.Local.Start` (lancement llama-server Vulkan) une fois le binaire en place.
- Test bout-en-bout sur une vraie vidéo (fournie par l'utilisateur) au Plan 5.
