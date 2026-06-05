# Plan 2 — Sous-titres & MKV — Implementation Plan

> **Statut : IMPLÉMENTÉ** — tous les tests verts, y compris l'intégration contre les vrais binaires MKVToolNix.

**Goal :** Lire/écrire des sous-titres et les découper en lots+contexte (`subs`), et piloter MKVToolNix pour scanner/extraire/remuxer des pistes (`mkv`).

**Architecture :** Deux paquets `internal/*` à responsabilité unique, testables isolément. `subs` s'appuie sur `github.com/asticode/go-astisub` (SRT/ASS/VTT). `mkv` exécute `mkvmerge`/`mkvextract` en sous-processus cachés, avec chemins de binaires injectables.

**Tech Stack :** Go, go-astisub v0.40.0, MKVToolNix (externe).

---

## Fichiers créés

```
internal/subs/
  subs.go        Subtitle: Open / Len / Texts / SetTexts / SaveSRT
  batch.go       Batch + MakeBatches(texts, batchSize, contextSize)
  subs_test.go   parsing SRT réel, roundtrip set+save, mismatch
  batch_test.go  couverture, contexte, tailles limites
internal/mkv/
  mkv.go          Track, Tool{Mkvmerge,Mkvextract}, parseTracksJSON,
                  ScanSubtitleTracks / ExtractTrack / Mux, run()
  proc_windows.go hideWindow → CREATE_NO_WINDOW (0x08000000)
  proc_other.go   hideWindow no-op (//go:build !windows)
  mkv_test.go     parsing JSON (+ PGS image, langue 'und'), intégration
```

## Tâches (réalisées)

- **`subs.Subtitle`** : `Open` (format par extension), `Texts()` reconstruit chaque cue depuis `LineItem.Text` (lignes jointes par espace, retours à la ligne aplatis), `SetTexts()` remplace le texte (valide le nombre), `SaveSRT()` écrit en SRT en conservant index/minutage.
- **`subs.MakeBatches`** : découpe en lots de `batchSize`, chacun portant jusqu'à `contextSize` lignes précédentes (contexte non traduit). Garantit : couverture exacte de toutes les lignes une seule fois.
- **`mkv.parseTracksJSON`** : parse `mkvmerge -J`, ne garde que `type == "subtitles"`, déduit `IsImageBased` du `codec_id` (PGS/VobSub/DVBSUB/TEXTST), défaut langue `und`.
- **`mkv.Tool`** : `ScanSubtitleTracks` / `ExtractTrack` / `Mux` (piste FR `--default-track`).
- **`mkv.run`** : sous-processus caché ; tolère le **code de sortie 1** de MKVToolNix (avertissements = succès), n'échoue que sur `>= 2` ; capture stderr pour des messages clairs.

## Décisions

- **go-astisub** plutôt qu'un parseur maison : gère SRT **et** ASS/SSA/VTT, ce qui couvre les pistes texte réelles des MKV.
- Texte reconstruit depuis `LineItem.Text` (et non `Item.String()`) pour ne pas dépendre du séparateur interne de la lib.
- **Chemins de binaires injectables** (`Tool{Mkvmerge, Mkvextract}`) : permet à `runtime` (Plan 4) de fournir les binaires auto-extraits, et aux tests de pointer vers une install locale.
- **Test d'intégration** activé par la variable d'env `AISUBTITLE_MKVTOOLNIX_DIR` (sinon ignoré) : crée un MKV depuis un SRT, le scanne et l'extrait — round-trip réel sans avoir besoin d'une vidéo.

## Tests (verts)

- `subs` : ouverture SRT, `Texts()` attendu, mismatch de comptage, roundtrip set+save+relecture.
- `batch` : vide, couverture exacte, contexte (1 et 2 lignes), gros lot.
- `mkv` : parse JSON (3 pistes, PGS=image, langue manquante→`und`), JSON invalide, sans sous-titres, **intégration scan+extract** contre binaires réels.

## Suite

Le Plan 3 (Moteurs) définira l'interface `Translator` et consommera `subs.Batch`. Le Plan 4 (`runtime` embed) fournira les chemins de binaires à `mkv.Tool`.
