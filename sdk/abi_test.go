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
		Routes: []sdk.RouteMeta{{Path: "/api/live/state", Method: "GET"}},
		Events: []string{"state.updated", "match.destroyed"},
		Settings: []sdk.SettingSchema{{
			Key:          "autoupdate_check_url",
			Type:         "action",
			ActionPath:   "/api/autoupdate/check",
			ActionMethod: "POST",
			StatusPath:   "/api/autoupdate/status",
			DownloadPath: "/api/autoupdate/download",
		}},
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
	if len(got.Routes) != 1 || got.Routes[0].Path != "/api/live/state" || got.Routes[0].Method != "GET" {
		t.Errorf("Routes: got %v", got.Routes)
	}
	if len(got.Events) != 2 {
		t.Errorf("Events: got %v", got.Events)
	}
	if len(got.Settings) != 1 || got.Settings[0].ActionPath != "/api/autoupdate/check" || got.Settings[0].DownloadPath == "" {
		t.Errorf("Settings: got %v", got.Settings)
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

func TestParseBool(t *testing.T) {
	truthy := []string{"true", "1", "on", "TRUE", "ON", "True", "On"}
	for _, s := range truthy {
		if !sdk.ParseBool(s) {
			t.Errorf("ParseBool(%q) = false, want true", s)
		}
	}
	falsy := []string{"false", "0", "off", "", "yes", "t"}
	for _, s := range falsy {
		if sdk.ParseBool(s) {
			t.Errorf("ParseBool(%q) = true, want false", s)
		}
	}
}

func TestQueryParam(t *testing.T) {
	cases := []struct{ query, key, want string }{
		{"foo=bar&baz=qux", "foo", "bar"},
		{"foo=bar&baz=qux", "baz", "qux"},
		{"foo=bar", "missing", ""},
		{"", "foo", ""},
		{"%%%", "foo", ""},
	}
	for _, c := range cases {
		if got := sdk.QueryParam(c.query, c.key); got != c.want {
			t.Errorf("QueryParam(%q, %q) = %q, want %q", c.query, c.key, got, c.want)
		}
	}
}

func TestParseTime(t *testing.T) {
	// Valid formats
	valid := []string{
		"2024-01-15T10:30:00.123456789Z",
		"2024-01-15T10:30:00Z",
		"2024-01-15T10:30:00Z",
		"2024-01-15 10:30:05",
	}
	for _, s := range valid {
		if sdk.ParseTime(s).IsZero() {
			t.Errorf("ParseTime(%q) returned zero time", s)
		}
	}
	// Invalid input returns zero value
	if !sdk.ParseTime("").IsZero() {
		t.Error("ParseTime(\"\") should return zero time")
	}
	if !sdk.ParseTime("not-a-date").IsZero() {
		t.Error("ParseTime(\"not-a-date\") should return zero time")
	}
}
