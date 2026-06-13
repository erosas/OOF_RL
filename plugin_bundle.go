package main

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Release builds embed the public plugin wasm + assets here
// (scripts/package-release.ps1 populates bundled/plugins before go build),
// which makes oof_rl.exe fully self-contained: users keep the single exe
// wherever they want and the app seeds %LOCALAPPDATA%\OOF_RL\plugins from
// the embedded copies at startup. Dev builds embed only the .gitkeep
// placeholder and seeding is a no-op.
//
//go:embed all:bundled/plugins
var bundledPluginsFS embed.FS

var bundledPublicPluginIDs = map[string]struct{}{
	"dashboard": {},
	"live":      {},
	"ranks":     {},
	"session":   {},
}

func seedBundledWASMPlugins(destDir string) error {
	if err := seedEmbeddedWASMPlugins(destDir); err != nil {
		return err
	}
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	return seedBundledWASMPluginsFromExecutable(exePath, destDir)
}

func seedEmbeddedWASMPlugins(destDir string) error {
	sub, err := fs.Sub(bundledPluginsFS, "bundled/plugins")
	if err != nil {
		return fmt.Errorf("embedded plugins: %w", err)
	}
	if !hasBundledContent(sub) {
		return nil
	}
	return copyBundledWASMPlugins(sub, destDir)
}

// seedBundledWASMPluginsFromExecutable copies from a plugins folder next to
// the exe. Kept alongside the embedded seeding for dev builds and for release
// folders from the pre-portable zip layout; it runs second so a sidecar
// folder wins over the embedded copies.
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
	return copyBundledWASMPlugins(os.DirFS(srcDir), destDir)
}

// hasBundledContent reports whether src holds anything seeding would copy,
// so an empty (placeholder-only) source doesn't create destDir.
func hasBundledContent(src fs.FS) bool {
	entries, err := fs.ReadDir(src, ".")
	if err != nil {
		// Don't silently skip seeding on a broken/unreadable source: report
		// "has content" so copyBundledWASMPlugins runs and surfaces the real
		// error instead of a no-op that hides it.
		return true
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if isBundledPublicPluginID(strings.ToLower(entry.Name())) {
				return true
			}
			continue
		}
		if _, ok := bundledPublicPluginIDFromWASM(entry.Name()); ok {
			return true
		}
	}
	return false
}

func copyBundledWASMPlugins(src fs.FS, destDir string) error {
	entries, err := fs.ReadDir(src, ".")
	if err != nil {
		return fmt.Errorf("read bundled plugins: %w", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create plugin dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			pluginID := strings.ToLower(entry.Name())
			if !isBundledPublicPluginID(pluginID) {
				continue
			}
			if err := copyPluginAssetDir(src, entry.Name(), filepath.Join(destDir, pluginID)); err != nil {
				return err
			}
			continue
		}
		pluginID, ok := bundledPublicPluginIDFromWASM(entry.Name())
		if !ok {
			continue
		}
		if changed, err := copyFileIfChanged(src, entry.Name(), filepath.Join(destDir, pluginID+".wasm")); err != nil {
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

func copyPluginAssetDir(src fs.FS, srcDir, destDir string) error {
	return fs.WalkDir(src, srcDir, func(srcPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, filepath.FromSlash(srcPath))
		if err != nil {
			return err
		}
		destPath := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		if changed, err := copyFileIfChanged(src, srcPath, destPath); err != nil {
			return err
		} else if changed {
			log.Printf("[core] installed bundled plugin asset: %s", filepath.Join(filepath.Base(destDir), rel))
		}
		return nil
	})
}

func copyFileIfChanged(srcFS fs.FS, srcPath, destPath string) (bool, error) {
	src, err := fs.ReadFile(srcFS, srcPath)
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
