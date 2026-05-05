package singleinstance

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireRejectsSecondLock(t *testing.T) {
	dir := t.TempDir()

	first, err := Acquire(dir)
	if err != nil {
		t.Fatalf("Acquire first: %v", err)
	}
	defer first.Release()

	second, err := Acquire(dir)
	if !errors.Is(err, ErrAlreadyRunning) {
		if second != nil {
			_ = second.Release()
		}
		t.Fatalf("Acquire second error = %v, want %v", err, ErrAlreadyRunning)
	}
}

func TestAcquireReleaseAllowsReacquire(t *testing.T) {
	dir := t.TempDir()

	first, err := Acquire(dir)
	if err != nil {
		t.Fatalf("Acquire first: %v", err)
	}
	if err := first.Release(); err != nil {
		t.Fatalf("Release first: %v", err)
	}

	second, err := Acquire(dir)
	if err != nil {
		t.Fatalf("Acquire second: %v", err)
	}
	defer second.Release()
}

func TestAcquireCreatesLockFileInDataDir(t *testing.T) {
	dir := t.TempDir()

	lock, err := Acquire(dir)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer lock.Release()

	if _, err := os.Stat(filepath.Join(dir, lockFileName)); err != nil {
		t.Fatalf("lock file should exist in data dir: %v", err)
	}
}
