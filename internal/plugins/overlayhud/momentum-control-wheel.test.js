'use strict';

const assert = require('node:assert/strict');

global.window = {};

require('./view.js');

const math = window.MomentumControlWheelMath;

assert.ok(math, 'MomentumControlWheelMath should be exported');

{
  const config = math.sanitizeWheelConfig({});
  assert.equal(config.variant, 'compact');
  assert.equal(config.scale, 1);
  assert.equal(config.opacity, 0.92);
  assert.equal(config.theme, 'oof-default');
  assert.equal(config.debug.enabled, false);
  assert.equal(config.momentumControlWheel.visual.segments.brightness, 0.88);
  assert.equal(config.momentumControlWheel.visual.blueSegments.opacity, 0.94);
  assert.equal(config.momentumControlWheel.visual.orangeSegments.opacity, 0.94);
  assert.equal(config.momentumControlWheel.visual.centerText.confidenceBrightness, 1);
  assert.equal(config.momentumControlWheel.visual.sparks.intensity, 0.55);
  assert.equal(config.momentumControlWheel.visual.frontLine.coreSize, 1);
  assert.equal(config.momentumControlWheel.visual.seamSparks.volatileMultiplier, 1.25);
  assert.equal(config.momentumControlWheel.visual.outerSparks.dominantMultiplier, 0.85);
  assert.equal(config.momentumControlWheel.visual.eventReactions.goalFullRingPulseStrength, 1);
  assert.equal(config.momentumControlWheel.visual.stateMultipliers.neutral, 0.35);
  assert.equal(config.momentumControlWheel.response.pressureSensitivity, 1);
  assert.equal(config.momentumControlWheel.response.smoothing, 0.5);
  assert.equal(config.momentumControlWheel.response.transitionSharpness, 0.75);
  assert.equal(config.momentumControlWheel.response.timing.reactionSpeed, 1);
  assert.equal(config.momentumControlWheel.response.timing.afterglowDuration, 1.6);
  assert.equal(config.momentumControlWheel.response.timing.eventBurstDuration, 1);
}

{
  const config = math.sanitizeWheelConfig({
    theme: 'performance-safe',
    performanceMode: true,
    momentumControlWheel: {
      visual: {
        sparks: { intensity: 1, opacity: 1, reactiveness: 1 },
        outerSparks: { intensity: 1, opacity: 1 },
        aura: { intensity: 1, pulse: 1 },
        centerWash: { intensity: 1 },
      },
      response: {
        eventReactiveness: 2,
        timing: {
          afterglowDuration: 5,
          eventBurstDuration: 3,
        },
      },
    },
  });
  assert.equal(config.theme, 'performance-safe');
  assert.equal(config.performanceMode, true);
  assert.equal(config.momentumControlWheel.visual.sparks.intensity, 0);
  assert.equal(config.momentumControlWheel.visual.sparks.opacity, 0);
  assert.equal(config.momentumControlWheel.visual.outerSparks.intensity, 0);
  assert.equal(config.momentumControlWheel.visual.outerSparks.opacity, 0);
  assert.equal(config.momentumControlWheel.visual.aura.intensity, 0.28);
  assert.equal(config.momentumControlWheel.visual.aura.pulse, 0);
  assert.equal(config.momentumControlWheel.visual.centerWash.intensity, 0.42);
  assert.equal(config.momentumControlWheel.response.eventReactiveness, 0);
  assert.equal(config.momentumControlWheel.response.timing.afterglowDuration, 1.5);
  assert.equal(config.momentumControlWheel.response.timing.eventBurstDuration, 0.85);
}

{
  const config = math.sanitizeWheelConfig({
    variant: 'giant',
    scale: 99,
    opacity: -2,
    glowIntensity: 4,
    segmentBrightness: -1,
    inactiveSegmentVisibility: 9,
    seamIntensity: 'bad',
    staticAuraIntensity: 2,
    volatileEffects: -3,
    dominantPulse: 11,
    theme: 'missing',
  });
  assert.equal(config.variant, 'compact');
  assert.equal(config.scale, 1.75);
  assert.equal(config.opacity, 0.2);
  assert.equal(config.glowIntensity, 1);
  assert.equal(config.segmentBrightness, 0);
  assert.equal(config.inactiveSegmentVisibility, 1);
  assert.equal(config.seamIntensity, 0.82);
  assert.equal(config.staticAuraIntensity, 1);
  assert.equal(config.volatileEffects, 0);
  assert.equal(config.dominantPulse, 1);
  assert.equal(config.theme, 'oof-default');
}

