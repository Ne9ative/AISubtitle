//go:build windows

package mkv

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// hideWindow empêche l'apparition d'une fenêtre console (flash noir)
// au lancement d'un sous-processus.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
