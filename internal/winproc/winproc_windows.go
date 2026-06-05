//go:build windows

// Package winproc masque la fenêtre console des sous-processus sous Windows.
package winproc

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// HideWindow empêche le flash d'une fenêtre console au lancement d'un sous-processus.
func HideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
