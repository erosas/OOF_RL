package singleinstance

import (
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
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

func TestAcquireWritesPID(t *testing.T) {
	dir := t.TempDir()

	lock, err := Acquire(dir)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer lock.Release()

	if _, err := lock.file.Seek(0, 0); err != nil {
		t.Fatalf("Seek: %v", err)
	}
	data, err := io.ReadAll(lock.file)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	got := strings.TrimSpace(string(data))
	want := strconv.Itoa(os.Getpid())
	if got != want {
		t.Fatalf("PID = %q, want %q", got, want)
	}
}
