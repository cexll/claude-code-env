//go:build darwin || linux

package term

import (
	"syscall"
	"unsafe"
)

// State captures the original terminal settings so they can be restored.
type State struct {
	termios syscall.Termios
}

// IsTerminal reports whether the file descriptor is a terminal.
func IsTerminal(fd int) bool {
	var st syscall.Termios
	return tcget(fd, &st) == nil
}

// MakeRaw puts the terminal into raw mode and returns the previous state.
func MakeRaw(fd int) (*State, error) {
	var old syscall.Termios
	if err := tcget(fd, &old); err != nil {
		return nil, err
	}

	newState := old
	newState.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	newState.Oflag &^= syscall.OPOST
	newState.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	newState.Cflag &^= syscall.CSIZE | syscall.PARENB
	newState.Cflag |= syscall.CS8
	newState.Cc[syscall.VMIN] = 1
	newState.Cc[syscall.VTIME] = 0

	if err := tcset(fd, &newState); err != nil {
		return nil, err
	}

	return &State{termios: old}, nil
}

// Restore resets the terminal to a previous state.
func Restore(fd int, state *State) error {
	if state == nil {
		return syscall.EINVAL
	}
	return tcset(fd, &state.termios)
}

// GetSize returns the terminal width and height in characters.
func GetSize(fd int) (int, int, error) {
	var ws winsize
	if err := ioctlWinsize(fd, &ws); err != nil {
		return 0, 0, err
	}
	return int(ws.Col), int(ws.Row), nil
}

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func tcget(fd int, p *syscall.Termios) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), ioctlGetTermios, uintptr(unsafe.Pointer(p)))
	if errno != 0 {
		return errno
	}
	return nil
}

func tcset(fd int, p *syscall.Termios) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), ioctlSetTermios, uintptr(unsafe.Pointer(p)))
	if errno != 0 {
		return errno
	}
	return nil
}

func ioctlWinsize(fd int, ws *winsize) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), ioctlGetWinsize, uintptr(unsafe.Pointer(ws)))
	if errno != 0 {
		return errno
	}
	return nil
}
