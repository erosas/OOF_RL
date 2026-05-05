package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	archiveDirName = "logs"
	logPrefix      = "oof_rl-"
	logExt         = ".log"
)

// RotateOptions controls startup log rotation.
type RotateOptions struct {
	// Retain is the number of archived logs to keep. Values less than zero are
	// treated as zero.
	Retain int
	// Now supplies the timestamp used for archive filenames. When nil, time.Now
	// is used.
	Now func() time.Time
}

// Rotate moves an existing current log file into dataDir/logs and prunes old
// archives. It is safe to call when the current log does not exist or is empty.
func Rotate(currentLogPath string, opts RotateOptions) error {
	if currentLogPath == "" {
		return nil
	}

	info, err := os.Stat(currentLogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return pruneArchives(filepath.Dir(currentLogPath), opts.Retain)
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("log path is a directory: %s", currentLogPath)
	}
	if info.Size() == 0 {
		_ = os.Remove(currentLogPath)
		return pruneArchives(filepath.Dir(currentLogPath), opts.Retain)
	}

	dataDir := filepath.Dir(currentLogPath)
	archiveDir := filepath.Join(dataDir, archiveDirName)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return err
	}

	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	archivePath := uniqueArchivePath(archiveDir, now())
	if err := os.Rename(currentLogPath, archivePath); err != nil {
		return err
	}

	return pruneArchives(dataDir, opts.Retain)
}

func uniqueArchivePath(archiveDir string, ts time.Time) string {
	base := filepath.Join(archiveDir, logPrefix+ts.Format("20060102-150405")+logExt)
	if _, err := os.Stat(base); os.IsNotExist(err) {
		return base
	}
	for i := 1; ; i++ {
		path := filepath.Join(archiveDir, fmt.Sprintf("%s%s-%02d%s", logPrefix, ts.Format("20060102-150405"), i, logExt))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
	}
}

func pruneArchives(dataDir string, retain int) error {
	if retain < 0 {
		retain = 0
	}
	archiveDir := filepath.Join(dataDir, archiveDirName)
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var archives []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, logPrefix) || !strings.HasSuffix(name, logExt) {
			continue
		}
		archives = append(archives, entry)
	}
	sort.Slice(archives, func(i, j int) bool {
		infoI, errI := archives[i].Info()
		infoJ, errJ := archives[j].Info()
		if errI == nil && errJ == nil && !infoI.ModTime().Equal(infoJ.ModTime()) {
			return infoI.ModTime().After(infoJ.ModTime())
		}
		return archives[i].Name() > archives[j].Name()
	})

	if len(archives) <= retain {
		return nil
	}
	for _, entry := range archives[retain:] {
		if err := os.Remove(filepath.Join(archiveDir, entry.Name())); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
