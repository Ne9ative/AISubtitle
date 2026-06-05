# AI Subtitle Pro

Application Windows de bureau (**Go + Wails**) qui traduit les sous-titres d'une
vidéo en **français**, soit en **local sur GPU** (llama.cpp, build CUDA), soit via
l'**API Gemini**.

> Réécriture en Go d'un ancien outil Python/PyQt, avec une interface repensée,
> une traduction **par lots avec contexte** (dialogues cohérents) et une
> distribution en un seul exécutable.

## Fonctionnement

1. Glisser-déposer une vidéo (`.mkv`, `.mp4`, `.avi`) — ou parcourir.
2. Choisir la **piste de sous-titres**, la **langue source** et le **moteur**.
3. Traduction par lots avec contexte → réassemblage.
4. Remux d'une nouvelle piste **FR par défaut** → `{nom}_PRO_FR.mkv`
   (ou `{nom}_TEST_20s.mkv` en mode Test).

## Distribution

- **Un seul `.exe`** + un dossier **`models/`** (vos fichiers `.gguf`) à côté.
- `mkvmerge` / `mkvextract` (MKVToolNix) sont **embarqués** puis auto-extraits dans
  `%LOCALAPPDATA%\AISubtitlePro\bin\`.
- `llama-server` (CUDA) est **téléchargé automatiquement** au 1er lancement
  (~620 Mo) dans ce même cache. Une connexion internet est requise cette
  première fois ; ensuite tout est local.

## Développement

**Prérequis :** Go 1.23+, Node 18+, et le CLI Wails :
```
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

**Préparer les binaires embarqués (avant le build) :** copier `mkvmerge.exe` et
`mkvextract.exe` (depuis https://mkvtoolnix.download/) dans
`internal/runtime/binaries/`. Ils ne sont pas versionnés (voir `.gitignore`).

**Lancer / builder :**
```
wails dev      # développement (hot reload)
wails build    # produit build/bin/AISubtitlePro.exe
```

**Tests :**
```
go test ./...
# tests d'intégration MKV (optionnels), pointer vers une install MKVToolNix :
AISUBTITLE_MKVTOOLNIX_DIR="C:\chemin\mkvtoolnix" go test ./internal/mkv/
```

## Architecture

| Paquet | Rôle |
|---|---|
| `app.go` / `main.go` | bindings Wails + fenêtre |
| `internal/config` | chargement/sauvegarde des réglages |
| `internal/runtime` | embarquage mkvtoolnix + téléchargement llama-server CUDA |
| `internal/mkv` | scan / extraction / remux (MKVToolNix) |
| `internal/subs` | lecture/écriture SRT·ASS + découpage en lots+contexte |
| `internal/engine` | interface `Translator` + `Local` (llama-server) + `Gemini` |
| `internal/pipeline` | orchestration extraction → traduction → remux |
| `internal/winproc` | masquage des fenêtres console (Windows) |

## Licences des composants externes

- **MKVToolNix** (mkvmerge, mkvextract) — GPL-2.0
- **llama.cpp** (llama-server) — MIT
