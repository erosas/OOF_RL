package overlayhud

import "strings"

const nativeShellTitle = "Momentum Overlay"

type NativeShellSpec struct {
	Target SurfaceTarget
	URL    string
	Title  string
}

func MomentumNativeShellSpec(baseAppURL string) (NativeShellSpec, bool) {
	target := MomentumSurfaceTarget()
	targetURL, ok := target.URL(baseAppURL)
	if !ok {
		return NativeShellSpec{}, false
	}

	return NativeShellSpec{
		Target: target,
		URL:    targetURL,
		Title:  nativeShellTitle,
	}, true
}

func (s NativeShellSpec) Valid() bool {
	return s.Target.ID == SurfaceTargetID &&
		s.Target.Route == previewRoutePath &&
		strings.TrimSpace(s.URL) != "" &&
		strings.TrimSpace(s.Title) != ""
}
