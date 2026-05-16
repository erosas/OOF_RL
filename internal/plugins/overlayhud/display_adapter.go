package overlayhud

import (
	"fmt"
	"time"

	"OOF_RL/internal/momentum"
)

// DisplayAdapter renders Momentum Overlay display output on demand.
// It owns no background work and renders only when a caller requests output.
type DisplayAdapter struct {
	momentum momentum.SnapshotProvider
}

func NewDisplayAdapter(provider momentum.SnapshotProvider) DisplayAdapter {
	return DisplayAdapter{momentum: provider}
}

func (a DisplayAdapter) RenderSVG(now time.Time) (string, bool) {
	view, ok := momentumViewModelFromProvider(a.momentum, now)
	if !ok {
		return "", false
	}
	return RenderSVG(buildRenderModel(view)), true
}

func (a DisplayAdapter) RenderHTML(now time.Time) (string, bool) {
	svg, ok := a.RenderSVG(now)
	if !ok {
		return "", false
	}
	return wrapPreviewHTML(svg), true
}

func momentumViewModelFromProvider(provider momentum.SnapshotProvider, now time.Time) (ViewModel, bool) {
	if provider == nil {
		return ViewModel{}, false
	}
	return mapMomentumViewModel(provider.Snapshot(), provider.Status(), now), true
}

func wrapPreviewHTML(svg string) string {
	return fmt.Sprintf("<!doctype html><html><head><meta charset=\"utf-8\"><title>Momentum Overlay Preview</title></head><body>%s</body></html>", svg)
}