{
  const visual = math.sanitizeWheelVisualConfig({
    segments: { brightness: 5, saturation: -1, glow: 'bad' },
    blueSegments: { brightness: 5, saturation: 3, glow: 2, opacity: -1 },
    orangeSegments: { brightness: -1, saturation: -1, glow: 'bad', opacity: 2 },
    inactiveSegments: { opacity: 0, brightness: 3 },
    seam: { intensity: 2, flare: -1, flicker: 3 },
    frontLine: { intensity: 2, coreSize: 9, glowSize: -1, opacity: 3, trailStrength: 3, trailDuration: 9, layerPriority: -1 },
    aura: { intensity: 2, pulse: 2, pulseSpeed: 9, reactiveness: -1, blueStrength: 2, orangeStrength: -1, volatilePurpleStrength: 3 },
    volatileAura: { intensity: 9, saturation: 3 },
    sparks: { intensity: 2, saturation: 3, opacity: -1, reactiveness: 2 },
    seamSparks: { intensity: 2, opacity: -1, travelDistance: 9, duration: -1, density: 9, volatileMultiplier: 9 },
    outerSparks: { intensity: 2, opacity: -1, speed: 9, pressureMultiplier: 9, controlMultiplier: -1, dominantMultiplier: 9 },
    eventReactions: { shotPulseStrength: 9, saveFlashStrength: -1, epicSaveAfterglow: 9, goalFullRingPulseStrength: 9, demoJaggedSparkStrength: -1 },
    stateMultipliers: { neutral: 9, pressure: -1, control: 9, volatile: 9, dominant: -1 },
    centerWash: { intensity: -1, saturation: 4, blueStrength: 2, orangeStrength: -1, purpleStrength: 3 },
    centerText: { brightness: 5, confidenceBrightness: -1, scale: 3 },
    innerTicks: { opacity: 2, brightness: 5, saturation: 4 },
    frame: { brightness: 2, saturation: 3, opacity: -1 },
    badge: { opacity: 2 },
  });
  assert.equal(visual.segments.brightness, 1);
  assert.equal(visual.segments.saturation, 0);
  assert.equal(visual.segments.glow, 0.72);
  assert.equal(visual.blueSegments.brightness, 1.5);
  assert.equal(visual.blueSegments.saturation, 1.5);
  assert.equal(visual.blueSegments.glow, 1);
  assert.equal(visual.blueSegments.opacity, 0);
  assert.equal(visual.orangeSegments.brightness, 0.5);
  assert.equal(visual.orangeSegments.saturation, 0);
  assert.equal(visual.orangeSegments.glow, 0.72);
  assert.equal(visual.orangeSegments.opacity, 1);
  assert.equal(visual.inactiveSegments.opacity, 0);
  assert.equal(visual.inactiveSegments.brightness, 1.5);
  assert.equal(visual.seam.intensity, 1);
  assert.equal(visual.seam.flare, 0);
  assert.equal(visual.seam.flicker, 1);
  assert.equal(visual.frontLine.intensity, 1);
  assert.equal(visual.frontLine.coreSize, 1.8);
  assert.equal(visual.frontLine.glowSize, 0.35);
  assert.equal(visual.frontLine.opacity, 1);
  assert.equal(visual.frontLine.trailStrength, 1);
  assert.equal(visual.frontLine.trailDuration, 3);
  assert.equal(visual.frontLine.layerPriority, 0);
  assert.equal(visual.aura.intensity, 1);
  assert.equal(visual.aura.pulse, 1);
  assert.equal(visual.aura.pulseSpeed, 2);
  assert.equal(visual.aura.reactiveness, 0);
  assert.equal(visual.aura.blueStrength, 1);
  assert.equal(visual.aura.orangeStrength, 0);
  assert.equal(visual.aura.volatilePurpleStrength, 1.5);
  assert.equal(visual.volatileAura.intensity, 1);
  assert.equal(visual.volatileAura.saturation, 1.6);
  assert.equal(visual.sparks.intensity, 1);
  assert.equal(visual.sparks.saturation, 1.6);
  assert.equal(visual.sparks.opacity, 0);
  assert.equal(visual.sparks.reactiveness, 1);
  assert.equal(visual.seamSparks.intensity, 1);
  assert.equal(visual.seamSparks.opacity, 0);
  assert.equal(visual.seamSparks.travelDistance, 2);
  assert.equal(visual.seamSparks.duration, 0.5);
  assert.equal(visual.seamSparks.density, 2);
  assert.equal(visual.seamSparks.volatileMultiplier, 2);
  assert.equal(visual.outerSparks.intensity, 1);
  assert.equal(visual.outerSparks.opacity, 0);
  assert.equal(visual.outerSparks.speed, 2);
  assert.equal(visual.outerSparks.pressureMultiplier, 2);
  assert.equal(visual.outerSparks.controlMultiplier, 0);
  assert.equal(visual.outerSparks.dominantMultiplier, 2);
  assert.equal(visual.eventReactions.shotPulseStrength, 2);
  assert.equal(visual.eventReactions.saveFlashStrength, 0);
  assert.equal(visual.eventReactions.epicSaveAfterglow, 2);
  assert.equal(visual.eventReactions.goalFullRingPulseStrength, 2);
  assert.equal(visual.eventReactions.demoJaggedSparkStrength, 0);
  assert.equal(visual.stateMultipliers.neutral, 2);
  assert.equal(visual.stateMultipliers.pressure, 0);
  assert.equal(visual.stateMultipliers.control, 2);
  assert.equal(visual.stateMultipliers.volatile, 2);
  assert.equal(visual.stateMultipliers.dominant, 0);
  assert.equal(visual.centerWash.intensity, 0);
  assert.equal(visual.centerWash.saturation, 1.6);
  assert.equal(visual.centerWash.blueStrength, 1);
  assert.equal(visual.centerWash.orangeStrength, 0);
  assert.equal(visual.centerWash.purpleStrength, 1);
  assert.equal(visual.centerText.brightness, 1.35);
  assert.equal(visual.centerText.confidenceBrightness, 0.72);
  assert.equal(visual.centerText.scale, 1.22);
  assert.equal(visual.innerTicks.opacity, 0.62);
  assert.equal(visual.innerTicks.brightness, 1.5);
  assert.equal(visual.innerTicks.saturation, 1.6);
  assert.equal(visual.frame.brightness, 1.5);
  assert.equal(visual.frame.saturation, 1.6);
  assert.equal(visual.frame.opacity, 0);
  assert.equal(visual.badge.opacity, 1);
}

{
  const input = {
    pressureSensitivity: 9,
    volatilitySensitivity: -2,
    confidenceInfluence: 'bad',
    smoothing: 4,
    transitionSharpness: -1,
    eventReactiveness: 3,
    timing: {
      reactionSpeed: 9,
      holdDuration: -2,
      decaySpeed: 'bad',
      pulseDuration: 4,
      afterglowDuration: -1,
      eventBurstDuration: 9,
    },
  };
  const before = JSON.stringify(input);
  const response = math.sanitizeWheelResponseConfig(input);
  assert.equal(response.pressureSensitivity, 2);
  assert.equal(response.volatilitySensitivity, 0);
  assert.equal(response.confidenceInfluence, 1);
  assert.equal(response.smoothing, 1);
  assert.equal(response.transitionSharpness, 0);
  assert.equal(response.eventReactiveness, 2);
  assert.equal(response.timing.reactionSpeed, 2);
  assert.equal(response.timing.holdDuration, 0.25);
  assert.equal(response.timing.decaySpeed, 1);
  assert.equal(response.timing.pulseDuration, 3);
  assert.equal(response.timing.afterglowDuration, 0);
  assert.equal(response.timing.eventBurstDuration, 3);
  assert.equal(JSON.stringify(input), before, 'response sanitizer should not mutate input');
}

{
  const response = math.sanitizeWheelResponseConfig({
    pressureSensitivity: 2,
    volatilitySensitivity: 2,
    confidenceInfluence: 2,
    smoothing: 1,
    transitionSharpness: 1,
    eventReactiveness: 2,
    timing: {
      reactionSpeed: 2,
      holdDuration: 3,
      decaySpeed: 2,
      pulseDuration: 3,
      afterglowDuration: 5,
      eventBurstDuration: 3,
    },
  }, { reducedMotion: true, performanceMode: true });
  assert.equal(response.pressureSensitivity, 2);
  assert.equal(response.volatilitySensitivity, 0.35);
  assert.equal(response.confidenceInfluence, 2);
  assert.equal(response.smoothing, 1);
  assert.equal(response.transitionSharpness, 0.45);
  assert.equal(response.eventReactiveness, 0);
  assert.equal(response.timing.reactionSpeed, 2);
  assert.equal(response.timing.holdDuration, 3);
  assert.equal(response.timing.decaySpeed, 2);
  assert.equal(response.timing.pulseDuration, 0.25);
  assert.equal(response.timing.afterglowDuration, 0);
  assert.equal(response.timing.eventBurstDuration, 0.25);
}

