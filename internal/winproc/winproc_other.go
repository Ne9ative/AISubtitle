//go:build !windows

package winproc

import "os/exec"

// HideWindow est un no-op hors Windows.
func HideWindow(cmd *exec.Cmd) {}
