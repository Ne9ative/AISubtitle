//go:build !windows

package mkv

import "os/exec"

// hideWindow est un no-op hors Windows.
func hideWindow(cmd *exec.Cmd) {}
