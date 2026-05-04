package mmr

import "strings"

// Platform identifies which game store the account is tied to.
type Platform string

const (
	PlatformSteam  Platform = "steam"
	PlatformEpic   Platform = "epic"
	PlatformPSN    Platform = "psn"
	PlatformXbox   Platform = "xbox"
	PlatformSwitch Platform = "switch"
)

// PlayerIdentity carries whatever information is known about a player.
// Providers use whichever fields they support.
type PlayerIdentity struct {
	// PrimaryID is the platform-specific account identifier.
	// For Steam this is the SteamID64 decimal string (e.g. "76561198025501695").
	// For Epic/PSN/Xbox/Switch this is the display name or platform username.
	PrimaryID   string
	DisplayName string
	Platform    Platform
}

// PlaylistRank holds the MMR result for one ranked playlist.
type PlaylistRank struct {
	PlaylistID   int     // RL playlist number (10=duel, 11=doubles, 13=standard…)
	PlaylistName string  // human-readable ("Ranked Duel 1v1", …)
	MMR          float64
	Tier         int    // 0 = unranked
	TierName     string // "Gold III", "Champion I", …
	Division     int    // 0-indexed within the tier
	IconURL      string
}

// NormalizePlatform maps lowercase platform slugs from RL primary IDs or
// tracker.gg URL parameters to canonical Platform values.
func NormalizePlatform(raw string) Platform {
	switch strings.ToLower(raw) {
	case "steam":
		return PlatformSteam
	case "epic", "epicgames":
		return PlatformEpic
	case "ps4", "ps5", "psn", "playstation":
		return PlatformPSN
	case "xboxone", "xbox", "xbl":
		return PlatformXbox
	case "nintendo", "switch":
		return PlatformSwitch
	default:
		return Platform(strings.ToLower(raw))
	}
}

// Provider is the MMR lookup contract.
type Provider interface {
	// Name returns a short, user-facing identifier ("tracker.gg", "rlstats.net", …).
	Name() string

	// Supports reports whether this provider can look up the given platform.
	Supports(platform Platform) bool

	// Lookup fetches ranked MMR data for the given player.
	Lookup(id PlayerIdentity) ([]PlaylistRank, error)
}