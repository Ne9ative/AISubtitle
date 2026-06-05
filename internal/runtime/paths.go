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
