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
