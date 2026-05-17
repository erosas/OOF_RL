package overlayhud

import (
	"fmt"
	"html"
	"math"
	"strings"
)

const (
	svgOuterFrameRadius  = 410
	svgSegmentInnerRim   = 294
	svgSegmentOuterRim   = 406
	svgCenterPanelRadius = 230
)

// RenderSVG renders a static SVG from a RenderModel.
// It does not read Momentum state, register runtime behavior, or emit assets/routes.
func RenderSVG(model RenderModel) string {
	var b strings.Builder
	rootID := groupID(model.Groups.Root, "hud-root")

	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="%s" class="%s" data-state="%s" role="img" aria-label="%s">`,
		escapeAttr(model.ViewBox),
		escapeAttr(strings.Join(model.StateClasses, " ")),
		escapeAttr(model.DisplayState),
		escapeAttr(svgAriaLabel(model)),
	)
	fmt.Fprintf(&b, `<g id="%s">`, escapeAttr(rootID))
	renderBackground(&b, model)
	renderEmptyGroup(&b, model.Groups.OuterAura)
	renderEmptyGroup(&b, model.Groups.OuterStreaks)
	renderEmptyGroup(&b, model.Groups.OuterSparks)
	renderFrame(&b, model)
	renderSegments(&b, model)
	renderTicks(&b, model)
	renderConfidence(&b, model)
	renderCenter(&b, model)
	renderEmptyGroup(&b, model.Groups.CenterColorWashes)
	renderEmptyGroup(&b, model.Groups.CenterTexture)
	renderEmptyGroup(&b, model.Groups.CenterRim)
	renderEmptyGroup(&b, model.Groups.ContestedFrontLine)
	renderStateOverlay(&b, model)
	renderEmptyGroup(&b, model.Groups.Badge)
	renderEmptyGroup(&b, model.Groups.DebugOverlays)
	b.WriteString(`</g></svg>`)

	return b.String()
}

func renderBackground(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s" class="overlayhud-background">`, escapeAttr(groupID(model.Groups.Background, "background")))
	fmt.Fprintf(b, `<rect id="bg-transparent-hitbox" x="0" y="0" width="1024" height="1024" fill="transparent"/>`)
	fmt.Fprintf(b, `<circle id="bg-radial-shadow" cx="%s" cy="%s" r="440" class="overlayhud-background-disc"/>`, num(model.CenterX), num(model.CenterY))
	b.WriteString(`</g>`)
}

func renderFrame(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s" class="mcw-frame">`, escapeAttr(groupID(model.Groups.Frame, "outer-mechanical-frame")))
	fmt.Fprintf(b, `<circle id="outer-frame-base" cx="%s" cy="%s" r="%d" class="mcw-frame-base"/>`, num(model.CenterX), num(model.CenterY), svgOuterFrameRadius)
	fmt.Fprintf(b, `<circle id="outer-frame-highlight" cx="%s" cy="%s" r="%d" class="mcw-frame-highlight"/>`, num(model.CenterX), num(model.CenterY), svgSegmentOuterRim)
	fmt.Fprintf(b, `<circle id="outer-frame-shadow" cx="%s" cy="%s" r="%d" class="mcw-frame-shadow"/>`, num(model.CenterX), num(model.CenterY), svgSegmentInnerRim)
	b.WriteString(`</g>`)
}

func renderConfidence(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s" class="overlayhud-confidence-ring">`, escapeAttr(groupID(model.Groups.ConfidenceRing, "hud-confidence-ring")))
	fmt.Fprintf(b, `<g id="hud-confidence-track"><circle cx="%s" cy="%s" r="%d" class="overlayhud-confidence-track"/></g>`, num(model.CenterX), num(model.CenterY), svgOuterFrameRadius)
	fmt.Fprintf(b, `<path class="%s" data-value="%s" data-intensity="%s" d="%s"/>`,
		escapeAttr(model.Confidence.ClassName),
		num(model.Confidence.Value),
		num(model.Confidence.Intensity),
		escapeAttr(arcPath(model.CenterX, model.CenterY, svgOuterFrameRadius, renderStartDeg, renderStartDeg+model.Confidence.Value*renderFullSweepDeg)),
	)
	b.WriteString(`<g id="hud-confidence-label"></g>`)
	b.WriteString(`</g>`)
}

