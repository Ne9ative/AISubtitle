package runtime

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed binaries/mkvmerge.exe binaries/mkvextract.exe
var embeddedBins embed.FS

// EnsureMkvtoolnix extrait mkvmerge/mkvextract dans le cache et renvoie leurs chemins.
func EnsureMkvtoolnix() (mkvmerge, mkvextract string, err error) {
	dir, err := EnsureCacheDir()
	if err != nil {
		return "", "", err
	}
	return extractMkvtoolnixTo(dir)
}

func extractMkvtoolnixTo(dir string) (mkvmerge, mkvextract string, err error) {
	mkvmerge, err = extractEmbedded("binaries/mkvmerge.exe", filepath.Join(dir, "mkvmerge.exe"))
	if err != nil {
		return "", "", err
	}
	mkvextract, err = extractEmbedded("binaries/mkvextract.exe", filepath.Join(dir, "mkvextract.exe"))
	if err != nil {
		return "", "", err
	}
	return mkvmerge, mkvextract, nil
}

// extractEmbedded écrit un fichier embarqué vers dest. Idempotent : saute la
// réécriture si dest existe déjà avec la même taille.
func extractEmbedded(embedPath, dest string) (string, error) {
	data, err := embeddedBins.ReadFile(embedPath)
	if err != nil {
		return "", fmt.Errorf("runtime: lecture embarquée %q: %w", embedPath, err)
	}
	if fi, err := os.Stat(dest); err == nil && fi.Size() == int64(len(data)) {
		return dest, nil
	}
	if err := os.WriteFile(dest, data, 0o755); err != nil {
		return "", fmt.Errorf("runtime: écriture %q: %w", dest, err)
	}
	return dest, nil
}
