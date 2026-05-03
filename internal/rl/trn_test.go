package rl

import "testing"

func TestTRNSlug(t *testing.T) {
	cases := []struct{ in, want string }{
		{"steam", "steam"},
		{"ps4", "psn"},
		{"ps5", "psn"},
		{"playstation", "psn"},
		{"xboxone", "xbl"},
		{"xbox", "xbl"},
		{"epicgames", "epic"},
		{"nintendo", "switch"},
		{"unknown", "unknown"},
		{"", ""},
	}
	for _, c := range cases {
		got := trnSlug(c.in)
		if got != c.want {
			t.Errorf("trnSlug(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTRNProfileURL(t *testing.T) {
	cases := []struct {
		name      string
		primaryID string
		dispName  string
		want      string
	}{
		{
			"steam uses numeric ID not display name",
			"steam|76561198000000001",
			"SteamUser",
			"https://tracker.gg/rocket-league/profile/steam/76561198000000001",
		},
		{
			"steam with no display name",
			"steam|76561198000000002",
			"",
			"https://tracker.gg/rocket-league/profile/steam/76561198000000002",
		},
		{
			"epic uses display name",
			"epicgames|someepicid",
			"EpicGamer",
			"https://tracker.gg/rocket-league/profile/epic/EpicGamer",
		},
		{
			"ps4 uses display name",
			"ps4|psnid",
			"PSNPlayer",
			"https://tracker.gg/rocket-league/profile/psn/PSNPlayer",
		},
		{
			"xboxone uses display name",
			"xboxone|xboxid",
			"XboxGamer",
			"https://tracker.gg/rocket-league/profile/xbl/XboxGamer",
		},
		{
			"nintendo uses display name when not masked",
			"nintendo|realid",
			"SwitchPlayer",
			"https://tracker.gg/rocket-league/profile/switch/SwitchPlayer",
		},
		{
			"masked Switch ID returns empty",
			"nintendo|****",
			"",
			"",
		},
		{
			// RL masks both the ID and the name for Switch privacy; the ID mask
			// takes precedence so we never produce a link even if a name is passed.
			"masked Switch ID returns empty even with display name",
			"nintendo|****",
			"SomeUser",
			"",
		},
		{
			"no separator returns empty",
			"invalidid",
			"",
			"",
		},
		{
			"empty primary ID returns empty",
			"",
			"",
			"",
		},
		{
			"non-steam falls back to ID when display name empty",
			"epicgames|fallbackid",
			"",
			"https://tracker.gg/rocket-league/profile/epic/fallbackid",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := trnProfileURL(c.primaryID, c.dispName)
			if got != c.want {
				t.Errorf("trnProfileURL(%q, %q) = %q, want %q", c.primaryID, c.dispName, got, c.want)
			}
		})
	}
}

func TestSanitizePathPart(t *testing.T) {
	cases := []struct{ in, want string }{
		{"simple", "simple"},
		{"with spaces", "with_spaces"},
		{"with/slash", "with_slash"},
		{`with\backslash`, "with_backslash"},
		{"with:colon", "with_colon"},
		{"with|pipe", "with_pipe"},
		{"", ""},
		{"  spaces only  ", "spaces_only"},
		{"guid-abc:123", "guid-abc_123"},
	}
	for _, c := range cases {
		got := sanitizePathPart(c.in)
		if got != c.want {
			t.Errorf("sanitizePathPart(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}