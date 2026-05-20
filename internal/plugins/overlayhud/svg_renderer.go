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

	fmt.Fprintf(&b, `<svg id="momentum-control-wheel" xmlns="http://www.w3.org/2000/svg" viewBox="%s" class="%s" data-state="%s" data-seam-angle="%s" data-blue-share="%s" data-orange-share="%s" data-blue-pressure-share="%s" data-orange-pressure-share="%s" data-blue-control-share="%s" data-orange-control-share="%s" data-confidence-bucket="%s" data-volatility="%s" data-recent-event-energy="%s" data-recent-event-team="%s" data-recent-event-type="%s" style="%s" role="img" aria-label="%s">`,
		escapeAttr(model.ViewBox),
		escapeAttr(strings.Join(model.StateClasses, " ")),
		escapeAttr(model.DisplayState),
		num(model.SeamAngleDeg),
		num(model.Diagnostics.BlueShare),
		num(model.Diagnostics.OrangeShare),
		num(model.Diagnostics.BluePressureShare),
		num(model.Diagnostics.OrangePressureShare),
		num(model.Diagnostics.BlueControlShare),
		num(model.Diagnostics.OrangeControlShare),
		escapeAttr(model.Diagnostics.ConfidenceBucket),
		num(model.Diagnostics.Volatility),
		num(model.RecentEvent.Energy),
		escapeAttr(model.RecentEvent.Team),
		escapeAttr(model.RecentEvent.Type),
		escapeAttr(model.StyleVars),
		escapeAttr(svgAriaLabel(model)),
	)
	renderDefs(&b)
	fmt.Fprintf(&b, `<g id="%s">`, escapeAttr(rootID))
	renderBackground(&b, model)
	renderAura(&b, model)
	renderEnergyStreaks(&b, model)
	renderSparks(&b, model)
	renderFrame(&b, model)
	renderSegments(&b, model)
	renderTicks(&b, model)
	renderConfidence(&b, model)
	renderCenter(&b, model)
	renderCenterWashes(&b, model)
	renderEmptyGroup(&b, model.Groups.CenterTexture)
	renderCenterRim(&b, model)
	renderContestedFrontLine(&b, model)
	renderTextLayer(&b, model)
	renderStateOverlay(&b, model)
	renderEmptyGroup(&b, model.Groups.Badge)
	renderEmptyGroup(&b, model.Groups.DebugOverlays)
	b.WriteString(`</g></svg>`)

	return b.String()
}

func renderDefs(b *strings.Builder) {
	b.WriteString(`<defs>`)
	b.WriteString(`<filter id="mcw-soft-blur" x="-80%" y="-80%" width="260%" height="260%"><feGaussianBlur stdDeviation="18"/></filter>`)
	b.WriteString(`<radialGradient id="mcw-center-blue-wash" cx="35%" cy="42%" r="62%"><stop offset="0%" stop-color="#38B2F6"/><stop offset="100%" stop-color="#38B2F6" stop-opacity="0"/></radialGradient>`)
	b.WriteString(`<radialGradient id="mcw-center-orange-wash" cx="65%" cy="42%" r="62%"><stop offset="0%" stop-color="#F97316"/><stop offset="100%" stop-color="#F97316" stop-opacity="0"/></radialGradient>`)
	b.WriteString(`<radialGradient id="mcw-center-purple-wash" cx="50%" cy="18%" r="58%"><stop offset="0%" stop-color="#F0B8FF"/><stop offset="100%" stop-color="#B45CFF" stop-opacity="0"/></radialGradient>`)
	b.WriteString(`</defs>`)
}

func renderBackground(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s" class="overlayhud-background">`, escapeAttr(groupID(model.Groups.Background, "background")))
	fmt.Fprintf(b, `<rect id="bg-transparent-hitbox" x="0" y="0" width="1024" height="1024" fill="transparent"/>`)
	fmt.Fprintf(b, `<circle id="bg-radial-shadow" cx="%s" cy="%s" r="440" class="overlayhud-background-disc"/>`, num(model.CenterX), num(model.CenterY))
	b.WriteString(`</g>`)
}