{
  const visual = math.sanitizeWheelVisualConfig({
    aura: { pulse: 1, reactiveness: 1 },
    frontLine: { trailStrength: 1, trailDuration: 3 },
    sparks: { intensity: 1, reactiveness: 1 },
    seamSparks: { intensity: 1 },
    outerSparks: { intensity: 1 },
    eventReactions: { goalFullRingPulseStrength: 2, demoJaggedSparkStrength: 2 },
    volatileAura: { intensity: 1 },
  }, { reducedMotion: true });
  assert.equal(visual.aura.pulse, 0);
  assert.equal(visual.aura.reactiveness, 0);
  assert.equal(visual.frontLine.trailStrength, 0);
  assert.equal(visual.frontLine.trailDuration, 0.25);
  assert.equal(visual.sparks.intensity, 0);
  assert.equal(visual.sparks.reactiveness, 0);
  assert.equal(visual.seamSparks.intensity, 0);
  assert.equal(visual.outerSparks.intensity, 0);
  assert.equal(visual.eventReactions.goalFullRingPulseStrength, 0);
  assert.equal(visual.eventReactions.demoJaggedSparkStrength, 0);
  assert.equal(visual.seam.flicker, 0);
  assert.equal(visual.volatileAura.intensity, 0.24);
}

{
  const visual = math.sanitizeWheelVisualConfig({
    segments: { glow: 1 },
    blueSegments: { glow: 1 },
    orangeSegments: { glow: 1 },
    seam: { flare: 1, flicker: 1 },
    frontLine: { glowSize: 2, trailStrength: 1, trailDuration: 3 },
    aura: { intensity: 1, pulse: 1 },
    sparks: { intensity: 1, opacity: 1, reactiveness: 1 },
    seamSparks: { intensity: 1, opacity: 1 },
    outerSparks: { intensity: 1, opacity: 1 },
    eventReactions: { goalFullRingPulseStrength: 2, epicSaveAfterglow: 2 },
    centerWash: { intensity: 1 },
  }, { performanceMode: true });
  assert.equal(visual.segments.glow, 0.42);
  assert.equal(visual.blueSegments.glow, 0.42);
  assert.equal(visual.orangeSegments.glow, 0.42);
  assert.equal(visual.seam.flare, 0.35);
  assert.equal(visual.seam.flicker, 0.2);
  assert.equal(visual.frontLine.glowSize, 1);
  assert.equal(visual.frontLine.trailStrength, 0.25);
  assert.equal(visual.frontLine.trailDuration, 0.8);
  assert.equal(visual.aura.intensity, 0.28);
  assert.equal(visual.aura.pulse, 0);
  assert.equal(visual.sparks.intensity, 0);
  assert.equal(visual.sparks.opacity, 0);
  assert.equal(visual.sparks.reactiveness, 0);
  assert.equal(visual.seamSparks.intensity, 0);
  assert.equal(visual.seamSparks.opacity, 0);
  assert.equal(visual.outerSparks.intensity, 0);
  assert.equal(visual.outerSparks.opacity, 0);
  assert.equal(visual.eventReactions.goalFullRingPulseStrength, 0.35);
  assert.equal(visual.eventReactions.epicSaveAfterglow, 0.35);
  assert.equal(visual.centerWash.intensity, 0.42);
}

{
  const config = math.sanitizeMomentumWidgetConfig({
    visual: 'planet',
    variant: 'massive',
    showConfidence: false,
    colorOverrides: { enabled: true, blue: '#00AAFF', text: 'bad' },
    momentumControlWheel: {
      visual: {
        sparks: { intensity: 4 },
        frame: { brightness: 4 },
      },
      response: {
        pressureSensitivity: 4,
        smoothing: -1,
        eventReactiveness: 1.4,
        timing: {
          reactionSpeed: 1.8,
          afterglowDuration: 4,
        },
      },
    },
  });
  assert.equal(config.visual, 'bar');
  assert.equal(config.variant, 'compact');
  assert.equal(config.showConfidence, false);
  assert.equal(config.colorOverrides.blue, '#00AAFF');
  assert.equal(config.colorOverrides.text, null);
  assert.equal(config.momentumControlWheel.visual.sparks.intensity, 1);
  assert.equal(config.momentumControlWheel.visual.frame.brightness, 1.5);
  assert.equal(config.momentumControlWheel.response.pressureSensitivity, 2);
  assert.equal(config.momentumControlWheel.response.smoothing, 0);
  assert.equal(config.momentumControlWheel.response.eventReactiveness, 1.4);
  assert.equal(config.momentumControlWheel.response.timing.reactionSpeed, 1.8);
  assert.equal(config.momentumControlWheel.response.timing.afterglowDuration, 4);
  assert.equal(config.momentumControlWheel.response.timing.eventBurstDuration, 1);
}

