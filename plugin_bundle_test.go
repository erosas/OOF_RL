package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyBundledWASMPluginsCopiesWASMAndAssets(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dest := filepath.Join(root, "dest")
	mustWrite(t, filepath.Join(src, "live.wasm"), "wasm")
	mustWrite(t, filepath.Join(src, "live", "view.html"), "<main>live</main>")
	mustWrite(t, filepath.Join(src, "ballchasing.wasm"), "not bundled")
	mustWrite(t, filepath.Join(src, "ballchasing", "view.html"), "<main>ballchasing</main>")
	mustWrite(t, filepath.Join(src, "notes.txt"), "not copied")

	if err := copyBundledWASMPlugins(os.DirFS(src), dest); err != nil {
		t.Fatalf("copyBundledWASMPlugins: %v", err)
	}

	assertFile(t, filepath.Join(dest, "live.wasm"), "wasm")
	assertFile(t, filepath.Join(dest, "live", "view.html"), "<main>live</main>")
	if _, err := os.Stat(filepath.Join(dest, "ballchasing.wasm")); !os.IsNotExist(err) {
		t.Fatalf("ballchasing.wasm copied unexpectedly, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "ballchasing")); !os.IsNotExist(err) {
		t.Fatalf("ballchasing assets copied unexpectedly, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "notes.txt")); !os.IsNotExist(err) {
		t.Fatalf("notes.txt copied unexpectedly, stat err=%v", err)
	}
}

func TestCopyBundledWASMPluginsUpdatesChangedFiles(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dest := filepath.Join(root, "dest")
	mustWrite(t, filepath.Join(src, "ranks.wasm"), "new")
	mustWrite(t, filepath.Join(dest, "ranks.wasm"), "old")

	if err := copyBundledWASMPlugins(os.DirFS(src), dest); err != nil {
		t.Fatalf("copyBundledWASMPlugins: %v", err)
	}

	assertFile(t, filepath.Join(dest, "ranks.wasm"), "new")
}

func TestCopyBundledWASMPluginsPreservesCustomAppDataPlugins(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dest := filepath.Join(root, "dest")
	mustWrite(t, filepath.Join(src, "live.wasm"), "bundled live")
	mustWrite(t, filepath.Join(src, "live", "view.html"), "bundled asset")
	mustWrite(t, filepath.Join(src, "custom.wasm"), "bundled custom ignored")
	mustWrite(t, filepath.Join(src, "custom", "view.html"), "bundled custom asset ignored")
	mustWrite(t, filepath.Join(dest, "live.wasm"), "old live")
	mustWrite(t, filepath.Join(dest, "custom.wasm"), "custom app-data plugin")
	mustWrite(t, filepath.Join(dest, "custom", "view.html"), "custom app-data asset")
	mustWrite(t, filepath.Join(dest, "notes.txt"), "unrelated")

	if err := copyBundledWASMPlugins(os.DirFS(src), dest); err != nil {
		t.Fatalf("copyBundledWASMPlugins: %v", err)
	}

	assertFile(t, filepath.Join(dest, "live.wasm"), "bundled live")
	assertFile(t, filepath.Join(dest, "live", "view.html"), "bundled asset")
	assertFile(t, filepath.Join(dest, "custom.wasm"), "custom app-data plugin")
	assertFile(t, filepath.Join(dest, "custom", "view.html"), "custom app-data asset")
	assertFile(t, filepath.Join(dest, "notes.txt"), "unrelated")
}

func TestCopyBundledWASMPluginsPreservesUnrelatedFilesInPublicPluginDirs(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dest := filepath.Join(root, "dest")
	mustWrite(t, filepath.Join(src, "session", "view.html"), "new session")
	mustWrite(t, filepath.Join(dest, "session", "dev-only.txt"), "keep me")

	if err := copyBundledWASMPlugins(os.DirFS(src), dest); err != nil {
		t.Fatalf("copyBundledWASMPlugins: %v", err)
	}

	assertFile(t, filepath.Join(dest, "session", "view.html"), "new session")
	assertFile(t, filepath.Join(dest, "session", "dev-only.txt"), "keep me")
}

func TestSeedBundledWASMPluginsNoSidecarNoops(t *testing.T) {
	root := t.TempDir()
	exePath := filepath.Join(root, "OOF_RL", "oof_rl.exe")
	dest := filepath.Join(root, "appdata", "plugins")

	if err := seedBundledWASMPluginsFromExecutable(exePath, dest); err != nil {
		t.Fatalf("seedBundledWASMPluginsFromExecutable: %v", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("dest dir should not be created without sidecar plugins, stat err=%v", err)
	}
}

func TestSeedEmbeddedWASMPluginsPlaceholderOnlyNoops(t *testing.T) {
	// Dev builds embed only the .gitkeep placeholder; seeding must not
	// create the destination directory for it. A local release build can
	// leave real plugins staged under bundled/plugins, in which case the
	// embed legitimately has content and this invariant no longer applies.
	sub, err := fs.Sub(bundledPluginsFS, "bundled/plugins")
	if err != nil {
		t.Fatalf("sub embedded plugins: %v", err)
	}
	if hasBundledContent(sub) {
		t.Skip("bundled/plugins has real embedded plugins; placeholder-only invariant not testable here")
	}
	dest := filepath.Join(t.TempDir(), "plugins")
	if err := seedEmbeddedWASMPlugins(dest); err != nil {
		t.Fatalf("seedEmbeddedWASMPlugins: %v", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("dest dir should not be created from placeholder-only embed, stat err=%v", err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("%s: got %q, want %q", path, got, want)
	}
}
