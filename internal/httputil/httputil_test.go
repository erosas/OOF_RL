package httputil_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"OOF_RL/internal/httputil"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	httputil.WriteJSON(w, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}
	var got map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["key"] != "value" {
		t.Errorf("key: got %q, want value", got["key"])
	}
}

func TestWriteJSONEncodesSlice(t *testing.T) {
	w := httptest.NewRecorder()
	httputil.WriteJSON(w, []int{1, 2, 3})

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}
	var got []int
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 3 || got[1] != 2 {
		t.Errorf("slice: got %v", got)
	}
}

func TestJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	httputil.JSONError(w, http.StatusBadRequest, "bad input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}
	var got map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["error"] != "bad input" {
		t.Errorf("error: got %q, want bad input", got["error"])
	}
}

func TestJSONErrorStatusCodes(t *testing.T) {
	cases := []struct {
		code int
		msg  string
	}{
		{http.StatusNotFound, "not found"},
		{http.StatusInternalServerError, "oops"},
		{http.StatusMethodNotAllowed, "method not allowed"},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		httputil.JSONError(w, c.code, c.msg)
		if w.Code != c.code {
			t.Errorf("code %d: status got %d", c.code, w.Code)
		}
		var got map[string]string
		json.Unmarshal(w.Body.Bytes(), &got)
		if got["error"] != c.msg {
			t.Errorf("code %d: error got %q, want %q", c.code, got["error"], c.msg)
		}
	}
}