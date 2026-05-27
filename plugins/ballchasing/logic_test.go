//go:build !wasip1

package main

import (
	"testing"
	"time"
)

func TestNormalizeGUID(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"abc123", "ABC123"},
		{"A1B2-C3D4-E5F6", "A1B2C3D4E5F6"},
		{"a1b2-c3d4-e5f6", "A1B2C3D4E5F6"},
		{"ALREADY", "ALREADY"},
	}
	for _, c := range cases {
		if got := normalizeGUID(c.in); got != c.want {
			t.Errorf("normalizeGUID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestApplySettings(t *testing.T) {
	apiKey = ""
	deleteAfterUpload = false

	applySettings([]byte(`{"ballchasing_api_key":"testkey","ballchasing_delete_after_upload":"true"}`))
	if apiKey != "testkey" {
		t.Errorf("apiKey = %q, want %q", apiKey, "testkey")
	}
	if !deleteAfterUpload {
		t.Error("deleteAfterUpload should be true")
	}

	applySettings([]byte(`{"ballchasing_delete_after_upload":"false"}`))
	if deleteAfterUpload {
		t.Error("deleteAfterUpload should be false after explicit false")
	}

	// "on" and "1" are also truthy
	applySettings([]byte(`{"ballchasing_delete_after_upload":"1"}`))
	if !deleteAfterUpload {
		t.Error("deleteAfterUpload should be true for value '1'")
	}
	applySettings([]byte(`{"ballchasing_delete_after_upload":"on"}`))
	if !deleteAfterUpload {
		t.Error("deleteAfterUpload should be true for value 'on'")
	}
}

func TestApplySettingsBadJSON(t *testing.T) {
	if code := applySettings([]byte(`not json`)); code != 1 {
		t.Errorf("expected return code 1 for bad JSON, got %d", code)
	}
}

func TestMatchReplayFiles(t *testing.T) {
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	matches := []matchInfo{
		{MatchGUID: "AAA", StartedAt: base},
		{MatchGUID: "BBB", StartedAt: base.Add(40 * time.Minute)},
	}

	t.Run("file within window maps to match", func(t *testing.T) {
		files := []replayFileEntry{
			{name: "aaa.replay", modTime: base.Add(10 * time.Minute)},
		}
		got := matchReplayFiles(files, matches)
		if got[0] != "aaa.replay" {
			t.Errorf("expected aaa.replay for match 0, got %q", got[0])
		}
		if got[1] != "" {
			t.Errorf("expected no file for match 1, got %q", got[1])
		}
	})

	t.Run("file outside window matches nothing", func(t *testing.T) {
		files := []replayFileEntry{
			// 35 minutes after match 0 start — outside the 30-minute window
			{name: "late.replay", modTime: base.Add(35 * time.Minute)},
		}
		got := matchReplayFiles(files, matches)
		// late.replay is 35m after match 0 (outside window) but only 5m before
		// match 1 ends its window — actually it's BEFORE match 1 starts,
		// so it should match nothing.
		if got[0] != "" || got[1] != "" {
			t.Errorf("expected no matches, got %v", got)
		}
	})

	t.Run("file before match start matches nothing", func(t *testing.T) {
		files := []replayFileEntry{
			{name: "early.replay", modTime: base.Add(-5 * time.Minute)},
		}
		got := matchReplayFiles(files, matches)
		if got[0] != "" || got[1] != "" {
			t.Errorf("expected no matches, got %v", got)
		}
	})

	t.Run("each file matches at most one match", func(t *testing.T) {
		files := []replayFileEntry{
			{name: "r1.replay", modTime: base.Add(5 * time.Minute)},
			{name: "r2.replay", modTime: base.Add(45 * time.Minute)},
		}
		got := matchReplayFiles(files, matches)
		if got[0] != "r1.replay" {
			t.Errorf("expected r1.replay for match 0, got %q", got[0])
		}
		if got[1] != "r2.replay" {
			t.Errorf("expected r2.replay for match 1, got %q", got[1])
		}
	})

	t.Run("no files produces empty result", func(t *testing.T) {
		got := matchReplayFiles(nil, matches)
		if len(got) != 0 {
			t.Errorf("expected empty result, got %v", got)
		}
	})
}