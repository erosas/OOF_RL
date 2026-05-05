//go:build windows

package singleinstance

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// Acquire creates and exclusively locks a file in dataDir. Windows releases the
// lock when the process exits, so a stale file after a crash does not block the
// next app launch.
func Acquire(dataDir string) (*Lock, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	path := lockPath(dataDir)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	var ol windows.Overlapped
	err = windows.LockFileEx(
		windows.Handle(file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		1,
		0,
		&ol,
	)
	if err != nil {
		_ = file.Close()
		if err == windows.ERROR_LOCK_VIOLATION {
			return nil, ErrAlreadyRunning
		}
		return nil, fmt.Errorf("lock %s: %w", filepath.Base(path), err)
	}

	if err := writePID(file); err != nil {
		_ = windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &ol)
		_ = file.Close()
		return nil, err
	}

	return &Lock{file: file, path: path}, nil
}

// Release unlocks and closes the held file lock.
func (l *Lock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	var ol windows.Overlapped
	unlockErr := windows.UnlockFileEx(windows.Handle(l.file.Fd()), 0, 1, 0, &ol)
	closeErr := l.file.Close()
	_ = os.Remove(l.path)
	l.file = nil
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}
