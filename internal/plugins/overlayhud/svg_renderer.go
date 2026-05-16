package overlayhud

import (
	"fmt"
	"html"
	"math"
	"strings"
)

const (
	svgOuterRadius       = 146
	svgMomentumRadius    = 132
	svgVolatilityRadius  = 108
	svgCenterPanelRadius = 64
)

// RenderSVG renders a static SVG from a RenderModel.
// It does not read Momentum state, register runtime behavior, or emit assets/routes.
func RenderSVG(model RenderModel) string {
	var b strings.Builder
	rootID := groupID(model.Groups.Root, "hud-root")

	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="%s" class="%s" role="img" aria-label="%s">`,
		escapeAttr(model.ViewBox),
		escapeAttr(strings.Join(model.StateClasses, " ")),
		escapeAttr(svgAriaLabel(model)),
	)
	fmt.Fprintf(&b, `<g id="%s">`, escapeAttr(rootID))
	renderBackground(&b, model)
	renderConfidence(&b, model)
	renderMomentumArcs(&b, model)
	renderVolatility(&b, model)
	renderCenter(&b, model)
	renderStateOverlay(&b, model)
	b.WriteString(`</g></svg>`)

	return b.String()
}

func renderBackground(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s" class="overlayhud-background">`, escapeAttr(groupID(model.Groups.Background, "hud-background")))
	fmt.Fprintf(b, `<circle cx="%s" cy="%s" r="%d" class="overlayhud-background-disc"/>`, num(model.CenterX), num(model.CenterY), svgOuterRadius+4)
	b.WriteString(`</g>`)
}

func renderConfidence(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s" class="overlayhud-confidence-ring">`, escapeAttr(groupID(model.Groups.ConfidenceRing, "hud-confidence-ring")))
	fmt.Fprintf(b, `<g id="hud-confidence-track"><circle cx="%s" cy="%s" r="%d" class="overlayhud-confidence-track"/></g>`, num(model.CenterX), num(model.CenterY), svgOuterRadius)
	fmt.Fprintf(b, `<path class="%s" data-value="%s" data-intensity="%s" d="%s"/>`,
		escapeAttr(model.Confidence.ClassName),
		num(model.Confidence.Value),
		num(model.Confidence.Intensity),
		escapeAttr(arcPath(model.CenterX, model.CenterY, svgOuterRadius, renderStartDeg, renderStartDeg+model.Confidence.Value*renderFullSweepDeg)),
	)
	b.WriteString(`<g id="hud-confidence-label"></g>`)
	b.WriteString(`</g>`)
}

func renderMomentumArcs(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s" class="overlayhud-momentum-arcs">`, escapeAttr(groupID(model.Groups.MomentumArcs, "hud-momentum-arcs")))
	fmt.Fprintf(b, `<g id="hud-momentum-track"><circle cx="%s" cy="%s" r="%d" class="overlayhud-momentum-track"/></g>`, num(model.CenterX), num(model.CenterY), svgMomentumRadius)
	b.WriteString(`<g id="hud-momentum-blue">`)
	renderArcPath(b, model.CenterX, model.CenterY, svgMomentumRadius, model.BlueArc)
	b.WriteString(`</g>`)
	b.WriteString(`<g id="hud-momentum-orange">`)
	renderArcPath(b, model.CenterX, model.CenterY, svgMomentumRadius, model.OrangeArc)
	b.WriteString(`</g>`)
	b.WriteString(`</g>`)
}

func renderArcPath(b *strings.Builder, centerX, centerY float64, radius int, arc ArcModel) {
	fmt.Fprintf(b, `<path class="%s" data-share="%s" data-start="%s" data-end="%s" data-sweep="%s" d="%s"/>`,
		escapeAttr(arc.ClassName),
		num(arc.Share),
		num(arc.StartDeg),
		num(arc.EndDeg),
		num(arc.SweepDeg),
		escapeAttr(arcPath(centerX, centerY, radius, arc.StartDeg, arc.EndDeg)),
	)
}

