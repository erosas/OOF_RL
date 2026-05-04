package mmr

// OrFallback returns s if non-empty, otherwise fallback.
func OrFallback(s, fallback string) string {
	if s != "" {
		return s
	}
	return fallback
}

// IsAllAsterisks reports whether s is non-empty and consists entirely of '*'
// characters — the marker RL uses for masked/private Switch identities.
func IsAllAsterisks(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c != '*' {
			return false
		}
	}
	return true
}