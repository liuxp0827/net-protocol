//go:build freebsd || solaris
// +build freebsd solaris

package reexec

import (
	"os/exec"
)

// Self returns the path to the current process's binary.
// Uses os.Args[0].
func Self() string {
	return naiveSelf()
}

// Command returns *exec.Cmd which have Path as current binary.
// For example if current binary is "docker" at "/usr/bin/", then cmd.Path will
// be set to "/usr/bin/docker".
func Command(args ...string) *exec.Cmd {
	return &exec.Cmd{
		Path: Self(),
		Args: args,
	}
}
