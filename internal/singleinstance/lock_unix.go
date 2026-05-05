//go:build !windows

package singleinstance

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// Acquire creates and exclusively locks a file in dataDir.
func Acquire(dataDir string) (*Lock, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	path := lockPath(dataDir)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, ErrAlreadyRunning
		}
		return nil, fmt.Errorf("lock %s: %w", filepath.Base(path), err)
	}

	if err := writePID(file); err != nil {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
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
	unlockErr := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	closeErr := l.file.Close()
	_ = os.Remove(l.path)
	l.file = nil
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}
