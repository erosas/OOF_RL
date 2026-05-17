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
	return fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><title>Momentum Overlay Preview</title><style>
:root {
	color-scheme: dark;
	background: transparent;
}
html,
body {
	margin: 0;
	width: 100%%;
	min-height: 100%%;
	background: #06090f;
}
body {
	display: grid;
	place-items: center;
}
svg {
	width: min(72vmin, 420px);
	height: min(72vmin, 420px);
	overflow: visible;
	font-family: "Segoe UI", Arial, sans-serif;
}
#momentum-overlay-root {
	display: grid;
	place-items: center;
}
.overlayhud-background-disc {
	fill: rgba(5, 10, 18, 0.78);
	stroke: rgba(170, 195, 225, 0.18);
	stroke-width: 2;
}
.overlayhud-confidence-track,
.overlayhud-momentum-track,
.overlayhud-volatility-track {
	fill: none;
	stroke: rgba(190, 210, 240, 0.14);
	stroke-linecap: round;
}
.overlayhud-confidence-track {
	stroke-width: 6;
}
.overlayhud-momentum-track {
	stroke-width: 18;
}
.overlayhud-volatility-track {
	stroke-width: 8;
}
.overlayhud-confidence,
.overlayhud-arc-blue,
.overlayhud-arc-orange,
.overlayhud-volatility-segment {
	fill: none;
	stroke-linecap: round;
	transition: opacity 160ms ease, stroke 160ms ease;
}
.overlayhud-confidence {
	stroke: #e8f0ff;
	stroke-width: 6;
	opacity: 0.75;
}
.overlayhud-confidence.is-empty {
	opacity: 0;
}
.overlayhud-confidence.is-low {
	stroke: #8aa1b8;
}
.overlayhud-confidence.is-medium {
	stroke: #d6dfed;
}
.overlayhud-confidence.is-high {
	stroke: #f5fbff;
}
.overlayhud-arc-blue {
	stroke: #3aa8ff;
	stroke-width: 18;
	filter: drop-shadow(0 0 8px rgba(58, 168, 255, 0.45));
}
.overlayhud-arc-orange {
	stroke: #ff9a3a;
	stroke-width: 18;
	filter: drop-shadow(0 0 8px rgba(255, 154, 58, 0.42));
}
.overlayhud-volatility-segment {
	stroke: rgba(210, 225, 245, 0.18);
	stroke-width: 7;
}
.overlayhud-volatility-segment.is-active {
	stroke: #f4d35e;
	opacity: 0.85;
}
.mcw-frame-base {
	fill: none;
	stroke: rgba(14, 22, 34, 0.96);
	stroke-width: 20;
}
.mcw-frame-highlight {
	fill: none;
	stroke: rgba(195, 215, 240, 0.18);
	stroke-width: 4;
}
.mcw-frame-shadow {
	fill: none;
	stroke: rgba(0, 0, 0, 0.38);
	stroke-width: 8;
}
.mcw-segment-inactive {
	fill: rgba(105, 125, 150, 0.18);
}
.mcw-segment-blue {
	fill: #3aa8ff;
}
.mcw-segment-orange {
	fill: #ff9a3a;
}
.mcw-segment-cap {
	fill: rgba(245, 250, 255, 0.82);
}
.mcw-segment-bevel {
	fill: rgba(255, 255, 255, 0.10);
}
.mcw-segment-inner-shadow,
.mcw-segment-outer-highlight {
	fill: none;
}
.mcw-segment-inner-shadow {
	stroke: rgba(0, 0, 0, 0.40);
	stroke-width: 8;
}
.mcw-segment-outer-highlight {
	stroke: rgba(230, 240, 255, 0.15);
	stroke-width: 5;
}
.mcw-tick {
	stroke-width: 1.8;
	stroke-linecap: round;
}
.mcw-tick-blue {
	stroke: rgba(58, 168, 255, 0.52);
}
.mcw-tick-orange {
	stroke: rgba(255, 154, 58, 0.50);
}
.mcw-tick-neutral {
	stroke: rgba(190, 210, 230, 0.24);
}
.mcw-crosshair-line {
	stroke: rgba(210, 225, 245, 0.16);
	stroke-width: 1.2;
}
.overlayhud-center-core {
	fill: rgba(8, 14, 24, 0.92);
	stroke: rgba(220, 235, 255, 0.20);
	stroke-width: 4;
}
.overlayhud-timer-text {
	fill: #f7fbff;
	font-size: 108px;
	font-weight: 700;
	letter-spacing: 0;
}
.overlayhud-state-label,
.overlayhud-status-text {
	fill: #cbd8e8;
	font-size: 38px;
	font-weight: 700;
	letter-spacing: 0;
}
.overlayhud-status-text {
	fill: #f4d35e;
}
.mcw-state-blue-pressure .overlayhud-state-label,
.mcw-state-blue-control .overlayhud-state-label {
	fill: #3aa8ff;
}
.mcw-state-orange-pressure .overlayhud-state-label,
.mcw-state-orange-control .overlayhud-state-label {
	fill: #ff9a3a;
}
.mcw-state-volatile .overlayhud-state-label {
	fill: #f0b8ff;
}
.is-inactive .overlayhud-arc-blue,
.is-inactive .overlayhud-arc-orange,
.is-inactive .mcw-segment,
.has-no-data .overlayhud-volatility-segment,
.has-no-data .overlayhud-confidence,
.has-no-data .mcw-segment {
	opacity: 0.28;
	filter: none;
}
</style></head><body><main id="momentum-overlay-root">%s</main><script>
(() => {
	const root = document.getElementById("momentum-overlay-root");
	const source = %q;
	const intervalMs = 250;
	async function refresh() {
		try {
			const response = await fetch(source, { cache: "no-store" });
			if (!response.ok) {
				return;
			}
			const svg = await response.text();
			if (svg && root.innerHTML !== svg) {
				root.innerHTML = svg;
			}
		} catch (_) {
		}
	}
	setInterval(refresh, intervalMs);
})();
</script></body></html>`, svg, previewSVGRoutePath)
}
