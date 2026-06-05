# AI Subtitle Pro — Réécriture en Go / Wails — Document de conception

**Date :** 2026-06-05
**Statut :** Validé (en attente de relecture finale)
**Dépôt :** https://github.com/Ne9ative/AISubtitle

---

## 1. Contexte & objectif

Il existe une application Windows fonctionnelle (`trad.py`, PyQt6 + `llama-cpp-python`) qui :

1. reçoit une vidéo par glisser-déposer (`.mkv`, `.mp4`, `.avi`) ;
2. scanne les pistes de sous-titres via `mkvmerge -J` ;
3. extrait une piste via `mkvextract` ;
4. traduit chaque ligne en français, soit **en local** (modèle GGUF Gemma sur GPU via llama.cpp), soit **en ligne** (API Gemini) ;
5. ré-injecte le SRT traduit dans la vidéo via `mkvmerge` (nouvelle piste FR par défaut).

**Objectif :** réécrire cette application **en Go**, distribuée en **un seul `.exe`** (avec un dossier `models/` visible à côté), dotée d'une **interface nettement plus soignée**, tout en **améliorant la qualité de traduction**.

Le « cœur » coûteux (inférence LLM, démultiplexage Matroska) reste assuré par des composants externes éprouvés (llama.cpp, MKVToolNix). Go apporte : une distribution propre en binaire unique (fini l'enfer `venv` / `pip` / wheel CUDA) et une interface moderne via Wails.

---

## 2. Périmètre

### Inclus
- Glisser-déposer vidéo + scan des pistes de sous-titres.
- Deux moteurs de traduction : **Local** (GGUF/GPU) et **Gemini** (API).
- Traduction **par lots avec contexte** (sortie structurée JSON).
- Remux du SRT traduit dans un `.mkv` de sortie (piste FR par défaut).
- Mode **Test (20 s)**, **bouton Annuler**, réglages persistés.
- Interface **sombre moderne épurée**.
- Distribution **mono-`.exe`** : binaires techniques embarqués et auto-extraits.

### Exclus (hors périmètre, voir §12)
- OCR des sous-titres **image** (PGS / VobSub) → détectés et signalés, pas traduits.
- Traitement par lot de plusieurs fichiers à la fois.
- Préservation du style avancé ASS/SSA (sortie en SRT).
- Plateformes autres que Windows x64.

---

## 3. Choix techniques validés

| Sujet | Décision | Justification courte |
|---|---|---|
| Langage | **Go** | Binaire unique, pas de runtime Python à installer |
| Interface | **Wails v2** (Go + frontend Svelte) | Fenêtre native (WebView2, déjà sur Win10/11), plafond esthétique élevé |
| Moteurs | **Local + Gemini** | Conserver la polyvalence actuelle |
| Inférence locale | **llama-server** (llama.cpp) en sous-processus, **backend Vulkan** | GPU conservé sans compilation CGo ; Vulkan = exe léger + tout GPU récent |
| Empaquetage | **Tout embarqué** (`go:embed`) | 100 % hors-ligne et autonome |
| Outils Matroska | **MKVToolNix** (mkvmerge/mkvextract) | Éprouvé, robuste, déjà open source (GPL-2.0) |
| Traduction | **Par lots + contexte**, JSON structuré | Qualité + vitesse |
| Sous-titres | lib `go-astisub` (SRT/ASS) | Lecture/écriture fiable |

---

## 4. Architecture

### 4.1 Vue d'ensemble

```
┌─────────────────────────── AISubtitlePro.exe ───────────────────────────┐
│  Frontend Svelte (UI sombre)  ◀── events (progress/log/done) ──┐         │
│        │  appels liés (bindings Wails)                          │         │
│        ▼                                                        │         │
│  app (orchestration, goroutine + context.Context)  ────────────┘         │
│    │        │            │              │            │                    │
│    ▼        ▼            ▼              ▼            ▼                     │
│  config   mkv         subs          engine       runtime                 │
│         (mkvmerge/  (go-astisub,  (Translator:  (go:embed +              │
│          mkvextract) lots+contexte) local|gemini) extraction cache)      │
└──────────────────────────────────────────────────────────────────────────┘
        │                                   │
        ▼                                   ▼
  %LOCALAPPDATA%\AISubtitlePro\bin\   ←  llama-server + mkvtoolnix extraits
```

### 4.2 Arborescence disque (ce que voit l'utilisateur)

```
AISubtitlePro.exe        ← binaire unique (Vulkan + MKVToolNix embarqués)
models/                  ← fichiers .gguf (visibles, fournis par l'utilisateur)
config.json              ← réglages (créé au 1er enregistrement, jamais versionné)
```

Cache caché, créé au premier lancement (hors du dossier de l'app) :

```
%LOCALAPPDATA%\AISubtitlePro\bin\<hash-version>\
    llama-server.exe, *.dll (Vulkan)
    mkvmerge.exe, mkvextract.exe
```

### 4.3 Paquets backend (Go)

Chaque paquet a une responsabilité unique, une interface claire, et est testable isolément.

- **`runtime`** — embarque les binaires (`go:embed`), les extrait dans le cache au 1er lancement (extraction conditionnée par un hash de version), expose les chemins : `LlamaServerPath()`, `MkvmergePath()`, `MkvextractPath()`.
- **`mkv`** — enveloppe les appels MKVToolNix :
  - `ScanSubtitleTracks(video) ([]Track, error)` (via `mkvmerge -J`)
  - `ExtractTrack(video, id, out) error`
  - `Mux(video, sub, out, lang, trackName) error`
  - Tous les sous-processus sont lancés **sans fenêtre console** (`CREATE_NO_WINDOW`).
- **`subs`** — lecture/écriture via `go-astisub` ; découpage en lots + contexte ; réassemblage en conservant le minutage.
- **`engine`** — interface `Translator` + implémentations `local` et `gemini` (voir §6).
- **`config`** — `Load()/Save()` du `config.json` (à côté de l'exe). La clé API n'est jamais journalisée.
- **`app`** — structure liée à Wails ; expose les méthodes au frontend, orchestre le pipeline dans une goroutine, émet les events, gère l'annulation.

### 4.4 Frontend (Svelte)

Fenêtre unique, sections de haut en bas :
1. **Zone de dépôt** (drag & drop + clic pour parcourir) — affiche le nom du fichier.
2. **Moteur** (Local / Gemini) — affiche conditionnellement le champ clé API ou la liste des modèles `.gguf`.
3. **Langue source** + **Piste de sous-titres** (listes déroulantes).
4. **Actions** : `Test (20 s)`, `Démarrer`, `Annuler`.
5. **Barre de progression** animée (%, compteur de lignes, ETA).
6. **Journal** repliable, monospace, coloré par niveau.

Style : fond sombre, **une** couleur d'accent, cartes à coins arrondis, ombres légères, transitions douces, police type Inter.

---

## 5. Flux de données détaillé

1. **Dépôt vidéo** → le frontend envoie le chemin à `app.ScanTracks`.
2. `mkv.ScanSubtitleTracks` → renvoie `[{id, langue, codec, nom, isImageBased}]` → la liste des pistes se remplit (les pistes image sont marquées « non traduisible »).
3. L'utilisateur choisit moteur / modèle (ou clé API) / langue / piste, puis clique **Démarrer** (ou **Test**).
4. `app.StartTranslation` (goroutine + `context.Context`) :
   a. `mkv.ExtractTrack` → SRT temporaire.
   b. `subs` lit le SRT, le découpe en **lots de N lignes** + **K lignes de contexte** précédentes.
   c. Pour chaque lot : `engine.Translate(ctx, lot, langueSource)` → tableau JSON de traductions ; validation du nombre de lignes (repli si écart).
   d. `subs` réinjecte les traductions et exporte le SRT final.
   e. `mkv.Mux` → `{nom}_PRO_FR.mkv` (piste FR, `--default-track`).
   f. Nettoyage des fichiers temporaires.
5. Tout au long : events `progress` / `log` ; à la fin : `done` (chemin de sortie) ou `error`.
6. **Annuler** → annule le `context`, tue `llama-server`, nettoie.

---

## 6. Stratégie de traduction par lots + contexte

**Problème actuel :** chaque ligne est traduite isolément → perte de contexte (pronoms, continuité), lenteur (un appel GPU par ligne), et « bavures » du modèle nettoyées à coups de regex.

**Nouvelle approche :**
- Regrouper N lignes (par défaut **10–15**) en un seul appel, en fournissant **K lignes précédentes** (par défaut **2–3**) en **contexte non traduit** (lecture seule).
- Demander une **sortie strictement JSON** : `{"translations": ["…", "…", …]}`, alignée 1:1 sur les lignes d'entrée.
- **Validation** : si le nombre de traductions ≠ nombre d'entrées → 1 nouvel essai, puis repli en sous-lots / ligne par ligne pour ce lot uniquement.
- Avantages : cohérence des dialogues, moins d'appels GPU (plus rapide), suppression du nettoyage regex.

### Interface `Translator`

```go
type Batch struct {
    Context []string // lignes précédentes, non traduites (contexte)
    Lines   []string // lignes à traduire
}

type Translator interface {
    // Renvoie len(batch.Lines) traductions, dans l'ordre.
    Translate(ctx context.Context, b Batch, srcLang string) ([]string, error)
    Close() error
}
```

- **`local`** : démarre `llama-server -m <modèle> -ngl 99 --host 127.0.0.1 --port <libre> --ctx-size 4096` (caché) ; attend `/health` = 200 ; envoie des requêtes `/v1/chat/completions` (compatible OpenAI) avec consigne JSON. Le serveur reste vivant toute la durée du job ; rechargé uniquement si le modèle change ; arrêté à la fin/à l'annulation.
- **`gemini`** : `POST …/v1beta/models/<modèle>:generateContent?key=…` avec `responseMimeType: application/json` (approche déjà éprouvée dans l'ancien code). Gestion des 429 (backoff) et des erreurs de quota/clé avec message clair.

---

## 7. Gestion des erreurs

| Situation | Comportement |
|---|---|
| Aucun `.gguf` dans `models/` | Message clair, mode Local désactivé |
| `llama-server` ne démarre pas (GPU/Vulkan absent, modèle corrompu) | Capture stderr, message explicite, suggestion (maj pilotes / autre modèle) |
| Pilote Vulkan absent | Message dédié : « mettez à jour vos pilotes GPU » |
| Piste de sous-titres **image** (PGS/VobSub) | Détectée au scan, marquée non traduisible, sélection bloquée avec explication |
| Erreur API Gemini (quota, clé invalide, réseau) | Message clair, pas de crash, possibilité de réessayer |
| Réponse JSON du modèle mal formée / mauvais compte | 1 réessai, puis repli sous-lots / ligne par ligne |
| Annulation utilisateur | `context` annulé, `llama-server` tué, temporaires nettoyés |
| WebView2 absent (rare, Win10 ancien) | Wails propose son installation |

---

## 8. Tests

- **Unitaires (logique pure, TDD)** :
  - `subs` : découpage en lots + contexte, réassemblage, conservation du minutage, gestion `\N` / multi-lignes.
  - parsing de la réponse modèle : JSON valide, **JSON malformé**, compte incorrect, texte parasite.
  - `config` : load/save, valeurs par défaut, fichier absent/corrompu.
  - construction des prompts.
- **Pipeline** : `Translator` **mocké** → vérifier extraction → lots → réassemblage → mux (mux mocké).
- **Intégration** (optionnel) : petit `.mkv` d'exemple pour `mkv` (scan/extract/mux réels).
- **Manuel / E2E** : un vrai épisode, moteur Local puis Gemini.

---

## 9. Améliorations par rapport à la version actuelle

- Qualité : lots + contexte (vs ligne par ligne) ; fin du nettoyage regex.
- **Bouton Annuler** (absent aujourd'hui).
- Messages d'erreur clairs (pistes image, échec GPU, erreurs API).
- Réglages mémorisés (dernier moteur/modèle/langue).
- Distribution : un `.exe`, fini `install.bat` / `venv` / wheel CUDA.
- Interface modernisée.

---

## 10. Décisions techniques & justifications

- **Wails plutôt que Fyne/Walk** : seule option permettant un rendu vraiment moderne tout en restant une fenêtre native mono-exe.
- **Vulkan plutôt que CUDA** : exe ~100–150 Mo (vs ~400–700 Mo en CUDA, à cause des DLL cuBLAS), compatible NVIDIA/AMD/Intel, perfs très proches. La `vulkan-1.dll` est fournie par les pilotes GPU (non embarquée).
- **llama-server (sous-processus) plutôt que CGo** : conserve le GPU sans cauchemar de compilation ; isolation des crashs ; mises à jour de llama.cpp triviales.
- **MKVToolNix plutôt que ffmpeg/Go pur** : c'est ce qui fonctionne déjà, robuste ; réécrire le muxing Matroska en Go serait risqué pour un gain nul.
- **Tout embarqué plutôt que téléchargement au 1er lancement** : autonomie totale, au prix d'un exe plus lourd (assumé).

---

## 11. Risques & inconnues

- **Taille de l'exe** (~100–200 Mo) et **délai d'extraction** au 1er lancement (quelques secondes) — acceptés.
- **Dépendance Vulkan** : nécessite des pilotes GPU à jour.
- **VRAM** : les gros modèles (26B) peuvent dépasser la VRAM → message clair, suggestion d'un modèle plus petit.
- **Sous-titres image** non gérés (hors périmètre).
- **Alignement JSON** : risque que le modèle local renvoie un mauvais compte de lignes → mitigé par validation + repli.
- **Compatibilité API Gemini** : noms de modèles susceptibles d'évoluer côté Google.

---

## 12. Hors périmètre / évolutions futures

- OCR des sous-titres image (PGS/VobSub).
- Traitement par lot multi-fichiers / file d'attente.
- Préservation du style ASS/SSA avancé.
- Portage macOS/Linux (Wails et Vulkan le permettraient).
- Choix de la langue **cible** (pas seulement le français).
