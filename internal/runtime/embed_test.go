package runtime

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestExtractMkvtoolnix(t *testing.T) {
	dir := t.TempDir()
	mkvmerge, mkvextract, err := extractMkvtoolnixTo(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{mkvmerge, mkvextract} {
		fi, err := os.Stat(p)
		if err != nil || fi.Size() == 0 {
			t.Fatalf("binaire non extrait: %s (%v)", p, err)
		}
	}
	// Idempotence : un 2e appel ne doit pas échouer (même taille → saut).
	if _, _, err := extractMkvtoolnixTo(dir); err != nil {
		t.Fatalf("2e extraction: %v", err)
	}
	// Le binaire extrait est exécutable et répond.
	out, err := exec.Command(mkvmerge, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("mkvmerge --version: %v: %s", err, out)
	}
	if !strings.Contains(strings.ToLower(string(out)), "mkvmerge") {
		t.Fatalf("sortie --version inattendue: %s", out)
	}
}
