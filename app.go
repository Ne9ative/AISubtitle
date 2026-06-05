package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Ne9ative/AISubtitle/internal/config"
	"github.com/Ne9ative/AISubtitle/internal/engine"
	"github.com/Ne9ative/AISubtitle/internal/mkv"
	"github.com/Ne9ative/AISubtitle/internal/pipeline"
	"github.com/Ne9ative/AISubtitle/internal/runtime"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App est la structure liée à Wails (pont Go ↔ frontend).
type App struct {
	ctx     context.Context
	mu      sync.Mutex
	cancel  context.CancelFunc
	running bool
}

// NewApp crée la structure App.
func NewApp() *App { return &App{} }

// startup mémorise le contexte Wails (nécessaire pour émettre les events).
func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// AppInfo renvoie le nom de l'app (témoin du pont Go↔UI).
func (a *App) AppInfo() string { return "AI Subtitle Pro" }

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// GetConfig renvoie la configuration persistée (ou les valeurs par défaut).
func (a *App) GetConfig() config.Config {
	cfg, _ := config.Load(configPath())
	return cfg
}

// SaveConfig persiste la configuration.
func (a *App) SaveConfig(c config.Config) error {
	return config.Save(configPath(), c)
}

// ---------------------------------------------------------------------------
// Modèles & pistes
// ---------------------------------------------------------------------------

// ListModels liste les fichiers .gguf du dossier models/ situé à côté de l'exe.
func (a *App) ListModels() []string {
	out := []string{}
	entries, err := os.ReadDir(modelsDir())
	if err != nil {
		return out
	}
	for _, e := range entries {
		name := e.Name()
		low := strings.ToLower(name)
		if e.IsDir() || !strings.HasSuffix(low, ".gguf") {
			continue
		}
		if strings.HasPrefix(low, "mmproj") {
			continue // fichier de projection multimodale, pas un modèle de langue
		}
		out = append(out, name)
	}
	return out
}

// ScanTracks renvoie les pistes de sous-titres d'une vidéo.
func (a *App) ScanTracks(path string) ([]mkv.Track, error) {
	mkvmerge, mkvextract, err := runtime.EnsureMkvtoolnix()
	if err != nil {
		return nil, err
	}
	tool := mkv.Tool{Mkvmerge: mkvmerge, Mkvextract: mkvextract}
	return tool.ScanSubtitleTracks(a.ctx, path)
}

// SelectVideo ouvre une boîte de dialogue native et renvoie le chemin choisi
// (chaîne vide si l'utilisateur annule).
func (a *App) SelectVideo() (string, error) {
	return wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "Choisir une vidéo",
		Filters: []wruntime.FileFilter{
			{DisplayName: "Vidéos (*.mkv;*.mp4;*.avi)", Pattern: "*.mkv;*.mp4;*.avi"},
			{DisplayName: "Tous les fichiers (*.*)", Pattern: "*.*"},
		},
	})
}

// ---------------------------------------------------------------------------
// Traduction
// ---------------------------------------------------------------------------

// TranslateRequest = paramètres d'un job envoyés par le frontend.
type TranslateRequest struct {
	VideoPath string `json:"videoPath"`
	Engine    string `json:"engine"` // "Local" ou "Gemini"
	Model     string `json:"model"`
	APIKey    string `json:"apiKey"`
	SrcLang   string `json:"srcLang"`
	TargetLang string `json:"targetLang"`
	TrackID   int    `json:"trackID"`
	TestMode  bool   `json:"testMode"`
}

// StartTranslation lance un job en arrière-plan.
// Events émis : progress, log, download, done, error.
func (a *App) StartTranslation(req TranslateRequest) {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		a.emitError("A job is already running.")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.running = true
	a.mu.Unlock()
	go a.runJob(ctx, req)
}

// Cancel annule le job en cours.
func (a *App) Cancel() {
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	a.mu.Unlock()
}