func renderAura(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s" class="mcw-aura-layer">`, escapeAttr(groupID(model.Groups.OuterAura, "outer-aura")))
	fmt.Fprintf(b, `<path id="outer-aura-blue" class="mcw-aura mcw-aura-blue" d="%s"/>`, escapeAttr(wheelArcPath(model.CenterX, model.CenterY, 436, 180, 180+model.BlueArc.Share*renderFullSweepDeg)))
	fmt.Fprintf(b, `<path id="outer-aura-orange" class="mcw-aura mcw-aura-orange" d="%s"/>`, escapeAttr(wheelArcPath(model.CenterX, model.CenterY, 436, 180+model.BlueArc.Share*renderFullSweepDeg, 540)))
	x, y := pointOnWheel(model.CenterX, model.CenterY, 428, model.SeamAngleDeg)
	fmt.Fprintf(b, `<circle id="outer-aura-purple-contest" class="mcw-aura mcw-aura-contest" cx="%s" cy="%s" r="34"/>`, num(x), num(y))
	b.WriteString(`</g>`)
}

func renderEnergyStreaks(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s" class="mcw-streaks">`, escapeAttr(groupID(model.Groups.OuterStreaks, "outer-energy-streaks")))
	b.WriteString(`<g id="outer-energy-streaks-blue" class="mcw-streaks-blue"><line x1="228" y1="260" x2="178" y2="220"/><line x1="194" y1="354" x2="138" y2="330"/></g>`)
	b.WriteString(`<g id="outer-energy-streaks-orange" class="mcw-streaks-orange"><line x1="796" y1="260" x2="846" y2="220"/><line x1="830" y1="354" x2="886" y2="330"/></g>`)
	b.WriteString(`</g>`)
}

func renderSparks(b *strings.Builder, model RenderModel) {
	className := "mcw-sparks"
	if model.RecentEvent.ClassName != "" {
		className += " " + model.RecentEvent.ClassName
	}
	fmt.Fprintf(b, `<g id="%s" class="%s" data-recent-event-energy="%s" data-recent-event-team="%s" data-recent-event-type="%s">`,
		escapeAttr(groupID(model.Groups.OuterSparks, "outer-sparks")),
		escapeAttr(className),
		num(model.RecentEvent.Energy),
		escapeAttr(model.RecentEvent.Team),
		escapeAttr(model.RecentEvent.Type),
	)
	b.WriteString(`<g id="outer-sparks-blue"><circle class="mcw-spark mcw-spark-role-pressure" cx="214" cy="216" r="4"/><circle class="mcw-spark mcw-spark-role-control" cx="166" cy="308" r="3"/></g>`)
	b.WriteString(`<g id="outer-sparks-orange"><circle class="mcw-spark mcw-spark-role-pressure" cx="810" cy="216" r="4"/><circle class="mcw-spark mcw-spark-role-control" cx="858" cy="308" r="3"/></g>`)
	b.WriteString(`<g id="outer-sparks-purple"><circle class="mcw-spark mcw-spark-role-volatile" cx="512" cy="76" r="5"/></g>`)
	b.WriteString(`<g id="outer-sparks-white"><circle class="mcw-spark mcw-spark-role-volatile" cx="512" cy="102" r="3"/></g>`)
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
	fmt.Fprintf(b, `<circle id="center-disc-base" cx="%s" cy="%s" r="%d" class="overlayhud-center-core"/>`, num(model.CenterX), num(model.CenterY), svgCenterPanelRadius)
	fmt.Fprintf(b, `<circle id="center-disc-inner-shadow" cx="%s" cy="%s" r="%d" class="mcw-center-inner-shadow"/>`, num(model.CenterX), num(model.CenterY), svgCenterPanelRadius-10)
	b.WriteString(`</g>`)
}

