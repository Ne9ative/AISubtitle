# Plan 1 — Fondations — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Obtenir une application Wails+Svelte qui se lance (fenêtre sombre) avec une structure de paquets propre, un paquet `config` et un squelette `runtime` testés, `go test ./...` au vert.

**Architecture :** Binaire unique Go via Wails v2 (frontend Svelte rendu par WebView2). Le backend est découpé en paquets `internal/*` à responsabilité unique. Cette phase ne pose que les fondations : pas encore de traduction ni d'embarquage de binaires.

**Tech Stack :** Go 1.26, Wails v2, Svelte + Vite, WebView2 (Windows).

**Module Go :** `github.com/Ne9ative/AISubtitle`
**Racine projet :** `E:\AISubtitle` (déjà : `docs/`, `.gitignore`, `config.example.json`)

---

## Structure de fichiers (cible à la fin du Plan 1)

```
E:\AISubtitle\
  go.mod                      module github.com/Ne9ative/AISubtitle
  go.sum
  main.go                     wails.Run + options fenêtre (sombre)
  app.go                      struct App (ctx, startup, 1 binding témoin)
  wails.json                  config Wails (outputfilename: AISubtitlePro)
  internal/
    config/
      config.go               type Config + Default/Load/Save
      config_test.go
    runtime/
      paths.go                CacheDir/EnsureCacheDir (%LOCALAPPDATA%\AISubtitlePro\bin)
      paths_test.go
  frontend/
    package.json, vite.config.*, src/App.svelte (placeholder sombre), ...
  build/                      (généré par Wails, ignoré par git)
  docs/...                    (déjà présent)
```

---

## Task 0: Installer le CLI Wails (prérequis)

**Files:** aucun (outillage global)

- [ ] **Step 1: Installer le CLI Wails**

Run :
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

- [ ] **Step 2: Vérifier l'installation et l'environnement**

Run :
```bash
wails version
wails doctor
```
Expected : `wails version` affiche `v2.x.x` ; `wails doctor` indique Go/Node/WebView2 « OK » (sous Windows, WebView2 est présent sur Win10/11).

> Si `wails` n'est pas trouvé, ajouter `%USERPROFILE%\go\bin` au PATH (c'est `go env GOPATH`/bin).

---

## Task 1: Scaffolder le projet Wails (template Svelte) à la racine

Wails génère dans un sous-dossier ; on scaffolde dans un dossier temporaire puis on fusionne dans `E:\AISubtitle` sans écraser notre `.gitignore` ni nos `docs/`.

**Files:**
- Create: `main.go`, `app.go`, `wails.json`, `go.mod`, `frontend/**` (générés par Wails)

- [ ] **Step 1: Scaffolder dans un dossier temporaire**

Run :
```bash
mkdir -p E:/_wails_tmp
cd E:/_wails_tmp && wails init -n AISubtitlePro -t svelte
```
Expected : création de `E:/_wails_tmp/AISubtitlePro/` (avec `main.go`, `app.go`, `frontend/`, `wails.json`, `go.mod`).

- [ ] **Step 2: Fusionner dans la racine (sans toucher à notre .gitignore ni docs/)**

Run :
```bash
cp E:/_wails_tmp/AISubtitlePro/main.go      E:/AISubtitle/main.go
cp E:/_wails_tmp/AISubtitlePro/app.go       E:/AISubtitle/app.go
cp E:/_wails_tmp/AISubtitlePro/wails.json   E:/AISubtitle/wails.json
cp E:/_wails_tmp/AISubtitlePro/go.mod       E:/AISubtitle/go.mod
cp -r E:/_wails_tmp/AISubtitlePro/frontend  E:/AISubtitle/frontend
rm -rf E:/_wails_tmp
```
> On NE copie PAS le `.gitignore` généré par Wails (le nôtre couvre déjà `build/`, `frontend/dist/`, `node_modules/`).

- [ ] **Step 3: Renommer le module Go**

Modifier la 1ʳᵉ ligne de `E:/AISubtitle/go.mod` :
```
module github.com/Ne9ative/AISubtitle
```

- [ ] **Step 4: Récupérer les dépendances et builder une 1ʳᵉ fois**

Run :
```bash
cd E:/AISubtitle && go mod tidy && wails build
```
Expected : `build/bin/AISubtitlePro.exe` est créé sans erreur.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: scaffold Wails v2 + Svelte (module github.com/Ne9ative/AISubtitle)"
```

---

## Task 2: Paquet `config` — test d'abord (defaults)

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Écrire le test des valeurs par défaut**

`internal/config/config_test.go` :
```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	c := Default()
	if c.Mode != "Local" || c.SourceLang != "ANGLAIS" || c.BatchSize != 12 || c.ContextSize != 2 {
		t.Fatalf("defaults inattendus: %+v", c)
	}
}
```

- [ ] **Step 2: Lancer le test → échec attendu**

Run : `go test ./internal/config/ -run TestDefault -v`
Expected : FAIL (compilation : `undefined: Default`).

- [ ] **Step 3: Implémentation minimale**

`internal/config/config.go` :
```go
package config