{
  const stored = math.sanitizeMomentumWidgetConfig({
    reducedMotion: true,
    performanceMode: true,
    glowIntensity: 1,
    volatileEffects: 1,
    momentumControlWheel: {
      visual: {
        blueSegments: { glow: 1 },
        orangeSegments: { glow: 1 },
        sparks: { intensity: 1, opacity: 1, reactiveness: 1 },
        aura: { intensity: 1, pulse: 1, reactiveness: 1 },
      },
      response: {
        volatilitySensitivity: 2,
        eventReactiveness: 2,
        transitionSharpness: 1,
        timing: {
          pulseDuration: 3,
          afterglowDuration: 5,
          eventBurstDuration: 3,
        },
      },
    },
  });
  assert.equal(stored.glowIntensity, 1, 'stored prefs should preserve global glow slider');
  assert.equal(stored.volatileEffects, 1, 'stored prefs should preserve volatile slider');
  assert.equal(stored.momentumControlWheel.visual.sparks.intensity, 1, 'stored prefs should preserve spark slider');
  assert.equal(stored.momentumControlWheel.visual.sparks.opacity, 1, 'stored prefs should preserve spark opacity slider');
  assert.equal(stored.momentumControlWheel.visual.aura.pulse, 1, 'stored prefs should preserve aura pulse slider');
  assert.equal(stored.momentumControlWheel.response.volatilitySensitivity, 2, 'stored prefs should preserve volatility response slider');
  assert.equal(stored.momentumControlWheel.response.eventReactiveness, 2, 'stored prefs should preserve event response slider');
  assert.equal(stored.momentumControlWheel.response.timing.pulseDuration, 3, 'stored prefs should preserve pulse timing slider');
  assert.equal(stored.momentumControlWheel.response.timing.afterglowDuration, 5, 'stored prefs should preserve afterglow timing slider');
  assert.equal(stored.momentumControlWheel.response.timing.eventBurstDuration, 3, 'stored prefs should preserve burst timing slider');
  const effective = math.sanitizeWheelConfig(stored);
  assert.equal(effective.glowIntensity, 0, 'render config should suppress glow under reduced motion');
  assert.equal(effective.volatileEffects, 0, 'render config should suppress volatile effects under reduced/performance modes');
  assert.equal(effective.momentumControlWheel.visual.sparks.intensity, 0, 'render config should suppress sparks without corrupting stored prefs');
  assert.equal(effective.momentumControlWheel.visual.sparks.opacity, 0, 'render config should suppress spark opacity in performance mode');
  assert.equal(effective.momentumControlWheel.visual.aura.pulse, 0, 'render config should suppress aura pulse without corrupting stored prefs');
  assert.equal(effective.momentumControlWheel.response.volatilitySensitivity, 0.35, 'render config should cap volatility response under reduced/performance modes');
  assert.equal(effective.momentumControlWheel.response.eventReactiveness, 0, 'render config should suppress event response under reduced/performance modes');
  assert.equal(effective.momentumControlWheel.response.timing.pulseDuration, 0.25, 'render config should minimize pulse timing under reduced motion');
  assert.equal(effective.momentumControlWheel.response.timing.afterglowDuration, 0, 'render config should disable afterglow timing under reduced motion');
  assert.equal(effective.momentumControlWheel.response.timing.eventBurstDuration, 0.25, 'render config should minimize burst timing under reduced motion');
}

{
  const config = math.sanitizeWheelConfig({
    variant: 'minimal',
    theme: 'high-contrast',
    colorOverrides: {
      enabled: true,
      blue: '#00AAFF',
      orange: '#ff8800',
      neutral: 'orange',
      frame: '#111111',
      text: '#FFFFFF',
    },
  });
  assert.equal(config.variant, 'minimal');
  assert.equal(config.theme, 'high-contrast');
  assert.equal(config.colorOverrides.enabled, true);
  assert.equal(config.colorOverrides.blue, '#00AAFF');
  assert.equal(config.colorOverrides.orange, '#ff8800');
  assert.equal(config.colorOverrides.neutral, null);
  assert.equal(config.colorOverrides.frame, '#111111');
  assert.equal(config.colorOverrides.text, '#FFFFFF');
}

{
  const config = math.sanitizeWheelConfig({
    reducedMotion: true,
    performanceMode: true,
    glowIntensity: 1,
    volatileEffects: 1,
    dominantPulse: 1,
  });
  assert.equal(config.reducedMotion, true);
  assert.equal(config.performanceMode, true);
  assert.equal(config.glowIntensity, 0);
  assert.equal(config.volatileEffects, 0);
  assert.equal(config.dominantPulse, 0);
}

{
  assert.equal(math.momentumWheelClamp(-10, 0, 100), 0);
  assert.equal(math.momentumWheelClamp(120, 0, 100), 100);
  assert.equal(math.momentumWheelClamp(44, 0, 100), 44);
}

{
  const p0 = math.polar(512, 512, 100, 0);
  const p90 = math.polar(512, 512, 100, 90);
  assert.equal(Math.round(p0.x), 512);
  assert.equal(Math.round(p0.y), 412);
  assert.equal(Math.round(p90.x), 612);
  assert.equal(Math.round(p90.y), 512);
}

{
  assert.equal(math.angularDistanceClockwise(350, 10), 20);
  assert.equal(math.angularDistanceClockwise(180, 270), 90);
  assert.equal(math.circularDistance(350, 10), 20);
  assert.equal(math.circularDistance(90, 270), 180);
}

{
  const normalized = math.normalizeMomentumWheelDisplayValues(68, 68);
  assert.equal(Number(normalized.bluePercent.toFixed(2)), 50);
  assert.equal(Number(normalized.orangePercent.toFixed(2)), 50);
}

{
  const normalized = math.normalizeMomentumWheelDisplayValues(undefined, undefined);
  assert.equal(normalized.bluePercent, 50);
  assert.equal(normalized.orangePercent, 50);
}

{
  assert.equal(math.momentumShareDisplayLabel({ bluePercent: 62, orangePercent: 38 }), 'BLUE 62%');
  assert.equal(math.momentumShareDisplayLabel({ bluePercent: 38, orangePercent: 62 }), 'ORANGE 62%');
  assert.equal(math.momentumShareDisplayLabel({ bluePercent: 50.4, orangePercent: 49.6 }), 'EVEN 50%');
}

{
  const signal = math.normalizeMomentumControlWheelSignal({
    bluePercent: 68,
    orangePercent: 32,
    state: 'blue-control',
    confidence: 'high',
  });
  assert.equal(signal.bluePercent, 68);
  assert.equal(signal.orangePercent, 32);
  assert.equal(signal.blueControlShare, 0.68);
  assert.equal(signal.orangeControlShare, 0.32);
  assert.equal(signal.state, 'blue-control');
  assert.equal(signal.confidence, 'high');
}

{
  assert.equal(math.calculateSeamAngle(50, 50), 0);
  assert.equal(math.calculateSeamAngle(75, 25), 90);
  assert.equal(math.calculateSeamAngle(25, 75), 270);
  assert.equal(math.calculateSeamAngle(100, 0), 180);
  assert.equal(math.calculateSeamAngle(0, 100), 180);
}

{
  assert.equal(math.momentumControlWheelSeamAngle(0.5), 0);
  assert.equal(math.momentumControlWheelSeamAngle(0.75), 90);
  assert.equal(math.momentumControlWheelSeamAngle(0.25), 270);
  assert.equal(math.momentumControlWheelSeamAngle(1), 180);
  assert.equal(math.momentumControlWheelSeamAngle(0), 180);
}

{
  assert.equal(math.calculateSegmentOwnership(270, 50, 50), 'blue', '50/50 left side should be blue');
  assert.equal(math.calculateSegmentOwnership(90, 50, 50), 'orange', '50/50 right side should be orange');
}

