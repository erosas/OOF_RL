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
		Platform:    NormalizePlatform(platform),
	}
}