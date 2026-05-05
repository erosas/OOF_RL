package singleinstance

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

const lockFileName = "oof_rl.lock"

// ErrAlreadyRunning is returned when another OOF RL process already holds the
// shared app lock.
var ErrAlreadyRunning = errors.New("OOF RL is already running")

// Lock represents the held single-instance lock.
type Lock struct {
	handle windows.Handle
	path   string
}

// LockPath returns the app-data lock file path used for the single-instance guard.
func LockPath(dataDir string) string {
	return filepath.Join(dataDir, lockFileName)
}

// Acquire creates a lock file in dataDir and opens it without sharing. Windows
// holds that file handle until Release or process exit, causing any second app
// instance to fail immediately when it tries to open the same file.
func Acquire(dataDir string) (*Lock, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	path := LockPath(dataDir)
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	handle, err := windows.CreateFile(
		name,
		windows.GENERIC_WRITE,
		0,
		nil,
		windows.OPEN_ALWAYS,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		if errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return nil, ErrAlreadyRunning
		}
		return nil, err
	}

	return &Lock{handle: handle, path: path}, nil
}

// Path returns the held lock file path.
func (l *Lock) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

// Release closes the held file handle.
func (l *Lock) Release() error {
	if l == nil || l.handle == 0 {
		return nil
	}
	err := windows.CloseHandle(l.handle)
	l.handle = 0
	_ = os.Remove(l.path)
	return err
}
