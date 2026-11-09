//go:build !windows

package svnx

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}