{
  assert.equal(math.calculateSegmentOwnership(45, 68, 32), 'blue', '68/32 blue should own upper-right territory');
  assert.equal(math.calculateSegmentOwnership(135, 68, 32), 'orange', '68/32 orange should be compressed lower-right');
}

{
  assert.equal(math.calculateSegmentOwnership(315, 32, 68), 'orange', '32/68 orange should own upper-left territory');
  assert.equal(math.calculateSegmentOwnership(225, 32, 68), 'blue', '32/68 blue should be compressed lower-left');
}

{
  assert.equal(math.calculateSegmentOwnership(90, 100, 0), 'blue');
  assert.equal(math.calculateSegmentOwnership(270, 0, 100), 'orange');
}

{
  const recent = math.momentumWheelRecentEvent({
    debug: {
      lastStrongEvent: {
        type: 'shot',
        team: 'blue',
        time: 100000,
      },
    },
  }, 100400);
  assert.equal(recent.recentEventType, 'shot');
  assert.equal(recent.recentEventTeam, 'blue');
  assert.ok(recent.recentEventEnergy > 0.8, 'recent shot should create visual afterglow');
}

{
  const old = math.momentumWheelRecentEvent({
    debug: {
      lastStrongEvent: {
        type: 'demo',
        team: 'orange',
        time: 100000,
      },
    },
  }, 110000);
  assert.equal(old.recentEventType, 'demo');
  assert.equal(old.recentEventTeam, 'orange');
  assert.equal(old.recentEventEnergy, 0, 'old event should decay to idle');
}

{
  const pulse = math.momentumWheelRecentEvent({
    overlay: {
      pulse: 'SAVE_FORCED',
      pulseTeam: 'blue',
    },
  }, 100000);
  assert.equal(pulse.recentEventType, 'save');
  assert.equal(pulse.recentEventTeam, 'blue');
  assert.ok(pulse.recentEventEnergy > 0.7, 'active pulse should create visual afterglow');
}

{
  const raw = {
    bluePercent: 70,
    orangePercent: 30,
    bluePressure: 0.5,
    orangePressure: 0.25,
    volatility: 0.4,
    recentEventEnergy: 0.5,
  };
  const before = JSON.stringify(raw);
  const visual = math.applyMomentumWheelDisplayResponse(raw, {
    pressureSensitivity: 2,
    volatilitySensitivity: 2,
    eventReactiveness: 2,
  });
  assert.equal(JSON.stringify(raw), before, 'display response should not mutate engine signal');
  assert.equal(raw.bluePercent, 70, 'raw ownership percent should remain untouched');
  assert.equal(visual.bluePercent, 70, 'visual response should not change ownership percent');
  assert.equal(visual.orangePercent, 30, 'visual response should not change opposing ownership percent');
  assert.ok(visual.bluePressure > raw.bluePressure, 'pressure response should scale visual pressure');
  assert.ok(visual.volatility > raw.volatility, 'volatility response should scale visual volatility');
  assert.ok(visual.recentEventEnergy > raw.recentEventEnergy, 'event response should scale visual event energy');
  const quickFade = math.applyMomentumWheelDisplayResponse(raw, {
    eventReactiveness: 1,
    timing: { holdDuration: 0.25, decaySpeed: 2 },
  });
  const longHold = math.applyMomentumWheelDisplayResponse(raw, {
    eventReactiveness: 1,
    timing: { holdDuration: 3, decaySpeed: 0.5 },
  });
  assert.ok(longHold.recentEventEnergy > quickFade.recentEventEnergy, 'visual timing should tune runtime-only event energy decay');
}

{
  const preset = math.buildMomentumWheelPresetExport({
    visual: 'wheel',
    variant: 'debug',
    theme: 'high-contrast',
    colorOverrides: { enabled: true, blue: '#00AAFF' },
    momentumControlWheel: {
      visual: {
        frontLine: { coreSize: 9 },
        seamSparks: { travelDistance: 1.4 },
      },
      response: {
        pressureSensitivity: 1.4,
        timing: { afterglowDuration: 2.8 },
      },
    },
  }, {
    position: 'top-right',
    scale: 9,
    opacity: -1,
  }, {
    name: 'Lab Tune',
    description: 'Display-only preset',
  });
  assert.equal(preset.version, 1);
  assert.equal(preset.presetType, 'momentum-control-wheel');
  assert.equal(preset.name, 'Lab Tune');
  assert.equal(preset.widget.visual, 'wheel');
  assert.equal(preset.widget.variant, 'debug');
  assert.equal(preset.widget.theme, 'high-contrast');
  assert.equal(preset.widget.momentumControlWheel.visual.frontLine.coreSize, 1.8, 'export should sanitize visual geometry-like tuning');
  assert.equal(preset.widget.momentumControlWheel.visual.seamSparks.travelDistance, 1.4);
  assert.equal(preset.widget.momentumControlWheel.response.pressureSensitivity, 1.4);
  assert.equal(preset.widget.momentumControlWheel.response.timing.afterglowDuration, 2.8);
  assert.equal(preset.widget.momentumControlWheel.timing.afterglowDuration, 2.8);
  assert.equal(preset.host.position, 'top-right');
  assert.equal(preset.host.scale, 1.25);
  assert.equal(preset.host.opacity, 0.35);
  assert.equal(JSON.stringify(preset).includes('match'), false, 'export should not include match data');
}

{
  const imported = math.parseMomentumWheelPresetJSON(JSON.stringify({
    version: 1,
    presetType: 'momentum-control-wheel',
    widget: {
      visual: 'wheel',
      variant: 'compact',
      theme: 'classic-blue-orange',
      momentumControlWheel: {
        visual: { outerSparks: { dominantMultiplier: 2.2 } },
        response: { eventReactiveness: 1.8 },
        timing: { eventBurstDuration: 2.5 },
      },
    },
    host: { position: 'side-left', scale: 1.1, opacity: 0.8 },
  }));
  assert.equal(imported.widget.visual, 'wheel');
  assert.equal(imported.widget.theme, 'classic-blue-orange');
  assert.equal(imported.widget.momentumControlWheel.visual.outerSparks.dominantMultiplier, 2);
  assert.equal(imported.widget.momentumControlWheel.response.eventReactiveness, 1.8);
  assert.equal(imported.widget.momentumControlWheel.response.timing.eventBurstDuration, 2.5);
  assert.equal(imported.host.position, 'side-left');
  assert.equal(imported.host.scale, 1.1);
  assert.equal(imported.host.opacity, 0.8);
}

