//go:build !wasip1

package main

import (
	"encoding/json"
	"strings"
	"testing"

	sdk "github.com/erosas/oof-plugin-sdk"
)

// --- validateLayout ---

func TestValidateLayoutEmpty(t *testing.T) {
	items, err := validateLayout(json.RawMessage(`[]`))
	if err != nil {
		t.Fatalf("empty layout: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestValidateLayoutValid(t *testing.T) {
	raw := `[{"id":"widget-1","x":0,"y":0,"w":4,"h":3}]`
	items, err := validateLayout(json.RawMessage(raw))
	if err != nil {
		t.Fatalf("valid layout: %v", err)
	}
	if len(items) != 1 || items[0].ID != "widget-1" {
		t.Errorf("unexpected items: %+v", items)
	}
}

func TestValidateLayoutNotArray(t *testing.T) {
	_, err := validateLayout(json.RawMessage(`{"id":"x"}`))
	if err == nil {
		t.Fatal("expected error for non-array")
	}
}

func TestValidateLayoutMissingID(t *testing.T) {
	_, err := validateLayout(json.RawMessage(`[{"x":0,"y":0,"w":1,"h":1}]`))
	if err == nil || !strings.Contains(err.Error(), "missing id") {
		t.Fatalf("expected missing id error, got: %v", err)
	}
}

func TestValidateLayoutNegativeXY(t *testing.T) {
	_, err := validateLayout(json.RawMessage(`[{"id":"a","x":-1,"y":0,"w":1,"h":1}]`))
	if err == nil {
		t.Fatal("expected error for negative x")
	}
}

func TestValidateLayoutZeroWH(t *testing.T) {
	_, err := validateLayout(json.RawMessage(`[{"id":"a","x":0,"y":0,"w":0,"h":1}]`))
	if err == nil {
		t.Fatal("expected error for w=0")
	}
}

func TestValidateLayoutExceedsColumns(t *testing.T) {
	_, err := validateLayout(json.RawMessage(`[{"id":"a","x":0,"y":0,"w":13,"h":1}]`))
	if err == nil {
		t.Fatal("expected error for w > maxGridColumns")
	}
}

func TestValidateLayoutXPlusWExceedsColumns(t *testing.T) {
	_, err := validateLayout(json.RawMessage(`[{"id":"a","x":10,"y":0,"w":4,"h":1}]`))
	if err == nil {
		t.Fatal("expected error for x+w > maxGridColumns")
	}
}

func TestValidateLayoutXAtMaxColumns(t *testing.T) {
	_, err := validateLayout(json.RawMessage(`[{"id":"a","x":12,"y":0,"w":1,"h":1}]`))
	if err == nil {
		t.Fatal("expected error for x >= maxGridColumns")
	}
}

func TestValidateLayoutTooManyItems(t *testing.T) {
	items := make([]map[string]any, maxLayoutItems+1)
	for i := range items {
		items[i] = map[string]any{"id": "a", "x": 0, "y": i, "w": 1, "h": 1}
	}
	raw, _ := json.Marshal(items)
	_, err := validateLayout(json.RawMessage(raw))
	if err == nil {
		t.Fatal("expected error for too many items")
	}
}

// --- handleLayout routing ---

func TestHandleLayoutGetEmpty(t *testing.T) {
	resp := handleHTTP(sdk.HTTPRequest{Method: "GET", Path: "/api/dashboard/layout"})
	if resp.Status != 200 {
		t.Fatalf("status: got %d, want 200", resp.Status)
	}
}

func TestHandleLayoutPostBadJSON(t *testing.T) {
	resp := handleHTTP(sdk.HTTPRequest{
		Method: "POST",
		Path:   "/api/dashboard/layout",
		Body:   `not json`,
	})
	if resp.Status != 400 {
		t.Fatalf("status: got %d, want 400", resp.Status)
	}
}

func TestHandleLayoutPostInvalidLayout(t *testing.T) {
	resp := handleHTTP(sdk.HTTPRequest{
		Method: "POST",
		Path:   "/api/dashboard/layout",
		Body:   `[{"id":"","x":0,"y":0,"w":1,"h":1}]`,
	})
	if resp.Status != 400 {
		t.Fatalf("status: got %d, want 400", resp.Status)
	}
}

func TestHandleLayoutBadMethod(t *testing.T) {
	resp := handleHTTP(sdk.HTTPRequest{Method: "DELETE", Path: "/api/dashboard/layout"})
	if resp.Status != 405 {
		t.Fatalf("status: got %d, want 405", resp.Status)
	}
}

func TestHandleUnknownRoute(t *testing.T) {
	resp := handleHTTP(sdk.HTTPRequest{Method: "GET", Path: "/api/dashboard/unknown"})
	if resp.Status != 404 {
		t.Fatalf("status: got %d, want 404", resp.Status)
	}
}
