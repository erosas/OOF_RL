package singleinstance

import (
	"errors"
	"os"
	"path/filepath"
)

const lockFileName = "oof_rl.lock"

// ErrAlreadyRunning is returned when another OOF RL process already holds the
// shared app lock.
var ErrAlreadyRunning = errors.New("OOF RL is already running")

// Lock represents the held single-instance lock.
type Lock struct {
	file *os.File
	path string
}

func lockPath(dataDir string) string {
	return filepath.Join(dataDir, lockFileName)
}
