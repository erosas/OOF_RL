package sdk_test

import (
	"encoding/json"
	"testing"

	sdk "github.com/erosas/oof-plugin-sdk"
)

func TestPluginMetaRoundTrip(t *testing.T) {
	meta := sdk.PluginMeta{
		ID:     "live",
		NavTab: sdk.NavTabMeta{ID: "live", Label: "Live", Order: 10},
		Routes: []string{"/api/live/state"},
		Events: []string{"state.updated", "match.destroyed"},
	}

	b, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got sdk.PluginMeta
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.ID != meta.ID {
		t.Errorf("ID: got %q, want %q", got.ID, meta.ID)
	}
	if len(got.Routes) != 1 || got.Routes[0] != "/api/live/state" {
		t.Errorf("Routes: got %v", got.Routes)
	}
	if len(got.Events) != 2 {
		t.Errorf("Events: got %v", got.Events)
	}
}

func TestHTTPRequestRoundTrip(t *testing.T) {
	req := sdk.HTTPRequest{Method: "GET", Path: "/api/live/state"}
	b, _ := json.Marshal(req)

	var got sdk.HTTPRequest
	json.Unmarshal(b, &got)

	if got.Method != "GET" || got.Path != "/api/live/state" {
		t.Errorf("got %+v", got)
	}
}

func TestHTTPResponseRoundTrip(t *testing.T) {
	resp := sdk.HTTPResponse{
		Status:  200,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    `{"active":false}`,
	}
	b, _ := json.Marshal(resp)

	var got sdk.HTTPResponse
	json.Unmarshal(b, &got)

	if got.Status != 200 || got.Body != `{"active":false}` {
		t.Errorf("got %+v", got)
	}
}