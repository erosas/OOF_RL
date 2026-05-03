package mmr

// NewPlayerIdentity constructs a PlayerIdentity from the raw fields the RL API gives us.
// platform is the raw string from RL events ("Steam", "Epic", "PS4", "XboxOne", "Switch").
// primaryID is the platform-specific account id (SteamID64 for Steam, else display name).
// displayName is always the visible in-game name.
func NewPlayerIdentity(platform, primaryID, displayName string) PlayerIdentity {
	if displayName == "" {
		displayName = primaryID
	}
	return PlayerIdentity{
		PrimaryID:   primaryID,
		DisplayName: displayName,
		Platform:    normalizePlatform(platform),
	}
}

func normalizePlatform(raw string) Platform {
	switch raw {
	case "Steam":
		return PlatformSteam
	case "Epic":
		return PlatformEpic
	case "PS4", "PS5", "PSN":
		return PlatformPSN
	case "XboxOne", "Xbox":
		return PlatformXbox
	case "Switch":
		return PlatformSwitch
	default:
		return Platform(raw)
	}
}