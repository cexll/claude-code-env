//go:build !darwin && !linux

package term

import "errors"

// State is a placeholder for unsupported platforms.
type State struct{}

func IsTerminal(fd int) bool { return false }

func MakeRaw(fd int) (*State, error) {
	return nil, errors.New("terminal raw mode not supported on this platform")
}

func Restore(fd int, state *State) error { return nil }

func GetSize(fd int) (int, int, error) {
	return 0, 0, errors.New("terminal size not available")
}