import (
	"encoding/json"
	"os"
)

// Config = réglages utilisateur, persistés à côté de l'exécutable.
type Config struct {
	Mode        string `json:"mode"`         // "Local" ou "Gemini"
	Model       string `json:"model"`        // nom de fichier .gguf ou id de modèle Gemini
	APIKey      string `json:"api_key"`      // clé Gemini (jamais journalisée)
	SourceLang  string `json:"source_lang"`  // ex. "ANGLAIS"
	BatchSize   int    `json:"batch_size"`   // lignes par lot de traduction
	ContextSize int    `json:"context_size"` // lignes précédentes données en contexte
}

// Default renvoie la configuration par défaut.
func Default() Config {
	return Config{
		Mode:        "Local",
		Model:       "",
		APIKey:      "",
		SourceLang:  "ANGLAIS",
		BatchSize:   12,
		ContextSize: 2,
	}
}
```

- [ ] **Step 4: Lancer le test → succès attendu**

Run : `go test ./internal/config/ -run TestDefault -v`
Expected : PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat(config): type Config + valeurs par defaut"
```

---

## Task 3: Paquet `config` — Save/Load (roundtrip, fichier absent, JSON invalide)

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Écrire les tests Save/Load**

Ajouter à `internal/config/config_test.go` :
```go
func TestSaveLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	in := Config{Mode: "Gemini", Model: "gemini-2.0-flash", APIKey: "k", SourceLang: "JAPONAIS", BatchSize: 5, ContextSize: 1}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("roundtrip: got %+v want %+v", out, in)
	}
}

func TestLoadMissingReturnsDefault(t *testing.T) {
	out, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatalf("fichier absent: erreur inattendue %v", err)
	}
	if out != Default() {
		t.Fatalf("attendu défaut, got %+v", out)
	}
}

func TestLoadInvalidReturnsDefaultAndError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{pas du json"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := Load(path)
	if err == nil {
		t.Fatal("attendu une erreur pour JSON invalide")
	}
	if out != Default() {
		t.Fatalf("attendu défaut sur invalide, got %+v", out)
	}
}
```

- [ ] **Step 2: Lancer → échec attendu**

Run : `go test ./internal/config/ -v`
Expected : FAIL (`undefined: Save`, `undefined: Load`).

- [ ] **Step 3: Implémenter Save/Load**

Ajouter à `internal/config/config.go` :
```go
// Load lit la config depuis path. Fichier absent → Default() sans erreur.
// Fichier invalide → Default() AVEC erreur.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return Default(), err
	}
	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), err
	}
	return cfg, nil
}

// Save écrit la config en JSON indenté.
func Save(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
```

- [ ] **Step 4: Lancer → succès attendu**

Run : `go test ./internal/config/ -v`
Expected : PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat(config): Save/Load JSON (defaut si absent, erreur si invalide)"
```

---

## Task 4: Paquet `runtime` — chemins du cache binaires

**Files:**
- Create: `internal/runtime/paths.go`
- Test: `internal/runtime/paths_test.go`

- [ ] **Step 1: Écrire les tests de chemins**

`internal/runtime/paths_test.go` :
```go
package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheDirContainsAppName(t *testing.T) {
	dir, err := CacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dir, appName) {
		t.Fatalf("cache dir %q devrait contenir %q", dir, appName)
	}
	if filepath.Base(dir) != "bin" {
		t.Fatalf("cache dir devrait finir par 'bin', got %q", dir)
	}
}

func TestEnsureCacheDirCreates(t *testing.T) {
	tmp := t.TempDir()
	// Sous Windows, os.UserCacheDir() utilise %LocalAppData%.
	t.Setenv("LOCALAPPDATA", tmp)
	dir, err := EnsureCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("le dossier devrait exister: %v", err)
	}
}
```

- [ ] **Step 2: Lancer → échec attendu**

Run : `go test ./internal/runtime/ -v`
Expected : FAIL (`undefined: CacheDir`).

- [ ] **Step 3: Implémentation**

`internal/runtime/paths.go` :
```go
package runtime

import (
	"os"
	"path/filepath"
)

// appName = dossier sous le cache utilisateur.
const appName = "AISubtitlePro"

// CacheDir = dossier d'extraction des binaires embarqués.
// Windows: %LOCALAPPDATA%\AISubtitlePro\bin
func CacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName, "bin"), nil
}