func renderVolatility(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s" class="overlayhud-volatility-segments">`, escapeAttr(groupID(model.Groups.VolatilityRing, "hud-volatility-segments")))
	fmt.Fprintf(b, `<g id="hud-volatility-track"><circle cx="%s" cy="%s" r="%d" class="overlayhud-volatility-track"/></g>`, num(model.CenterX), num(model.CenterY), svgVolatilityRadius)
	for _, segment := range model.Volatility {
		fmt.Fprintf(b, `<path class="%s" data-segment="%d" data-active="%t" data-intensity="%s" data-start="%s" data-end="%s" d="%s"/>`,
			escapeAttr(segment.ClassName),
			segment.Index,
			segment.Active,
			num(segment.Intensity),
			num(segment.StartDeg),
			num(segment.EndDeg),
			escapeAttr(arcPath(model.CenterX, model.CenterY, svgVolatilityRadius, segment.StartDeg, segment.EndDeg)),
		)
	}
	b.WriteString(`</g>`)
}

func renderCenter(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s" class="overlayhud-center-panel">`, escapeAttr(groupID(model.Groups.CenterCore, "hud-center-panel")))
	fmt.Fprintf(b, `<circle cx="%s" cy="%s" r="%d" class="overlayhud-center-core"/>`, num(model.CenterX), num(model.CenterY), svgCenterPanelRadius)
	b.WriteString(`</g>`)

	fmt.Fprintf(b, `<g id="%s" class="overlayhud-labels">`, escapeAttr(groupID(model.Groups.Labels, "hud-labels")))
	fmt.Fprintf(b, `<g id="hud-timer-text"><text x="%s" y="154" text-anchor="middle" class="overlayhud-timer-text">%s</text></g>`, num(model.CenterX), escapeText(model.Center.PrimaryText))
	fmt.Fprintf(b, `<g id="hud-state-label"><text x="%s" y="188" text-anchor="middle" class="overlayhud-state-label">%s</text></g>`, num(model.CenterX), escapeText(model.Center.StateLabel))
	b.WriteString(`</g>`)
}

func renderStateOverlay(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s" class="overlayhud-status-overlay">`, escapeAttr(groupID(model.Groups.StateOverlay, "hud-status-overlay")))
	if model.IsStale {
		fmt.Fprintf(b, `<text x="%s" y="214" text-anchor="middle" class="overlayhud-status-text">STALE</text>`, num(model.CenterX))
	}
	b.WriteString(`</g>`)
}

func arcPath(centerX, centerY float64, radius int, startDeg, endDeg float64) string {
	if startDeg == endDeg {
		x, y := pointOnCircle(centerX, centerY, radius, startDeg)
		return fmt.Sprintf("M %s %s", num(x), num(y))
	}
	if math.Abs(endDeg-startDeg) >= renderFullSweepDeg {
		midDeg := startDeg + (endDeg-startDeg)/2
		startX, startY := pointOnCircle(centerX, centerY, radius, startDeg)
		midX, midY := pointOnCircle(centerX, centerY, radius, midDeg)
		endX, endY := pointOnCircle(centerX, centerY, radius, endDeg)
		return fmt.Sprintf("M %s %s A %d %d 0 1 1 %s %s A %d %d 0 1 1 %s %s",
			num(startX),
			num(startY),
			radius,
			radius,
			num(midX),
			num(midY),
			radius,
			radius,
			num(endX),
			num(endY),
		)
	}

	startX, startY := pointOnCircle(centerX, centerY, radius, startDeg)
	endX, endY := pointOnCircle(centerX, centerY, radius, endDeg)
	largeArc := 0
	if math.Abs(endDeg-startDeg) > 180 {
		largeArc = 1
	}
	return fmt.Sprintf("M %s %s A %d %d 0 %d 1 %s %s",
		num(startX),
		num(startY),
		radius,
		radius,
		largeArc,
		num(endX),
		num(endY),
	)
}

func pointOnCircle(centerX, centerY float64, radius int, deg float64) (float64, float64) {
	rad := deg * math.Pi / 180
	return centerX + float64(radius)*math.Cos(rad), centerY + float64(radius)*math.Sin(rad)
}

func groupID(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func svgAriaLabel(model RenderModel) string {
	if model.Center.StateLabel == "" {
		return "Momentum overlay"
	}
	return "Momentum overlay " + model.Center.StateLabel
}

func escapeAttr(value string) string {
	return html.EscapeString(value)
}

func escapeText(value string) string {
	return html.EscapeString(value)
}

func num(value float64) string {
	return fmt.Sprintf("%.3f", value)
}