{
  const previousDocument = global.document;
  global.document = createFakeDocument();

  const root = new FakeElement('div');
  const wheel = new window.MomentumControlWheelWidget(root, {
    visual: 'wheel',
    variant: 'compact',
  });

  const svg = findById(root, 'momentum-control-wheel');
  assert.ok(svg, 'SVG root exists');
  assert.equal(svg.attrs.id, 'momentum-control-wheel');
  assert.equal(svg.attrs.viewBox, '0 0 1024 1024');
  assert.equal(countByClass(root, 'mcw-segment-inactive'), 96);
  assert.equal(countByClass(root, 'mcw-segment-blue'), 96);
  assert.equal(countByClass(root, 'mcw-segment-orange'), 96);
  assert.equal(countByClass(root, 'mcw-segment-cap'), 96);
  assert.equal(findById(root, 'inner-tick-ring-muted').children.length, 120);
  const firstSegment = findByClass(root, 'mcw-segment-inactive');
  assert.equal(firstSegment.attrs.x, '505', 'segments should use corrected wider x geometry');
  assert.equal(firstSegment.attrs.y, '108', 'segments should use corrected radial y geometry');
  assert.equal(firstSegment.attrs.width, '14', 'segments should use corrected wider pill width');
  assert.equal(firstSegment.attrs.height, '108', 'segments should fill the wheel band');
  const firstTick = findById(root, 'inner-tick-ring-muted').children[0];
  assert.equal(firstTick.attrs['stroke-width'], '1.50', 'inner ticks should use reduced stroke width');
  assert.equal(firstTick.attrs.opacity, '0.22', 'inner ticks should stay inside the reduced opacity range');
  assert.equal(findById(root, 'segment-ring-inner-shadow').attrs.r, '294', 'inner shadow should fit larger segment band');
  assert.equal(findById(root, 'segment-ring-outer-highlight').attrs.r, '406', 'outer highlight should fit larger segment band');
  assert.ok(findById(root, 'outer-sparks-blue'), 'blue spark group exists');
  assert.ok(findById(root, 'outer-sparks-orange'), 'orange spark group exists');
  assert.ok(findById(root, 'outer-sparks-purple'), 'purple spark group exists');
  assert.ok(findById(root, 'outer-sparks-white'), 'white spark group exists');
  assert.ok(findById(root, 'center-reactive-layer'), 'center reactive layer exists');
  assert.equal(countByClass(root, 'mcw-spark'), 31, 'deterministic spark fixtures should include outer arc energy particles');
  assert.equal(countByClass(root, 'mcw-spark-role-neutral'), 3, 'neutral should only have sparse recent-event sparks');
  assert.equal(countByClass(root, 'mcw-spark-role-dominant'), 8, 'dominant states should use stable edge-energy sparks');
  assert.equal(countByClass(root, 'mcw-spark-zone--arc'), 12, 'pressure/control states should include visible outer-arc spark sets');
  assert.equal(countByClass(root, 'mcw-spark-role-volatile'), 9, 'volatile should lean purple-white at the seam');
  assert.equal(countByClass(root, 'mcw-spark-role-volatile-accent'), 4, 'volatile should keep secondary team-color accents');
  assert.ok(findById(root, 'text-time'), 'timer text exists');
  assert.ok(findById(root, 'text-state'), 'state label exists');
  assert.ok(findById(root, 'text-confidence-label'), 'confidence label exists');
  assert.ok(findById(root, 'oof-badge'), 'OOF badge group exists');

  wheel.update({
    time: '2:14',
    bluePercent: 60,
    orangePercent: 40,
    state: 'blue-pressure',
    confidence: 'high',
    volatility: 0.2,
    showOOFBadge: true,
  });
  assert.equal(svg.attrs['data-state'], 'blue-pressure');
  assert.equal(svg.attrs['data-confidence'], 'high');
  assert.equal(svg.attrs['data-variant'], 'compact');
  assert.equal(svg.attrs['data-theme'], 'oof-default');
  assert.equal(findById(root, 'text-state').textContent, 'BLUE PRESSURE');
  assert.equal(findById(root, 'text-confidence-label').textContent, 'MOMENTUM');
  assert.equal(findById(root, 'text-confidence-value').textContent, 'BLUE 60%');
  assert.match(findById(root, 'outer-sparks-blue').attrs.transform, /rotate\(/, 'spark groups should rotate to seam angle');
  assert.match(findById(root, 'center-reactive-layer').attrs.transform, /rotate\(/, 'center reactive layer should follow seam angle');
  assert.equal(svg.style.props['--mcw-blue-aura-base'], '0.18', 'blue pressure should raise blue aura base');
  assert.equal(svg.style.props['--mcw-blue-aura-peak'], '0.30', 'blue pressure should raise blue aura peak');
  assert.equal(svg.style.props['--mcw-orange-aura-base'], '0.045', 'blue pressure should keep orange aura subdued');
  assert.equal(root.style.props['--mcw-blue-wash-opacity'], '0.16', 'blue wash opacity should be available as a tunable CSS variable');
  assert.equal(root.style.props['--mcw-orange-wash-opacity'], '0.055', 'orange wash opacity should be available as a tunable CSS variable');
  assert.notEqual(findById(root, 'outer-aura-purple-contest').attrs.cy, '104', 'contest aura should follow seam angle');

  wheel.setConfig({
    momentumControlWheel: {
      visual: {
        segments: { brightness: 0.42, saturation: 1.3, glow: 0.24 },
        blueSegments: { brightness: 1.1, saturation: 1.2, glow: 0.64, opacity: 0.86 },
        orangeSegments: { brightness: 0.92, saturation: 0.8, glow: 0.34, opacity: 0.72 },
        inactiveSegments: { opacity: 0.24, brightness: 0.7 },
        sparks: { intensity: 0.18, saturation: 1.4, opacity: 0.62, reactiveness: 0.8 },
        frontLine: { intensity: 0.66, coreSize: 1.25, glowSize: 1.35, opacity: 0.74, trailStrength: 0.57, trailDuration: 1.4 },
        aura: { intensity: 0.31, pulse: 0.25, pulseSpeed: 1.2, reactiveness: 0.6, blueStrength: 0.82, orangeStrength: 0.68, volatilePurpleStrength: 1.25 },
        seam: { intensity: 0.7, flare: 0.3, flicker: 0.44 },
        seamSparks: { intensity: 0.43, opacity: 0.58, travelDistance: 1.4, duration: 1.25, density: 1.3, volatileMultiplier: 1.7 },
        outerSparks: { intensity: 0.33, opacity: 0.51, speed: 1.4, pressureMultiplier: 0.8, controlMultiplier: 1.2, dominantMultiplier: 0.6 },
        eventReactions: { shotPulseStrength: 1.4, saveFlashStrength: 0.8, epicSaveAfterglow: 1.3, goalFullRingPulseStrength: 1.6, demoJaggedSparkStrength: 0.9 },
        stateMultipliers: { neutral: 0.4, pressure: 1.1, control: 1.2, volatile: 1.5, dominant: 1.35 },
        centerWash: { intensity: 0.44, saturation: 1.2, blueStrength: 0.9, orangeStrength: 0.8, purpleStrength: 0.7 },
        centerText: { brightness: 1.12, confidenceBrightness: 0.93, scale: 1.08 },
        innerTicks: { opacity: 0.36, brightness: 1.18, saturation: 1.1 },
        frame: { brightness: 0.66, saturation: 0.9, opacity: 0.74 },
        badge: { opacity: 0.47 },
      },
      response: {
        pressureSensitivity: 1.6,
        volatilitySensitivity: 1.3,
        confidenceInfluence: 0.25,
        smoothing: 0.9,
        transitionSharpness: 0.2,
        eventReactiveness: 1.7,
        timing: {
          reactionSpeed: 1.4,
          holdDuration: 2.2,
          decaySpeed: 0.8,
          pulseDuration: 1.5,
          afterglowDuration: 2.75,
          eventBurstDuration: 1.25,
        },
      },
    },
  });
  assert.equal(root.style.props['--mcw-segment-brightness'], '0.42');
  assert.equal(root.style.props['--mcw-segment-saturation'], '1.3');
  assert.equal(root.style.props['--mcw-segment-glow'], '0.24');
  assert.equal(root.style.props['--mcw-blue-segment-brightness'], '1.1');
  assert.equal(root.style.props['--mcw-blue-segment-saturation'], '1.2');
  assert.equal(root.style.props['--mcw-blue-segment-glow'], '0.64');
  assert.equal(root.style.props['--mcw-blue-segment-opacity'], '0.86');
  assert.equal(root.style.props['--mcw-orange-segment-brightness'], '0.92');
  assert.equal(root.style.props['--mcw-orange-segment-saturation'], '0.8');
  assert.equal(root.style.props['--mcw-orange-segment-glow'], '0.34');
  assert.equal(root.style.props['--mcw-orange-segment-opacity'], '0.72');
  assert.equal(root.style.props['--mcw-inactive-segment-opacity'], '0.24');
  assert.equal(root.style.props['--mcw-inactive-segment-brightness'], '0.7');
  assert.equal(root.style.props['--mcw-spark-intensity'], '0.18');
  assert.equal(root.style.props['--mcw-spark-saturation'], '1.4');
  assert.equal(root.style.props['--mcw-spark-opacity'], '0.62');
  assert.equal(root.style.props['--mcw-spark-reactiveness'], '0.8');
  assert.equal(root.style.props['--mcw-static-aura-intensity'], '0.31');
  assert.equal(root.style.props['--mcw-front-line-intensity'], '0.66');
  assert.equal(root.style.props['--mcw-front-line-core-size'], '1.25');
  assert.equal(root.style.props['--mcw-front-line-glow-size'], '1.35');
  assert.equal(root.style.props['--mcw-front-line-opacity'], '0.74');
  assert.equal(root.style.props['--mcw-front-line-trail-strength'], '0.57');
  assert.equal(root.style.props['--mcw-front-line-trail-duration'], '1.4');
  assert.equal(root.style.props['--mcw-aura-pulse'], '0.25');
  assert.equal(root.style.props['--mcw-aura-pulse-speed'], '1.2');
  assert.equal(root.style.props['--mcw-aura-reactiveness'], '0.6');
  assert.equal(root.style.props['--mcw-blue-aura-strength'], '0.82');
  assert.equal(root.style.props['--mcw-orange-aura-strength'], '0.68');
  assert.equal(root.style.props['--mcw-volatile-purple-aura-strength'], '1.25');
  assert.equal(root.style.props['--mcw-seam-intensity'], '0.7');
  assert.equal(root.style.props['--mcw-seam-flare'], '0.3');
  assert.equal(root.style.props['--mcw-seam-flicker'], '0.44');
  assert.equal(root.style.props['--mcw-seam-spark-intensity'], '0.43');
  assert.equal(root.style.props['--mcw-seam-spark-opacity'], '0.58');
  assert.equal(root.style.props['--mcw-seam-spark-travel-distance'], '1.4');
  assert.equal(root.style.props['--mcw-seam-spark-duration'], '1.25');
  assert.equal(root.style.props['--mcw-seam-spark-density'], '1.3');
  assert.equal(root.style.props['--mcw-volatile-seam-spark-multiplier'], '1.7');
  assert.equal(root.style.props['--mcw-outer-arc-spark-intensity'], '0.33');
  assert.equal(root.style.props['--mcw-outer-arc-spark-opacity'], '0.51');
  assert.equal(root.style.props['--mcw-outer-arc-spark-speed'], '1.4');
  assert.equal(root.style.props['--mcw-pressure-arc-spark-multiplier'], '0.8');
  assert.equal(root.style.props['--mcw-control-edge-spark-multiplier'], '1.2');
  assert.equal(root.style.props['--mcw-dominant-edge-spark-multiplier'], '0.6');
  assert.equal(root.style.props['--mcw-shot-pulse-strength'], '1.4');
  assert.equal(root.style.props['--mcw-save-flash-strength'], '0.8');
  assert.equal(root.style.props['--mcw-epic-save-afterglow'], '1.3');
  assert.equal(root.style.props['--mcw-goal-full-ring-pulse-strength'], '1.6');
  assert.equal(root.style.props['--mcw-demo-jagged-spark-strength'], '0.9');
  assert.equal(root.style.props['--mcw-state-neutral-intensity'], '0.4');
  assert.equal(root.style.props['--mcw-state-pressure-intensity'], '1.1');
  assert.equal(root.style.props['--mcw-state-control-intensity'], '1.2');
  assert.equal(root.style.props['--mcw-state-volatile-intensity'], '1.5');
  assert.equal(root.style.props['--mcw-state-dominant-intensity'], '1.35');
  assert.equal(root.style.props['--mcw-center-wash-strength'], '0.44');
  assert.equal(root.style.props['--mcw-center-wash-saturation'], '1.2');
  assert.equal(root.style.props['--mcw-center-wash-blue-strength'], '0.9');
  assert.equal(root.style.props['--mcw-center-wash-orange-strength'], '0.8');
  assert.equal(root.style.props['--mcw-center-wash-purple-strength'], '0.7');
  assert.equal(root.style.props['--mcw-center-text-brightness'], '1.12');
  assert.equal(root.style.props['--mcw-confidence-text-brightness'], '0.93');
  assert.equal(root.style.props['--mcw-inner-tick-opacity'], '0.36');
  assert.equal(root.style.props['--mcw-inner-tick-brightness'], '1.18');
  assert.equal(root.style.props['--mcw-inner-tick-saturation'], '1.1');
  assert.equal(root.style.props['--mcw-frame-brightness'], '0.66');
  assert.equal(root.style.props['--mcw-frame-opacity'], '0.74');
  assert.equal(root.style.props['--mcw-badge-opacity'], '0.47');
  assert.equal(root.style.props['--mcw-response-pressure'], '1.6');
  assert.equal(root.style.props['--mcw-response-volatility'], '1.3');
  assert.equal(root.style.props['--mcw-response-confidence'], '0.25');
  assert.equal(root.style.props['--mcw-response-event-reactiveness'], '1.7');
  assert.equal(root.style.props['--mcw-transition-sharpness'], '0.2');
  assert.equal(root.style.props['--mcw-smoothing'], '0.9');
  assert.equal(root.style.props['--mcw-reaction-speed'], '1.4');
  assert.equal(root.style.props['--mcw-hold-duration'], '2.2');
  assert.equal(root.style.props['--mcw-decay-speed'], '0.8');
  assert.equal(root.style.props['--mcw-pulse-duration'], '1.5');
  assert.equal(root.style.props['--mcw-afterglow-duration'], '2.75');
  assert.equal(root.style.props['--mcw-event-burst-duration'], '1.25');
  assert.equal(root.style.props['--mcw-afterglow-duration-ms'], '2750ms');
  assert.equal(root.style.props['--mcw-volatile-spark-duration-ms'], '775ms');

  wheel.update({
    time: '1:11',
    bluePercent: 52,
    orangePercent: 48,
    state: 'volatile',
    confidence: 'medium',
    volatility: 0.9,
    showOOFBadge: true,
  });
  assert.equal(findById(root, 'text-state').textContent, 'CONTESTED');
  assert.equal(findById(root, 'text-confidence-value').textContent, 'BLUE 52%');
  assert.equal(svg.style.props['--mcw-contest-aura-base'], '0.36', 'volatile should raise seam-adjacent aura base');
  assert.equal(svg.style.props['--mcw-contest-aura-peak'], '0.72', 'volatile should raise seam-adjacent aura peak');

  wheel.update({
    time: '0:44',
    bluePercent: 90,
    orangePercent: 10,
    state: 'dominant-blue',
    confidence: 'max',
    volatility: 0.1,
    showOOFBadge: true,
  });
  assert.equal(svg.style.props['--mcw-blue-aura-base'], '0.34', 'dominant blue should use heavy blue aura base');
  assert.equal(svg.style.props['--mcw-blue-aura-peak'], '0.58', 'dominant blue should use heavy blue aura peak');

  wheel.setConfig({
    showConfidence: false,
    showOOFBadge: false,
    variant: 'minimal',
  });
  assert.equal(findById(root, 'text-confidence-label').hidden, true);
  assert.equal(findById(root, 'text-confidence-value').hidden, true);
  assert.equal(findById(root, 'oof-badge').style.display, 'none');
  assert.equal(svg.attrs['data-variant'], 'minimal');

  wheel.destroy();
  assert.equal(root.children.length, 0);

  global.document = previousDocument;
}

console.log('Momentum Control Wheel math tests passed');

function createFakeDocument() {
  return {
    createElementNS: (_ns, tag) => new FakeElement(tag),
    createTextNode: text => ({ tagName: '#text', textContent: String(text), children: [] }),
  };
}

function findById(node, id) {
  if (!node) return null;
  if (node.attrs?.id === id || node.id === id) return node;
  for (const child of node.children || []) {
    const found = findById(child, id);
    if (found) return found;
  }
  return null;
}

function findByClass(node, className) {
  if (!node) return null;
  if (node.classList?.contains(className)) return node;
  for (const child of node.children || []) {
    const found = findByClass(child, className);
    if (found) return found;
  }
  return null;
}

function countByClass(node, className) {
  if (!node) return 0;
  let count = node.classList?.contains(className) ? 1 : 0;
  for (const child of node.children || []) {
    count += countByClass(child, className);
  }
  return count;
}

function syncClassSet(el, value) {
  el._classSet = new Set(String(value || '').split(/\s+/).filter(Boolean));
}

function makeStyle() {
  return {
    display: '',
    props: {},
    setProperty(name, value) {
      this.props[name] = String(value);
    },
  };
}

function makeClassList(el) {
  return {
    toggle(name, force) {
      if (!el._classSet) syncClassSet(el, el.attrs.class || el.className || '');
      const shouldAdd = force === undefined ? !el._classSet.has(name) : Boolean(force);
      if (shouldAdd) {
        el._classSet.add(name);
      } else {
        el._classSet.delete(name);
      }
      el.attrs.class = Array.from(el._classSet).join(' ');
      el.className = el.attrs.class;
    },
    contains(name) {
      if (!el._classSet) syncClassSet(el, el.attrs.class || el.className || '');
      return el._classSet.has(name);
    },
  };
}

function FakeElement(tagName) {
  this.tagName = tagName;
  this.attrs = {};
  this.children = [];
  this.style = makeStyle();
  this.hidden = false;
  this.className = '';
  this._classSet = new Set();
  this.classList = makeClassList(this);
  this.setAttribute = function setAttribute(name, value) {
    this.attrs[name] = String(value);
    if (name === 'id') this.id = String(value);
    if (name === 'class') {
      this.className = String(value);
      syncClassSet(this, value);
    }
  };
  this.appendChild = function appendChild(child) {
    this.children.push(child);
    return child;
  };
  Object.defineProperty(this, 'innerHTML', {
    get() {
      return this._innerHTML || '';
    },
    set(value) {
      this._innerHTML = String(value);
      this.children = [];
    },
  });
}

FakeElement.prototype.setAttribute = function setAttribute(name, value) {
  this.attrs[name] = String(value);
  if (name === 'id') this.id = String(value);
  if (name === 'class') {
    this.className = String(value);
    syncClassSet(this, value);
  }
};

FakeElement.prototype.appendChild = function appendChild(child) {
  this.children.push(child);
  return child;
};

Object.defineProperty(FakeElement.prototype, 'innerHTML', {
  get() {
    return this._innerHTML || '';
  },
  set(value) {
    this._innerHTML = String(value);
    this.children = [];
  },
});