// EnsureCacheDir crée le dossier de cache si besoin et renvoie son chemin.
func EnsureCacheDir() (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
```

- [ ] **Step 4: Lancer → succès attendu**

Run : `go test ./internal/runtime/ -v`
Expected : PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/runtime/
git commit -m "feat(runtime): chemins du cache binaires (%LOCALAPPDATA%/AISubtitlePro/bin)"
```

---

## Task 5: Fenêtre sombre + binding témoin

On configure la fenêtre (fond sombre, titre, taille) et on expose une méthode témoin pour valider le pont Go↔frontend.

**Files:**
- Modify: `main.go`
- Modify: `app.go`

- [ ] **Step 1: Configurer la fenêtre dans `main.go`**

Dans `main.go`, dans l'appel `wails.Run(&options.App{...})`, fixer/ajouter :
```go
		Title:            "AI Subtitle Pro",
		Width:            900,
		Height:           760,
		MinWidth:         760,
		MinHeight:        600,
		BackgroundColour: &options.RGBA{R: 13, G: 13, B: 13, A: 1},
```
> `options` provient de `github.com/wailsapp/wails/v2/pkg/options` (déjà importé par le scaffold). Garder `Assets`, `OnStartup: app.startup`, `Bind: []interface{}{app}` tels que générés.

- [ ] **Step 2: Ajouter une méthode témoin dans `app.go`**

Dans `app.go`, ajouter sur le type `App` :
```go
// AppInfo renvoie le nom de l'app (binding témoin pour valider le pont Go↔UI).
func (a *App) AppInfo() string {
	return "AI Subtitle Pro"
}
```
> Supprimer la méthode d'exemple `Greet` générée par le scaffold si présente.

- [ ] **Step 3: Builder pour vérifier la compilation**

Run : `cd E:/AISubtitle && wails build`
Expected : `build/bin/AISubtitlePro.exe` recompilé sans erreur.

- [ ] **Step 4: Commit**

```bash
git add main.go app.go
git commit -m "feat(ui): fenetre sombre + binding temoin AppInfo"
```

---

## Task 6: Placeholder Svelte sombre

Remplacer le contenu d'exemple du template par un écran d'accueil sombre minimal (les vrais composants viendront au Plan 5).

**Files:**
- Modify: `frontend/src/App.svelte`

- [ ] **Step 1: Remplacer `frontend/src/App.svelte`**

Contenu complet :
```svelte
<script>
  import { onMount } from "svelte";
  import { AppInfo } from "../wailsjs/go/main/App";

  let title = "AI Subtitle Pro";
  onMount(async () => {
    try {
      title = await AppInfo();
    } catch (e) {
      // pont indisponible (mode hors-Wails) : on garde le titre par défaut
    }
  });
</script>

<main>
  <h1>{title}</h1>
  <p class="subtitle">Fondations prêtes — l'interface arrive au Plan 5.</p>
</main>

<style>
  :global(body) {
    margin: 0;
    background: #0d0d0d;
    color: #eaeaea;
    font-family: "Segoe UI", system-ui, sans-serif;
  }
  main {
    height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
  }
  h1 {
    font-size: 2rem;
    font-weight: 700;
    letter-spacing: -0.02em;
  }
  .subtitle {
    color: #8a8a8a;
    font-size: 0.95rem;
  }
</style>
```
> Le chemin d'import `../wailsjs/go/main/App` est généré par Wails au build/`wails dev`. Si l'IDE le signale absent avant un 1ᵉʳ build, lancer `wails generate module` ou `wails build` une fois.

- [ ] **Step 2: Lancer en mode dev pour vérifier visuellement**

Run : `cd E:/AISubtitle && wails dev`
Expected : une fenêtre sombre s'ouvre, affiche « AI Subtitle Pro » au centre (valeur renvoyée par le binding Go). Fermer la fenêtre pour arrêter.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/App.svelte
git commit -m "feat(ui): ecran d'accueil sombre minimal relie au binding"
```

---

## Task 7: Vérification finale du jalon

**Files:** aucun (vérification)

- [ ] **Step 1: Toute la suite de tests passe**

Run : `cd E:/AISubtitle && go test ./...`
Expected : `ok github.com/Ne9ative/AISubtitle/internal/config` et `.../internal/runtime`, aucun échec.

- [ ] **Step 2: Build de production OK**

Run : `wails build`
Expected : `build/bin/AISubtitlePro.exe` généré ; double-clic → fenêtre sombre.

- [ ] **Step 3: Vérifier qu'aucun secret/gros binaire n'est suivi par git**

Run :
```bash
git status --short | grep -iE 'config\.json|\.exe|\.gguf|node_modules|frontend/dist|build/bin' && echo "ALERTE" || echo "OK propre"
```
Expected : `OK propre` (ces éléments sont ignorés par `.gitignore`).

- [ ] **Step 4: Push**

```bash
git push
```

---

## Notes pour les plans suivants (rappel)

- **Plan 2** créera `internal/subs` (basé sur `github.com/asticode/go-astisub`) et `internal/mkv`. Les signatures (`subs.Batch`, etc.) seront définies à ce moment-là, contre le code réel.
- **Plan 4** ajoutera l'embarquage réel des binaires dans `internal/runtime` (`go:embed`), en réutilisant `CacheDir/EnsureCacheDir` posés ici.
- La méthode témoin `AppInfo` pourra être retirée une fois les vrais bindings en place (Plan 4/5).
