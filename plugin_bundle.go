package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var bundledPublicPluginIDs = map[string]struct{}{
	"ballchasing": {},
	"dashboard":   {},
	"live":        {},
	"ranks":       {},
	"session":     {},
}

func seedBundledWASMPlugins(destDir string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	return seedBundledWASMPluginsFromExecutable(exePath, destDir)
}

func seedBundledWASMPluginsFromExecutable(exePath, destDir string) error {
	srcDir := filepath.Join(filepath.Dir(exePath), "plugins")
	info, err := os.Stat(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat bundled plugins: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("bundled plugins path is not a directory: %s", srcDir)
	}
	return copyBundledWASMPlugins(srcDir, destDir)
}

func copyBundledWASMPlugins(srcDir, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("read bundled plugins: %w", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create plugin dir: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		if entry.IsDir() {
			pluginID := strings.ToLower(entry.Name())
			if !isBundledPublicPluginID(pluginID) {
				continue
			}
			if err := copyPluginAssetDir(srcPath, filepath.Join(destDir, pluginID)); err != nil {
				return err
			}
			continue
		}
		pluginID, ok := bundledPublicPluginIDFromWASM(entry.Name())
		if !ok {
			continue
		}
		if changed, err := copyFileIfChanged(srcPath, filepath.Join(destDir, pluginID+".wasm")); err != nil {
			return err
		} else if changed {
			log.Printf("[core] installed bundled wasm plugin: %s.wasm", pluginID)
		}
	}
	return nil
}

func bundledPublicPluginIDFromWASM(name string) (string, bool) {
	lower := strings.ToLower(name)
	if !strings.HasSuffix(lower, ".wasm") {
		return "", false
	}
	pluginID := strings.TrimSuffix(lower, ".wasm")
	return pluginID, isBundledPublicPluginID(pluginID)
}

func isBundledPublicPluginID(pluginID string) bool {
	_, ok := bundledPublicPluginIDs[pluginID]
	return ok
}

func copyPluginAssetDir(srcDir, destDir string) error {
	return filepath.WalkDir(srcDir, func(srcPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, srcPath)
		if err != nil {
			return err
		}
		destPath := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		if changed, err := copyFileIfChanged(srcPath, destPath); err != nil {
			return err
		} else if changed {
			log.Printf("[core] installed bundled plugin asset: %s", filepath.Join(filepath.Base(destDir), rel))
		}
		return nil
	})
}

func copyFileIfChanged(srcPath, destPath string) (bool, error) {
	src, err := os.ReadFile(srcPath)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", srcPath, err)
	}
	if dest, err := os.ReadFile(destPath); err == nil && bytes.Equal(src, dest) {
		return false, nil
	} else if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("read %s: %w", destPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return false, fmt.Errorf("create %s: %w", filepath.Dir(destPath), err)
	}
	if err := os.WriteFile(destPath, src, 0644); err != nil {
		return false, fmt.Errorf("write %s: %w", destPath, err)
	}
	return true, nil
}
