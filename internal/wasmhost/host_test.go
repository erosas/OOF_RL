package wasmhost_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"OOF_RL/internal/wasmhost"
)

// emptyWasm is the smallest valid WASM binary: magic number + version only.
// It has no imports, exports, or sections, so it loads successfully but
// exports nothing — useful for testing error paths after instantiation.
var emptyWasm = []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}

func writeTempWasm(t *testing.T, data []byte) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "test.wasm")
	if err := os.WriteFile(f, data, 0644); err != nil {
		t.Fatalf("write temp wasm: %v", err)
	}
	return f
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := wasmhost.Load("nonexistent_plugin.wasm")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_InvalidWasmBytes(t *testing.T) {
	path := writeTempWasm(t, []byte("this is not a wasm binary"))
	_, err := wasmhost.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid WASM bytes")
	}
}

func TestLoad_MissingPluginMetadataExport(t *testing.T) {
	path := writeTempWasm(t, emptyWasm)
	_, err := wasmhost.Load(path)
	if err == nil {
		t.Fatal("expected error: empty WASM has no plugin_metadata export")
	}
	if !strings.Contains(err.Error(), "plugin_metadata") {
		t.Errorf("error should name the missing export, got: %v", err)
	}
}