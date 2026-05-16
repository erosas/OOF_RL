package overlayhud

import (
	"net/url"
	"strings"
)

const (
	SurfaceTargetID   = "momentum-overlay"
	SurfaceTargetName = "Momentum Overlay"
)

type SurfaceTarget struct {
	ID    string
	Name  string
	Route string
}

func MomentumSurfaceTarget() SurfaceTarget {
	return SurfaceTarget{
		ID:    SurfaceTargetID,
		Name:  SurfaceTargetName,
		Route: previewRoutePath,
	}
}

func (t SurfaceTarget) URL(baseAppURL string) (string, bool) {
	baseAppURL = strings.TrimSpace(baseAppURL)
	if baseAppURL == "" {
		return "", false
	}

	parsed, err := url.Parse(baseAppURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}

	parsed.Path = t.Route
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), true
}
