package runtime

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Build llama.cpp épinglé (variante CUDA 12.4, Windows x64).
const (
	llamaBuild   = "b9524"
	llamaCudaURL = "https://github.com/ggml-org/llama.cpp/releases/download/" + llamaBuild + "/llama-" + llamaBuild + "-bin-win-cuda-12.4-x64.zip"
	cudartURL    = "https://github.com/ggml-org/llama.cpp/releases/download/" + llamaBuild + "/cudart-llama-bin-win-cuda-12.4-x64.zip"
)

// Modèle GGUF par défaut, téléchargé au 1er run si le dossier models/ est vide.
const (
	defaultModelName = "gemma-3-12b-it-Q4_K_M.gguf"
	defaultModelURL  = "https://huggingface.co/ggml-org/gemma-3-12b-it-GGUF/resolve/main/" + defaultModelName
)

// EnsureDefaultModel garantit qu'un modèle GGUF est présent dans modelsDir.
// Si le fichier par défaut manque, il est téléchargé (Gemma 3 12B Q4, ~7 Go).
func EnsureDefaultModel(ctx context.Context, modelsDir string, progress ProgressFunc) (string, error) {
	dest := filepath.Join(modelsDir, defaultModelName)
	if fi, err := os.Stat(dest); err == nil && fi.Size() > 0 {
		return dest, nil
	}
	if err := os.MkdirAll(modelsDir, 0o755); err != nil {
		return "", err
	}
	if err := downloadTo(ctx, defaultModelURL, dest, "Téléchargement du modèle Gemma 3 12B (~7 Go)", progress); err != nil {
		os.Remove(dest) // nettoyer un téléchargement partiel
		return "", err
	}
	return dest, nil
}

// ProgressFunc rapporte l'avancement d'un téléchargement (octets ; total=-1 si inconnu).
type ProgressFunc func(stage string, done, total int64)

// EnsureLlamaServer s'assure que llama-server.exe (CUDA) est présent dans le cache,
// en le téléchargeant au 1er appel. Renvoie le chemin de llama-server.exe.
func EnsureLlamaServer(ctx context.Context, progress ProgressFunc) (string, error) {
	base, err := EnsureCacheDir()
	if err != nil {
		return "", err
	}
	return ensureLlamaServerIn(ctx, base, progress)
}

func ensureLlamaServerIn(ctx context.Context, base string, progress ProgressFunc) (string, error) {
	dir := filepath.Join(base, "llama-cuda-"+llamaBuild)
	server := filepath.Join(dir, "llama-server.exe")
	if _, err := os.Stat(server); err == nil {
		return server, nil // déjà installé
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := downloadAndUnzip(ctx, llamaCudaURL, dir, "Téléchargement de llama.cpp (CUDA)", progress); err != nil {
		return "", err
	}
	if err := downloadAndUnzip(ctx, cudartURL, dir, "Téléchargement du runtime CUDA", progress); err != nil {
		return "", err
	}
	if _, err := os.Stat(server); err != nil {
		return "", fmt.Errorf("runtime: llama-server.exe absent après extraction dans %s", dir)
	}
	return server, nil
}

func downloadAndUnzip(ctx context.Context, url, destDir, stage string, progress ProgressFunc) error {
	tmp, err := os.CreateTemp("", "aisub-dl-*.zip")
	if err != nil {
		return err
	}
	name := tmp.Name()
	tmp.Close()
	defer os.Remove(name)
	if err := downloadTo(ctx, url, name, stage, progress); err != nil {
		return err
	}
	return unzipInto(name, destDir)
}

// downloadTo télécharge url vers destPath en flux, avec rapport de progression.
func downloadTo(ctx context.Context, url, destPath, stage string, progress ProgressFunc) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("runtime: téléchargement %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("runtime: %s a renvoyé HTTP %d", url, resp.StatusCode)
	}
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	var done int64
	buf := make([]byte, 1<<20)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
			done += int64(n)
			if progress != nil {
				progress(stage, done, total)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	return out.Close()
}

// unzipInto extrait toutes les entrées (aplaties) de zipPath dans destDir, de
// sorte que llama-server.exe et ses DLL se retrouvent dans le même dossier.
func unzipInto(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if err := writeZipEntry(f, filepath.Join(destDir, filepath.Base(f.Name))); err != nil {
			return err
		}
	}
	return nil
}

func writeZipEntry(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}
