package logging

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRotateArchivesExistingLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "oof_rl.log")
	if err := os.WriteFile(logPath, []byte("previous run\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err := Rotate(logPath, RotateOptions{
		Retain: 5,
		Now: func() time.Time {
			return time.Date(2026, 5, 4, 21, 30, 12, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("current log exists after rotate: %v", err)
	}
	archived := filepath.Join(dir, "logs", "oof_rl-20260504-213012.log")
	data, err := os.ReadFile(archived)
	if err != nil {
		t.Fatalf("ReadFile archived: %v", err)
	}
	if string(data) != "previous run\n" {
		t.Fatalf("archived content = %q", string(data))
	}
}

func TestRotateNoExistingLogStillPrunes(t *testing.T) {
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		t.Fatal(err)
	}
	createArchive(t, archiveDir, "oof_rl-20260504-210000.log", time.Date(2026, 5, 4, 21, 0, 0, 0, time.UTC))
	createArchive(t, archiveDir, "oof_rl-20260504-211000.log", time.Date(2026, 5, 4, 21, 10, 0, 0, time.UTC))

	if err := Rotate(filepath.Join(dir, "oof_rl.log"), RotateOptions{Retain: 1}); err != nil {
		t.Fatalf("Rotate: %v", err)
	}

	if _, err := os.Stat(filepath.Join(archiveDir, "oof_rl-20260504-210000.log")); !os.IsNotExist(err) {
		t.Fatalf("old archive should be pruned: %v", err)
	}
	if _, err := os.Stat(filepath.Join(archiveDir, "oof_rl-20260504-211000.log")); err != nil {
		t.Fatalf("new archive should remain: %v", err)
	}
}

func TestRotatePrunesToRetentionLimit(t *testing.T) {
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 7; i++ {
		ts := time.Date(2026, 5, 4, 20, i, 0, 0, time.UTC)
		createArchive(t, archiveDir, "oof_rl-20260504-200"+string(rune('0'+i))+"00.log", ts)
	}

	if err := os.WriteFile(filepath.Join(dir, "oof_rl.log"), []byte("current\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Rotate(filepath.Join(dir, "oof_rl.log"), RotateOptions{
		Retain: 3,
		Now: func() time.Time {
			return time.Date(2026, 5, 4, 21, 0, 0, 0, time.UTC)
		},
	}); err != nil {
		t.Fatalf("Rotate: %v", err)
	}

	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	for _, entry := range entries {
		if !entry.IsDir() {
			count++
		}
	}
	if count != 3 {
		t.Fatalf("archive count = %d, want 3", count)
	}
}

func TestRotateIgnoresEmptyCurrentLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "oof_rl.log")
	if err := os.WriteFile(logPath, nil, 0644); err != nil {
		t.Fatal(err)
	}

	if err := Rotate(logPath, RotateOptions{Retain: 5}); err != nil {
		t.Fatalf("Rotate: %v", err)
	}

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("empty current log should be removed, got err %v", err)
	}
}

func createArchive(t *testing.T, dir, name string, modTime time.Time) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(name), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
}
