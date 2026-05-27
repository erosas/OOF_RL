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
	--mcw-blue-core: #38b2f6;
	--mcw-blue-edge: #76d8ff;
	--mcw-blue-glow: #00c8ff;
	--mcw-orange-core: #f97316;
	--mcw-orange-edge: #ffc06a;
	--mcw-orange-glow: #ff8a2a;
	--mcw-purple: #b45cff;
	--mcw-purple-hot: #f0b8ff;
	--mcw-seam: #f5f7fb;
	--mcw-frame: #111827;
	--mcw-frame-edge: #5e7086;
	--mcw-response-transition-ms: 260ms;
	--mcw-response-effect-transition-ms: 320ms;
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
.overlayhud-background-disc {
	filter: drop-shadow(0 16px 28px rgba(0, 0, 0, 0.48));
}
.mcw-aura {
	fill: none;
	stroke-linecap: round;
	stroke-width: 28;
	filter: url(#mcw-soft-blur);
	transition: opacity var(--mcw-response-effect-transition-ms) ease-out;
}
.mcw-aura-blue {
	stroke: var(--mcw-blue-glow);
	opacity: calc(var(--mcw-blue-aura-opacity) * var(--mcw-state-intensity));
}
.mcw-aura-orange {
	stroke: var(--mcw-orange-glow);
	opacity: calc(var(--mcw-orange-aura-opacity) * var(--mcw-state-intensity));
}
.mcw-aura-contest {
	fill: var(--mcw-purple-hot);
	stroke: none;
	opacity: calc(var(--mcw-contest-aura-opacity) * var(--mcw-seam-intensity));
}
.mcw-streaks {
	opacity: calc(0.16 + var(--mcw-state-intensity) * 0.34);
}
.mcw-streaks line {
	stroke-width: 6;
	stroke-linecap: round;
	opacity: 0.42;
	animation: mcw-energy-streak 1040ms ease-in-out infinite;
}
.mcw-streaks-blue line {
	stroke: var(--mcw-blue-glow);
}
.mcw-streaks-orange line {
	stroke: var(--mcw-orange-glow);
}
.mcw-spark {
	display: none;
	opacity: 0;
}
.mcw-state-blue-pressure #outer-sparks-blue .mcw-spark-role-pressure,
.mcw-state-orange-pressure #outer-sparks-orange .mcw-spark-role-pressure,
.mcw-state-blue-control #outer-sparks-blue .mcw-spark-role-control,
.mcw-state-orange-control #outer-sparks-orange .mcw-spark-role-control,
.mcw-state-volatile #outer-sparks-purple .mcw-spark-role-volatile,
.mcw-state-volatile #outer-sparks-white .mcw-spark-role-volatile {
	display: block;
	opacity: calc(0.32 + var(--mcw-volatility) * 0.42);
	animation: mcw-spark-fly 900ms cubic-bezier(0.2, 0.85, 0.28, 1) infinite;
}
#outer-sparks-blue .mcw-spark {
	fill: var(--mcw-blue-glow);
}
#outer-sparks-orange .mcw-spark {
	fill: var(--mcw-orange-glow);
}
#outer-sparks-purple .mcw-spark,
#outer-sparks-white .mcw-spark {
	fill: var(--mcw-purple-hot);
}
.mcw-frame-base {
	fill: none;
	stroke: var(--mcw-frame);
	stroke-width: 20;
}
.mcw-frame-highlight {
	fill: none;
	stroke: color-mix(in srgb, var(--mcw-frame-edge) 70%%, #f5f7fb);
	stroke-width: 4;
}
.mcw-frame-shadow {
	fill: none;
	stroke: rgba(0, 0, 0, 0.38);
	stroke-width: 8;
}
.mcw-segment-inactive {
	fill: rgba(105, 125, 150, 0.20);
	opacity: 0.58;
}
.mcw-segment-blue {
	fill: var(--mcw-blue-core);
	stroke: var(--mcw-blue-edge);
	stroke-width: 1.4;
	filter: drop-shadow(0 0 calc(3px + var(--mcw-blue-pressure) * var(--mcw-state-intensity) * 12px) var(--mcw-blue-glow));
	transition: opacity var(--mcw-response-transition-ms) ease-out, filter var(--mcw-response-effect-transition-ms) ease-out;
}
.mcw-segment-orange {
	fill: var(--mcw-orange-core);
	stroke: var(--mcw-orange-edge);
	stroke-width: 1.4;
	filter: drop-shadow(0 0 calc(3px + var(--mcw-orange-pressure) * var(--mcw-state-intensity) * 12px) var(--mcw-orange-glow));
	transition: opacity var(--mcw-response-transition-ms) ease-out, filter var(--mcw-response-effect-transition-ms) ease-out;
}
.mcw-segment-cap {
	fill: var(--mcw-seam);
	stroke: rgba(245, 247, 251, 0.9);
	stroke-width: 1.6;
	filter:
		drop-shadow(0 0 calc(3px + var(--mcw-seam-intensity) * 10px) var(--mcw-seam))
		drop-shadow(0 0 calc(1px + var(--mcw-volatility) * 12px) var(--mcw-purple-hot));
	opacity: calc(0.36 + var(--mcw-seam-intensity) * 0.34);
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
	stroke: color-mix(in srgb, var(--mcw-blue-edge) 72%%, transparent);
}
.mcw-tick-orange {
	stroke: color-mix(in srgb, var(--mcw-orange-edge) 72%%, transparent);
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
	filter: drop-shadow(0 8px 18px rgba(0, 0, 0, 0.48));
}
.mcw-center-inner-shadow,
.mcw-center-rim {
	fill: none;
	stroke: rgba(210, 225, 245, 0.22);
	stroke-width: 3;
}
.mcw-center-glass-highlight {
	fill: rgba(255, 255, 255, 0.08);
}
.mcw-center-wash-blue {
	fill: url(#mcw-center-blue-wash);
	opacity: var(--mcw-center-blue-wash);
}
.mcw-center-wash-orange {
	fill: url(#mcw-center-orange-wash);
	opacity: var(--mcw-center-orange-wash);
}
.mcw-center-wash-purple {
	fill: url(#mcw-center-purple-wash);
	opacity: calc(var(--mcw-center-purple-wash) + var(--mcw-volatility) * 0.08);
}
#contest-top-core {
	fill: var(--mcw-seam);
	opacity: calc(0.42 + var(--mcw-seam-intensity) * 0.44);
}
#contest-top-purple-glow {
	fill: var(--mcw-purple-hot);
	filter: url(#mcw-soft-blur);
	opacity: calc(var(--mcw-contest-aura-opacity) + var(--mcw-volatility) * 0.24);
}
#contest-top-vertical-beam {
	stroke: var(--mcw-purple-hot);
	stroke-width: 3;
	stroke-linecap: round;
	opacity: calc(0.10 + var(--mcw-seam-intensity) * 0.28);
}
#contest-bottom-seam {
	fill: var(--mcw-seam);
	opacity: calc(0.10 + var(--mcw-seam-intensity) * 0.24);
}
.overlayhud-timer-text {
	fill: #f7fbff;
	font-size: 108px;
	font-weight: 900;
	letter-spacing: 2px;
	paint-order: stroke;
	stroke: rgba(5, 10, 18, 0.70);
	stroke-width: 6px;
	filter: drop-shadow(0 0 9px rgba(245, 247, 251, 0.28));
}
.overlayhud-state-label,
.overlayhud-status-text {
	fill: #cbd8e8;
	font-size: 38px;
	font-weight: 900;
	letter-spacing: 4px;
	paint-order: stroke;
	stroke: rgba(5, 10, 18, 0.70);
	stroke-width: 4px;
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
.mcw-state-blue-pressure .mcw-segment-blue,
.mcw-state-blue-control .mcw-segment-blue {
	opacity: 0.98;
}
.mcw-state-blue-pressure .mcw-segment-orange,
.mcw-state-blue-control .mcw-segment-orange {
	opacity: 0.70;
}
.mcw-state-orange-pressure .mcw-segment-orange,
.mcw-state-orange-control .mcw-segment-orange {
	opacity: 0.98;
}
.mcw-state-orange-pressure .mcw-segment-blue,
.mcw-state-orange-control .mcw-segment-blue {
	opacity: 0.70;
}
.mcw-state-volatile .mcw-segment-cap,
.mcw-state-volatile #contest-top-core,
.mcw-state-volatile #contest-top-purple-glow,
.mcw-state-volatile #contest-top-vertical-beam {
	animation: mcw-contest-flicker 760ms ease-in-out infinite alternate;
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
@keyframes mcw-contest-flicker {
	0%% { opacity: calc(0.24 + var(--mcw-seam-intensity) * 0.16); }
	45%% { opacity: calc(0.44 + var(--mcw-seam-intensity) * 0.30); }
	100%% { opacity: calc(0.30 + var(--mcw-seam-intensity) * 0.24); }
}
@keyframes mcw-energy-streak {
	0%%, 100%% { opacity: 0.14; transform: translate(0, 0) scale(0.96); }
	45%% { opacity: 0.46; transform: translate(0, -2px) scale(1.08); }
}
@keyframes mcw-spark-fly {
	0%% { opacity: 0; transform: translate(0, 0) scale(0.86); }
	18%% { opacity: 0.92; transform: translate(0, -7px) scale(1); }
	72%% { opacity: 0.48; transform: translate(0, -18px) scale(0.84); }
	100%% { opacity: 0; transform: translate(0, -26px) scale(0.62); }
}
@media (prefers-reduced-motion: reduce) {
	.momentum-control-wheel-svg *,
	.momentum-control-wheel-svg .mcw-segment,
	.momentum-control-wheel-svg .mcw-aura {
		animation: none !important;
		transition: none !important;
	}
	.momentum-control-wheel-svg #outer-sparks,
	.momentum-control-wheel-svg #outer-energy-streaks {
		display: none !important;
	}
	.momentum-control-wheel-svg [filter],
	.momentum-control-wheel-svg .mcw-segment,
	.momentum-control-wheel-svg .mcw-aura,
	.momentum-control-wheel-svg .overlayhud-timer-text {
		filter: none !important;
	}
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
