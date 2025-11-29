//go:build darwin

package term

import "syscall"

const (
	ioctlGetTermios = syscall.TIOCGETA
	ioctlSetTermios = syscall.TIOCSETA
	ioctlGetWinsize = syscall.TIOCGWINSZ
)