func (a *App) runJob(ctx context.Context, req TranslateRequest) {
	defer func() {
		a.mu.Lock()
		a.running = false
		a.cancel = nil
		a.mu.Unlock()
	}()

	mkvmerge, mkvextract, err := runtime.EnsureMkvtoolnix()
	if err != nil {
		a.emitError("MKV tools unavailable: " + err.Error())
		return
	}
	tool := mkv.Tool{Mkvmerge: mkvmerge, Mkvextract: mkvextract}

	tr, err := a.buildTranslator(ctx, req)
	if err != nil {
		if ctx.Err() != nil {
			a.emitCancelled()
			return
		}
		a.emitError(err.Error())
		return
	}
	defer tr.Close()

	cfg, _ := config.Load(configPath())
	opts := pipeline.Options{
		VideoPath:   req.VideoPath,
		TrackID:     req.TrackID,
		SrcLang:     req.SrcLang,
		TargetLang:  req.TargetLang,
		BatchSize:   cfg.BatchSize,
		ContextSize: cfg.ContextSize,
		TestMode:    req.TestMode,
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 12
	}
	if opts.ContextSize <= 0 {
		opts.ContextSize = 2
	}

	out, err := pipeline.Run(ctx, tool, tr, opts, a)
	if err != nil {
		if ctx.Err() != nil {
			a.emitCancelled()
			return
		}
		a.emitError(err.Error())
		return
	}
	a.Log("✨ Done: " + filepath.Base(out))
	wruntime.EventsEmit(a.ctx, "done", out)
}

func (a *App) buildTranslator(ctx context.Context, req TranslateRequest) (engine.Translator, error) {
	if req.Engine == "Local" {
		dl := func(stage string, done, total int64) {
			wruntime.EventsEmit(a.ctx, "download", map[string]any{"stage": stage, "done": done, "total": total})
		}
		modelPath, err := a.resolveModel(ctx, req.Model, dl)
		if err != nil {
			return nil, fmt.Errorf("model: %w", err)
		}
		a.Log("Preparing local engine (CUDA)…")
		server, err := runtime.EnsureLlamaServer(ctx, dl)
		if err != nil {
			return nil, fmt.Errorf("engine download: %w", err)
		}
		local := engine.NewLocal(server, modelPath)
		a.Log("Starting llama-server (loading model onto GPU)…")
		if err := local.Start(ctx); err != nil {
			return nil, fmt.Errorf("starting local engine: %w", err)
		}
		return local, nil
	}
	if strings.TrimSpace(req.APIKey) == "" {
		return nil, fmt.Errorf("Gemini API key missing")
	}
	return engine.NewGemini(req.APIKey, req.Model), nil
}

// resolveModel renvoie le chemin du modèle local à utiliser ; si aucun modèle
// valide n'est sélectionné, télécharge le modèle par défaut (Gemma 3 12B).
func (a *App) resolveModel(ctx context.Context, model string, dl runtime.ProgressFunc) (string, error) {
	if model != "" {
		p := filepath.Join(modelsDir(), model)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, nil
		}
	}
	a.Log("No local model — downloading the default model (Gemma 3 12B, ~7 GB)…")
	return runtime.EnsureDefaultModel(ctx, modelsDir(), dl)
}

// ---------------------------------------------------------------------------
// pipeline.Reporter
// ---------------------------------------------------------------------------

// Progress implémente pipeline.Reporter.
func (a *App) Progress(done, total int) {
	wruntime.EventsEmit(a.ctx, "progress", map[string]int{"done": done, "total": total})
}

// Log implémente pipeline.Reporter.
func (a *App) Log(msg string) {
	wruntime.EventsEmit(a.ctx, "log", msg)
}

func (a *App) emitError(msg string) {
	a.Log("❌ " + msg)
	wruntime.EventsEmit(a.ctx, "error", msg)
}

func (a *App) emitCancelled() {
	a.Log("⏹ Cancelled.")
	wruntime.EventsEmit(a.ctx, "error", "Cancelled.")
}

// ---------------------------------------------------------------------------
// Chemins
// ---------------------------------------------------------------------------

func appDir() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	wd, _ := os.Getwd()
	return wd
}

func modelsDir() string {
	d := filepath.Join(appDir(), "models")
	if dirExists(d) {
		return d
	}
	if wd, err := os.Getwd(); err == nil {
		if alt := filepath.Join(wd, "models"); dirExists(alt) {
			return alt
		}
	}
	return d
}

func configPath() string { return filepath.Join(appDir(), "config.json") }

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}
