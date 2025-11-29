//go:build linux

package term

import "syscall"

const (
	ioctlGetTermios = syscall.TCGETS
	ioctlSetTermios = syscall.TCSETS
	ioctlGetWinsize = syscall.TIOCGWINSZ
)