func renderSegments(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s">`, escapeAttr(groupID(model.Groups.SegmentUnderlay, "segment-ring-underlay")))
	for _, segment := range model.Segments {
		renderSegmentRect(b, segment, "mcw-segment mcw-segment-inactive", "segment-underlay")
	}
	b.WriteString(`</g>`)

	fmt.Fprintf(b, `<g id="%s">`, escapeAttr(groupID(model.Groups.SegmentActive, "segment-ring-active")))
	b.WriteString(`<g id="segment-ring-blue-active">`)
	for _, segment := range model.Segments {
		if segment.Owner == "blue" {
			renderSegmentRect(b, segment, segment.ClassName, "segment-active")
		}
	}
	b.WriteString(`</g>`)
	b.WriteString(`<g id="segment-ring-orange-active">`)
	for _, segment := range model.Segments {
		if segment.Owner == "orange" {
			renderSegmentRect(b, segment, segment.ClassName, "segment-active")
		}
	}
	b.WriteString(`</g>`)
	b.WriteString(`<g id="segment-ring-neutral-caps">`)
	for _, segment := range model.Segments {
		if segment.IsSeam {
			renderSegmentRect(b, segment, "mcw-segment mcw-segment-cap", "segment-cap")
		}
	}
	b.WriteString(`</g></g>`)

	fmt.Fprintf(b, `<g id="%s">`, escapeAttr(groupID(model.Groups.SegmentBevels, "segment-ring-bevels")))
	fmt.Fprintf(b, `<circle id="segment-ring-inner-shadow" cx="%s" cy="%s" r="%d" class="mcw-segment-inner-shadow"/>`, num(model.CenterX), num(model.CenterY), svgSegmentInnerRim)
	fmt.Fprintf(b, `<circle id="segment-ring-outer-highlight" cx="%s" cy="%s" r="%d" class="mcw-segment-outer-highlight"/>`, num(model.CenterX), num(model.CenterY), svgSegmentOuterRim)
	b.WriteString(`<g id="segment-ring-bevel-overlays">`)
	for _, segment := range model.Segments {
		fmt.Fprintf(b, `<rect class="mcw-segment-bevel" data-bevel-segment="%d" x="507.5" y="113" width="3" height="42" rx="999" ry="999" transform="rotate(%s 512 512)"/>`,
			segment.Index,
			num(segment.AngleDeg),
		)
	}
	b.WriteString(`</g></g>`)
}

func renderSegmentRect(b *strings.Builder, segment WheelSegment, className, dataRole string) {
	fmt.Fprintf(b, `<rect class="%s" data-role="%s" data-segment="%d" data-owner="%s" data-angle="%s" x="505" y="108" width="14" height="108" rx="999" ry="999" transform="rotate(%s 512 512)"/>`,
		escapeAttr(className),
		escapeAttr(dataRole),
		segment.Index,
		escapeAttr(segment.Owner),
		num(segment.AngleDeg),
		num(segment.AngleDeg),
	)
}

func renderTicks(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s">`, escapeAttr(groupID(model.Groups.InnerTickRing, "inner-tick-ring")))
	b.WriteString(`<g id="inner-tick-ring-base"></g>`)
	b.WriteString(`<g id="inner-tick-ring-blue"></g>`)
	b.WriteString(`<g id="inner-tick-ring-orange"></g>`)
	b.WriteString(`<g id="inner-tick-ring-muted">`)
	for _, tick := range model.Ticks {
		innerX, innerY := pointOnCircle(model.CenterX, model.CenterY, 258, tick.AngleDeg)
		outerX, outerY := pointOnCircle(model.CenterX, model.CenterY, 274, tick.AngleDeg)
		fmt.Fprintf(b, `<line class="%s" data-tick="%d" data-owner="%s" x1="%s" y1="%s" x2="%s" y2="%s"/>`,
			escapeAttr(tick.ClassName),
			tick.Index,
			escapeAttr(tick.Owner),
			num(innerX),
			num(innerY),
			num(outerX),
			num(outerY),
		)
	}
	b.WriteString(`</g>`)
	b.WriteString(`<g id="inner-crosshair-lines"><line x1="512" y1="268" x2="512" y2="756" class="mcw-crosshair-line"/><line x1="268" y1="512" x2="756" y2="512" class="mcw-crosshair-line"/></g>`)
	b.WriteString(`</g>`)
}

func renderCenter(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s" class="overlayhud-center-panel">`, escapeAttr(groupID(model.Groups.CenterCore, "hud-center-panel")))
	fmt.Fprintf(b, `<circle cx="%s" cy="%s" r="%d" class="overlayhud-center-core"/>`, num(model.CenterX), num(model.CenterY), svgCenterPanelRadius)
	b.WriteString(`</g>`)

	fmt.Fprintf(b, `<g id="%s" class="overlayhud-labels">`, escapeAttr(groupID(model.Groups.TextLayer, "text-layer")))
	fmt.Fprintf(b, `<g id="hud-timer-text"><text id="text-time" x="%s" y="506" text-anchor="middle" class="overlayhud-timer-text mcw-time">%s</text></g>`, num(model.CenterX), escapeText(model.Center.PrimaryText))
	fmt.Fprintf(b, `<g id="hud-state-label"><text id="text-state" x="%s" y="570" text-anchor="middle" class="overlayhud-state-label mcw-state-label">%s</text></g>`, num(model.CenterX), escapeText(model.Center.StateLabel))
	b.WriteString(`</g>`)
}

func renderStateOverlay(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s" class="overlayhud-status-overlay">`, escapeAttr(groupID(model.Groups.StateOverlay, "hud-status-overlay")))
	if model.IsStale {
		fmt.Fprintf(b, `<text x="%s" y="214" text-anchor="middle" class="overlayhud-status-text">STALE</text>`, num(model.CenterX))
	}
	b.WriteString(`</g>`)
}

func renderEmptyGroup(b *strings.Builder, id string) {
	fmt.Fprintf(b, `<g id="%s"></g>`, escapeAttr(id))
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