func renderCenterWashes(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s">`, escapeAttr(groupID(model.Groups.CenterColorWashes, "center-color-washes")))
	fmt.Fprintf(b, `<circle id="center-disc-blue-wash" cx="%s" cy="%s" r="%d" class="mcw-center-wash-blue"/>`, num(model.CenterX), num(model.CenterY), svgCenterPanelRadius)
	fmt.Fprintf(b, `<circle id="center-disc-orange-wash" cx="%s" cy="%s" r="%d" class="mcw-center-wash-orange"/>`, num(model.CenterX), num(model.CenterY), svgCenterPanelRadius)
	fmt.Fprintf(b, `<circle id="center-disc-purple-contest-wash" cx="%s" cy="%s" r="%d" class="mcw-center-wash-purple"/>`, num(model.CenterX), num(model.CenterY), svgCenterPanelRadius)
	b.WriteString(`</g>`)
}

func renderCenterRim(b *strings.Builder, model RenderModel) {
	fmt.Fprintf(b, `<g id="%s">`, escapeAttr(groupID(model.Groups.CenterRim, "center-rim")))
	fmt.Fprintf(b, `<circle id="center-disc-rim" cx="%s" cy="%s" r="238" class="mcw-center-rim"/>`, num(model.CenterX), num(model.CenterY))
	fmt.Fprintf(b, `<ellipse id="center-disc-glass-highlight" cx="%s" cy="414" rx="124" ry="34" class="mcw-center-glass-highlight"/>`, num(model.CenterX))
	b.WriteString(`</g>`)
}

func renderContestedFrontLine(b *strings.Builder, model RenderModel) {
	line := model.ContestedLine
	if line.ClassName == "" {
		line = ContestedLineModel{
			Active:    model.HasData && model.MatchActive && !model.IsStale,
			AngleDeg:  model.SeamAngleDeg,
			BandDeg:   7.5,
			Intensity: 0.34,
			ClassName: "mcw-contested-front-line",
		}
	}
	fmt.Fprintf(b, `<g id="%s" class="%s" transform="rotate(%s 512 512)" data-contested-line-active="%t" data-contested-line-angle="%s" data-contested-line-band="%s" data-contested-line-intensity="%s">`,
		escapeAttr(groupID(model.Groups.ContestedFrontLine, "contested-front-line")),
		escapeAttr(line.ClassName),
		num(line.AngleDeg),
		line.Active,
		num(line.AngleDeg),
		num(line.BandDeg),
		num(line.Intensity),
	)
	b.WriteString(`<circle id="contest-top-core" cx="512" cy="92" r="8"/>`)
	b.WriteString(`<circle id="contest-top-purple-glow" cx="512" cy="92" r="34"/>`)
	b.WriteString(`<line id="contest-top-vertical-beam" x1="512" y1="100" x2="512" y2="270"/>`)
	b.WriteString(`<circle id="contest-bottom-seam" cx="512" cy="932" r="5"/>`)
	b.WriteString(`</g>`)
}

func renderTextLayer(b *strings.Builder, model RenderModel) {
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

func wheelArcPath(centerX, centerY float64, radius int, startDeg, endDeg float64) string {
	return arcPath(centerX, centerY, radius, wheelAngleToMathAngle(startDeg), wheelAngleToMathAngle(endDeg))
}

func pointOnCircle(centerX, centerY float64, radius int, deg float64) (float64, float64) {
	rad := deg * math.Pi / 180
	return centerX + float64(radius)*math.Cos(rad), centerY + float64(radius)*math.Sin(rad)
}

func pointOnWheel(centerX, centerY float64, radius int, deg float64) (float64, float64) {
	return pointOnCircle(centerX, centerY, radius, wheelAngleToMathAngle(deg))
}

func wheelAngleToMathAngle(deg float64) float64 {
	return deg - 90
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
