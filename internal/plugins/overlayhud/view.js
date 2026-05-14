'use strict';

let _overlayMomentumTimer = null;
let _momentumFlowBar = null;
let _momentumControlWheel = null;
let _lastMomentumOutput = null;
let _demoModeUntil = 0;
let _timelineStart = 0;
let _timelineSamples = [];
let _timelineEvents = [];
let _lastTimelineEventKey = '';
let _heldDisplayState = null;
let _lastDisplayDominance = null;
let _lastDisplayOutput = null;
let _replayFrozenOutput = null;
let _wasReplayActive = false;
let _postReplayNeutralStarted = 0;
let _postReplayNeutralUntil = 0;
let _postReplaySignalAfter = 0;
let _awaitingPostReplayLiveSignal = false;
let _lastGoalFallbackKey = '';
let _goalFallbackStarted = 0;
let _lastMomentumResetAt = 0;
let _currentMomentumWidgetConfig = null;
let _currentMomentumHostConfig = null;
let _lastServerPrefsAt = 0;
let _prefsPushTimer = null;
let _lastOverlayPerfSnapshot = null;

const MFB_PREFS_KEY = 'oofrl.overlay.momentumFlowBar.v1';
const OVERLAY_PERF_KEY = 'oofrl.overlay.perf.enabled';
const OVERLAY_PERF_CLIENT_KEY = 'oofrl.overlay.perf.clientId';
const OVERLAY_PERF_SCHEMA_VERSION = 2;
const MAX_TIMELINE_SAMPLES = 180;
const MAX_TIMELINE_EVENTS = 40;
const TIMELINE_MARKER_CLUSTER_MS = 6500;
const DISPLAY_STATE_MIN_HOLD_MS = 3600;
const DISPLAY_STATE_MAX_HOLD_MS = 9000;
const DISPLAY_CONTROL_PERCENT = 56;
const DISPLAY_PRESSURE_PERCENT = 66;
const DISPLAY_CONTESTED_SPREAD_PERCENT = 8;
const DISPLAY_CONTESTED_FLIP_MS = 3200;
const POST_REPLAY_MIN_NEUTRAL_MS = 1200;
const POST_REPLAY_MAX_NEUTRAL_MS = 12000;
const GOAL_FALLBACK_RESET_MS = 3750;
const WHEEL_EVENT_GLOW_FULL_MS = 900;
const WHEEL_EVENT_GLOW_DECAY_MS = 5200;
const WHEEL_GOAL_GLOW_DECAY_MS = 7600;

const _overlayPerf = createOverlayPerfState();

const DEFAULT_WHEEL_VISUAL_CONFIG = {
  segments: { brightness: 0.88, saturation: 1, glow: 0.72 },
  blueSegments: { brightness: 0.88, saturation: 1, glow: 0.72, opacity: 0.94 },
  orangeSegments: { brightness: 0.88, saturation: 1, glow: 0.72, opacity: 0.94 },
  inactiveSegments: { opacity: 0.28, brightness: 0.78 },
  seam: { intensity: 0.82, flare: 0.5, flicker: 0.5 },
  frontLine: { intensity: 0.82, coreSize: 1, glowSize: 1, opacity: 1, trailStrength: 0.5, trailDuration: 1, layerPriority: 1 },
  aura: { intensity: 0.45, pulse: 0.5, pulseSpeed: 1, reactiveness: 0.5, blueStrength: 1, orangeStrength: 1, volatilePurpleStrength: 1 },
  volatileAura: { intensity: 0.55, saturation: 1 },
  sparks: { intensity: 0.55, saturation: 1, opacity: 0.92, reactiveness: 0.5 },
  seamSparks: { intensity: 0.55, opacity: 0.92, travelDistance: 1, duration: 1, density: 1, volatileMultiplier: 1.25 },
  outerSparks: { intensity: 0.55, opacity: 0.82, speed: 1, pressureMultiplier: 1, controlMultiplier: 1, dominantMultiplier: 0.85 },
  eventReactions: { shotPulseStrength: 1, saveFlashStrength: 1, epicSaveAfterglow: 1, goalFullRingPulseStrength: 1, demoJaggedSparkStrength: 1 },
  stateMultipliers: { neutral: 0.35, pressure: 1, control: 1.12, volatile: 1.25, dominant: 1.18 },
  centerWash: { intensity: 0.5, saturation: 1, blueStrength: 1, orangeStrength: 1, purpleStrength: 1 },
  centerText: { brightness: 1, confidenceBrightness: 1, scale: 1 },
  innerTicks: { opacity: 0.28, brightness: 1, saturation: 1 },
  frame: { brightness: 0.5, saturation: 1, opacity: 1 },
  badge: { opacity: 1 },
};

const DEFAULT_WHEEL_RESPONSE_CONFIG = {
  pressureSensitivity: 1,
  volatilitySensitivity: 1,
  confidenceInfluence: 1,
  smoothing: 0.5,
  transitionSharpness: 0.75,
  eventReactiveness: 1,
  timing: {
    reactionSpeed: 1,
    holdDuration: 1,
    decaySpeed: 1,
    pulseDuration: 1,
    afterglowDuration: 1.6,
    eventBurstDuration: 1,
  },
};

const DEFAULT_MFB_CONFIG = {
  enabled: true,
  visual: 'bar',
  variant: 'compact',
  showConfidence: true,
  showLabels: true,
  showPercentages: true,
  smoothTransitions: true,
  pulseEnabled: true,
  lowConfidenceDimThreshold: 0.25,
  showOOFBadge: true,
  showStateLabel: true,
  showTelemetryWaveform: true,
  reducedMotion: false,
  performanceMode: false,
  theme: 'oof-default',
  glowIntensity: 0.72,
  segmentBrightness: 0.88,
  inactiveSegmentVisibility: 0.28,
  seamIntensity: 0.82,
  staticAuraIntensity: 0.45,
  volatileEffects: 0.55,
  dominantPulse: 0.45,
  timerScale: 1,
  labelScale: 1,
  forceHighContrastText: true,
  colorOverrides: {
    enabled: false,
    blue: null,
    orange: null,
    neutral: null,
    frame: null,
    text: null,
  },
  debug: false,
  debugWheel: {
    enabled: false,
    showSegmentIndexes: false,
    showSeamAngle: false,
    showOwnershipBoundaries: false,
    previewState: null,
  },
  momentumControlWheel: {
    visual: DEFAULT_WHEEL_VISUAL_CONFIG,
    response: DEFAULT_WHEEL_RESPONSE_CONFIG,
  },
};

const DEFAULT_MOMENTUM_WHEEL_CONFIG = {
  enabled: true,
  variant: 'compact',
  scale: 1,
  opacity: 0.92,
  position: {
    preset: 'bottom-center',
    x: 0,
    y: 0,
    anchor: 'center',
  },
  showOOFBadge: true,
  showConfidence: true,
  showStateLabel: true,
  showTelemetryWaveform: true,
  reducedMotion: false,
  performanceMode: false,
  glowIntensity: 0.72,
  segmentBrightness: 0.88,
  inactiveSegmentVisibility: 0.28,
  seamIntensity: 0.82,
  staticAuraIntensity: 0.45,
  volatileEffects: 0.55,
  dominantPulse: 0.45,
  timerScale: 1,
  labelScale: 1,
  forceHighContrastText: true,
  theme: 'oof-default',
  colorOverrides: {
    enabled: false,
    blue: null,
    orange: null,
    neutral: null,
    frame: null,
    text: null,
  },
  debug: {
    enabled: false,
    showSegmentIndexes: false,
    showSeamAngle: false,
    showOwnershipBoundaries: false,
    previewState: null,
  },
  momentumControlWheel: {
    visual: DEFAULT_WHEEL_VISUAL_CONFIG,
    response: DEFAULT_WHEEL_RESPONSE_CONFIG,
  },
};

const MOMENTUM_WHEEL_THEMES = {
  'oof-default': {
    blueCore: '#1ea7ff',
    blueEdge: '#76d8ff',
    blueGlow: '#00c8ff',
    orangeCore: '#ff7a1a',
    orangeEdge: '#ffc06a',
    orangeGlow: '#ff8a2a',
    frame: '#202834',
    frameEdge: '#5e7086',
    center: '#07111c',
    textPrimary: '#f4fbff',
    textSecondary: '#9edcff',
    textMuted: '#7f93a8',
    seam: '#f7fbff',
  },
  'high-contrast': {
    blueCore: '#00aaff',
    blueEdge: '#ffffff',
    blueGlow: '#00d8ff',
    orangeCore: '#ff8a00',
    orangeEdge: '#ffffff',
    orangeGlow: '#ff9f2f',
    frame: '#111111',
    frameEdge: '#aaaaaa',
    center: '#000000',
    textPrimary: '#ffffff',
    textSecondary: '#ffffff',
    textMuted: '#d0d0d0',
    seam: '#ffffff',
  },
  'classic-blue-orange': {
    blueCore: '#0077ff',
    blueEdge: '#4db8ff',
    blueGlow: '#009dff',
    orangeCore: '#ff6200',
    orangeEdge: '#ffb15c',
    orangeGlow: '#ff7a00',
    frame: '#1b2028',
    frameEdge: '#4b5568',
    center: '#080d14',
    textPrimary: '#eef8ff',
    textSecondary: '#a8d8ff',
    textMuted: '#8090a0',
    seam: '#ffffff',
  },
  'reduced-glow': {
    blueCore: '#2f9bd8',
    blueEdge: '#75c7ea',
    blueGlow: '#2f9bd8',
    orangeCore: '#d9742f',
    orangeEdge: '#e8ad72',
    orangeGlow: '#d9742f',
    frame: '#20242b',
    frameEdge: '#59616d',
    center: '#0b1017',
    textPrimary: '#f0f4f8',
    textSecondary: '#a8bfce',
    textMuted: '#7c8792',
    seam: '#dce7ee',
  },
  'performance-safe': {
    blueCore: '#2f8ed8',
    blueEdge: '#8fcdf0',
    blueGlow: '#2f8ed8',
    orangeCore: '#d8782f',
    orangeEdge: '#efba82',
    orangeGlow: '#d8782f',
    frame: '#20242b',
    frameEdge: '#687384',
    center: '#080e16',
    textPrimary: '#f3f8ff',
    textSecondary: '#bad4e8',
    textMuted: '#8d98a5',
    seam: '#eef6ff',
  },
};

const DEFAULT_HOST_CONFIG = {
  position: 'top-center',
  opacity: 1,
  scale: 1,
};

const MOMENTUM_WHEEL_SPARKS = {
  blue: [
    { type: 'line', zone: 'seam', roles: ['pressure', 'control', 'volatile-accent', 'neutral'], x: -18, r: 426, angle: -4, length: 20, travel: 42, drift: -10, delay: 0, duration: 760, opacity: 0.72, width: 3 },
    { type: 'circle', zone: 'seam', roles: ['pressure', 'control'], x: -6, r: 438, angle: 2, size: 3, travel: 30, drift: -6, delay: 130, duration: 820, opacity: 0.62 },
    { type: 'diamond', zone: 'seam', roles: ['pressure', 'control', 'volatile-accent'], x: -30, r: 444, angle: -8, size: 6, travel: 48, drift: -15, delay: 260, duration: 900, opacity: 0.6 },
    { type: 'line', zone: 'seam', roles: ['pressure', 'control'], x: 8, r: 420, angle: 5, length: 18, travel: 36, drift: 7, delay: 390, duration: 780, opacity: 0.58, width: 2.5 },
    { type: 'circle', zone: 'seam', roles: ['pressure'], x: -42, r: 414, angle: -12, size: 2.6, travel: 26, drift: -14, delay: 570, duration: 880, opacity: 0.5 },
    { type: 'line', zone: 'arc', roles: ['pressure', 'control', 'dominant'], localAngle: -38, r: 452, angle: -22, length: 30, travel: 58, drift: -8, delay: 720, duration: 1280, opacity: 0.58, width: 3.5 },
    { type: 'diamond', zone: 'arc', roles: ['pressure', 'control', 'dominant'], localAngle: -54, r: 438, angle: -30, size: 7, travel: 46, drift: -6, delay: 980, duration: 1460, opacity: 0.5 },
    { type: 'line', zone: 'arc', roles: ['pressure', 'control'], localAngle: -72, r: 468, angle: -35, length: 34, travel: 66, drift: -9, delay: 1120, duration: 1180, opacity: 0.54, width: 3 },
    { type: 'line', zone: 'arc', roles: ['control', 'dominant'], localAngle: -92, r: 462, angle: -46, length: 42, travel: 72, drift: -6, delay: 1360, duration: 1720, opacity: 0.46, width: 4 },
    { type: 'circle', zone: 'arc', roles: ['pressure'], localAngle: -28, r: 474, angle: -18, size: 3.2, travel: 40, drift: -8, delay: 1540, duration: 1080, opacity: 0.48 },
    { type: 'jagged', zone: 'arc', roles: ['dominant'], localAngle: -112, r: 456, angle: -52, travel: 64, drift: -5, delay: 1760, duration: 1880, opacity: 0.42, width: 3 },
  ],
  orange: [
    { type: 'line', zone: 'seam', roles: ['pressure', 'control', 'volatile-accent', 'neutral'], x: 18, r: 426, angle: 4, length: 20, travel: 42, drift: 10, delay: 80, duration: 760, opacity: 0.72, width: 3 },
    { type: 'circle', zone: 'seam', roles: ['pressure', 'control'], x: 6, r: 438, angle: -2, size: 3, travel: 30, drift: 6, delay: 210, duration: 820, opacity: 0.62 },
    { type: 'diamond', zone: 'seam', roles: ['pressure', 'control', 'volatile-accent'], x: 30, r: 444, angle: 8, size: 6, travel: 48, drift: 15, delay: 340, duration: 900, opacity: 0.6 },
    { type: 'line', zone: 'seam', roles: ['pressure', 'control'], x: -8, r: 420, angle: -5, length: 18, travel: 36, drift: -7, delay: 470, duration: 780, opacity: 0.58, width: 2.5 },
    { type: 'circle', zone: 'seam', roles: ['pressure'], x: 42, r: 414, angle: 12, size: 2.6, travel: 26, drift: 14, delay: 650, duration: 880, opacity: 0.5 },
    { type: 'line', zone: 'arc', roles: ['pressure', 'control', 'dominant'], localAngle: 38, r: 452, angle: 22, length: 30, travel: 58, drift: 8, delay: 800, duration: 1280, opacity: 0.58, width: 3.5 },
    { type: 'diamond', zone: 'arc', roles: ['pressure', 'control', 'dominant'], localAngle: 54, r: 438, angle: 30, size: 7, travel: 46, drift: 6, delay: 1060, duration: 1460, opacity: 0.5 },
    { type: 'line', zone: 'arc', roles: ['pressure', 'control'], localAngle: 72, r: 468, angle: 35, length: 34, travel: 66, drift: 9, delay: 1200, duration: 1180, opacity: 0.54, width: 3 },
    { type: 'line', zone: 'arc', roles: ['control', 'dominant'], localAngle: 92, r: 462, angle: 46, length: 42, travel: 72, drift: 6, delay: 1440, duration: 1720, opacity: 0.46, width: 4 },
    { type: 'circle', zone: 'arc', roles: ['pressure'], localAngle: 28, r: 474, angle: 18, size: 3.2, travel: 40, drift: 8, delay: 1620, duration: 1080, opacity: 0.48 },
    { type: 'jagged', zone: 'arc', roles: ['dominant'], localAngle: 112, r: 456, angle: 52, travel: 64, drift: 5, delay: 1840, duration: 1880, opacity: 0.42, width: 3 },
  ],
  purple: [
    { type: 'jagged', zone: 'seam', roles: ['volatile'], x: -16, r: 432, angle: -2, travel: 46, drift: -10, delay: 20, duration: 610, opacity: 0.9, width: 3 },
    { type: 'jagged', zone: 'seam', roles: ['volatile'], x: 14, r: 446, angle: 3, travel: 64, drift: 12, delay: 130, duration: 660, opacity: 0.84, width: 3 },
    { type: 'diamond', zone: 'seam', roles: ['volatile'], x: 0, r: 454, angle: 0, size: 8, travel: 70, drift: 0, delay: 240, duration: 620, opacity: 0.92 },
    { type: 'line', zone: 'seam', roles: ['volatile'], x: 24, r: 428, angle: 6, length: 20, travel: 40, drift: 16, delay: 360, duration: 690, opacity: 0.72, width: 2.5 },
    { type: 'jagged', zone: 'seam', roles: ['volatile'], x: -5, r: 418, angle: -6, travel: 52, drift: -4, delay: 480, duration: 640, opacity: 0.78, width: 2.5 },
    { type: 'circle', zone: 'seam', roles: ['volatile'], x: 9, r: 458, angle: 4, size: 3, travel: 62, drift: 5, delay: 580, duration: 700, opacity: 0.74 },
  ],
  white: [
    { type: 'line', zone: 'seam', roles: ['volatile', 'neutral'], x: -4, r: 418, angle: 0, length: 28, travel: 42, drift: -3, delay: 120, duration: 860, opacity: 0.88, width: 4 },
    { type: 'circle', zone: 'seam', roles: ['volatile'], x: 7, r: 456, angle: 1, size: 3.4, travel: 58, drift: 5, delay: 420, duration: 900, opacity: 0.8 },
    { type: 'diamond', zone: 'seam', roles: ['volatile'], x: -11, r: 446, angle: -3, size: 5.5, travel: 50, drift: -5, delay: 640, duration: 960, opacity: 0.74 },
  ],
};

const WHEEL_VISUAL_RANGE_CONTROLS = [
  ['mcw-blue-segment-brightness', ['blueSegments', 'brightness']],
  ['mcw-blue-segment-saturation', ['blueSegments', 'saturation']],
  ['mcw-blue-segment-glow', ['blueSegments', 'glow']],
  ['mcw-blue-segment-opacity', ['blueSegments', 'opacity']],
  ['mcw-orange-segment-brightness', ['orangeSegments', 'brightness']],
  ['mcw-orange-segment-saturation', ['orangeSegments', 'saturation']],
  ['mcw-orange-segment-glow', ['orangeSegments', 'glow']],
  ['mcw-orange-segment-opacity', ['orangeSegments', 'opacity']],
  ['mcw-inactive-segment-opacity', ['inactiveSegments', 'opacity']],
  ['mcw-inactive-segment-brightness', ['inactiveSegments', 'brightness']],
  ['mcw-aura-intensity', ['aura', 'intensity']],
  ['mcw-aura-pulse', ['aura', 'pulse']],
  ['mcw-aura-reactiveness', ['aura', 'reactiveness']],
  ['mcw-blue-aura-strength', ['aura', 'blueStrength']],
  ['mcw-orange-aura-strength', ['aura', 'orangeStrength']],
  ['mcw-aura-pulse-speed', ['aura', 'pulseSpeed']],
  ['mcw-volatile-purple-aura-strength', ['aura', 'volatilePurpleStrength']],
  ['mcw-seam-intensity', ['seam', 'intensity']],
  ['mcw-seam-flare', ['seam', 'flare']],
  ['mcw-seam-flicker', ['seam', 'flicker']],
  ['mcw-front-line-intensity', ['frontLine', 'intensity']],
  ['mcw-front-line-core-size', ['frontLine', 'coreSize']],
  ['mcw-front-line-glow-size', ['frontLine', 'glowSize']],
  ['mcw-front-line-opacity', ['frontLine', 'opacity']],
  ['mcw-front-line-trail-strength', ['frontLine', 'trailStrength']],
  ['mcw-front-line-trail-duration', ['frontLine', 'trailDuration']],
  ['mcw-volatile-aura-intensity', ['volatileAura', 'intensity']],
  ['mcw-volatile-aura-saturation', ['volatileAura', 'saturation']],
  ['mcw-spark-intensity', ['sparks', 'intensity']],
  ['mcw-spark-saturation', ['sparks', 'saturation']],
  ['mcw-spark-opacity', ['sparks', 'opacity']],
  ['mcw-spark-reactiveness', ['sparks', 'reactiveness']],
  ['mcw-seam-spark-intensity', ['seamSparks', 'intensity']],
  ['mcw-seam-spark-opacity', ['seamSparks', 'opacity']],
  ['mcw-seam-spark-travel-distance', ['seamSparks', 'travelDistance']],
  ['mcw-seam-spark-duration', ['seamSparks', 'duration']],
  ['mcw-seam-spark-density', ['seamSparks', 'density']],
  ['mcw-volatile-seam-spark-multiplier', ['seamSparks', 'volatileMultiplier']],
  ['mcw-outer-arc-spark-intensity', ['outerSparks', 'intensity']],
  ['mcw-outer-arc-spark-opacity', ['outerSparks', 'opacity']],
  ['mcw-outer-arc-spark-speed', ['outerSparks', 'speed']],
  ['mcw-pressure-arc-spark-multiplier', ['outerSparks', 'pressureMultiplier']],
  ['mcw-control-edge-spark-multiplier', ['outerSparks', 'controlMultiplier']],
  ['mcw-dominant-edge-spark-multiplier', ['outerSparks', 'dominantMultiplier']],
  ['mcw-shot-pulse-strength', ['eventReactions', 'shotPulseStrength']],
  ['mcw-save-flash-strength', ['eventReactions', 'saveFlashStrength']],
  ['mcw-epic-save-afterglow', ['eventReactions', 'epicSaveAfterglow']],
  ['mcw-goal-full-ring-pulse-strength', ['eventReactions', 'goalFullRingPulseStrength']],
  ['mcw-demo-jagged-spark-strength', ['eventReactions', 'demoJaggedSparkStrength']],
  ['mcw-state-neutral-intensity', ['stateMultipliers', 'neutral']],
  ['mcw-state-pressure-intensity', ['stateMultipliers', 'pressure']],
  ['mcw-state-control-intensity', ['stateMultipliers', 'control']],
  ['mcw-state-volatile-intensity', ['stateMultipliers', 'volatile']],
  ['mcw-state-dominant-intensity', ['stateMultipliers', 'dominant']],
  ['mcw-center-wash-strength', ['centerWash', 'intensity']],
  ['mcw-center-wash-saturation', ['centerWash', 'saturation']],
  ['mcw-center-wash-blue-strength', ['centerWash', 'blueStrength']],
  ['mcw-center-wash-orange-strength', ['centerWash', 'orangeStrength']],
  ['mcw-center-wash-purple-strength', ['centerWash', 'purpleStrength']],
  ['mcw-center-text-brightness', ['centerText', 'brightness']],
  ['mcw-confidence-text-brightness', ['centerText', 'confidenceBrightness']],
  ['mcw-badge-opacity', ['badge', 'opacity']],
  ['mcw-frame-brightness', ['frame', 'brightness']],
  ['mcw-frame-opacity', ['frame', 'opacity']],
  ['mcw-inner-tick-opacity', ['innerTicks', 'opacity']],
  ['mcw-inner-tick-brightness', ['innerTicks', 'brightness']],
  ['mcw-inner-tick-saturation', ['innerTicks', 'saturation']],
];

const WHEEL_RESPONSE_RANGE_CONTROLS = [
  ['mcw-response-pressure-sensitivity', ['pressureSensitivity']],
  ['mcw-response-volatility-sensitivity', ['volatilitySensitivity']],
  ['mcw-response-confidence-influence', ['confidenceInfluence']],
  ['mcw-response-smoothing', ['smoothing']],
  ['mcw-response-transition-sharpness', ['transitionSharpness']],
  ['mcw-response-event-reactiveness', ['eventReactiveness']],
  ['mcw-response-reaction-speed', ['timing', 'reactionSpeed']],
  ['mcw-response-hold-duration', ['timing', 'holdDuration']],
  ['mcw-response-decay-speed', ['timing', 'decaySpeed']],
  ['mcw-response-pulse-duration', ['timing', 'pulseDuration']],
  ['mcw-response-afterglow-duration', ['timing', 'afterglowDuration']],
  ['mcw-response-event-burst-duration', ['timing', 'eventBurstDuration']],
  ['mcw-event-afterglow-duration', ['timing', 'afterglowDuration']],
  ['mcw-event-burst-duration', ['timing', 'eventBurstDuration']],
];

class MomentumFlowBarWidget {
  constructor(root, config = {}) {
    this.root = root;
    this.config = { ...DEFAULT_MFB_CONFIG, ...config };
    this.lastPulseKey = '';
    this.renderShell();
  }

  setConfig(config = {}) {
    this.config = { ...this.config, ...config };
    this.applyConfig();
    if (this.lastOutput) this.update(this.lastOutput);
  }

  update(rawOutput) {
    if (!this.config.enabled || !this.root) return;
    const output = adaptMomentumFlowOutput(rawOutput);
    this.lastOutput = output;
    if (this.config.visual === 'wheel') {
      overlayPerfCount('bar.hiddenSkip');
      return;
    }
    this.applyConfig();

    this.root.className = [
      'momentum-flow-bar',
      `momentum-flow-bar--${this.config.variant}`,
      stateClass(output.state),
      this.config.pulseEnabled && output.pulseTeam ? `is-pulse-${output.pulseTeam}` : '',
      output.confidence < this.config.lowConfidenceDimThreshold ? 'is-low-confidence' : '',
    ].filter(Boolean).join(' ');

    const blue = clampPercent(output.bluePercent);
    const orange = 100 - blue;
    this.setText('bluePercent', `${Math.round(blue)}%`);
    this.setText('orangePercent', `${Math.round(orange)}%`);
    this.setText('leftLabel', leftStateLabel(output.state));
    this.setText('centerLabel', this.config.showConfidence ? `CONFIDENCE: ${output.confidence.toFixed(2)}` : '');
    this.setText('rightLabel', rightStateLabel(output.state));
    this.blueBar.style.width = `${blue}%`;
    this.orangeBar.style.width = `${orange}%`;

    this.triggerPulse(output);
  }

  renderShell() {
    this.root.innerHTML = `
      <div class="momentum-flow-bar__panel">
        <div class="momentum-flow-bar__main">
          <div class="momentum-flow-bar__percent momentum-flow-bar__percent--blue" data-mfb="bluePercent">50%</div>
          <div class="momentum-flow-bar__rail-wrap">
            <div class="momentum-flow-bar__rail">
              <div class="momentum-flow-bar__blue" data-mfb="blueBar" style="width:50%"></div>
              <div class="momentum-flow-bar__orange" data-mfb="orangeBar" style="width:50%"></div>
            </div>
            <div class="momentum-flow-bar__pulse" data-mfb="pulse" aria-hidden="true">
              <svg viewBox="0 0 32 32" role="img" aria-label="Momentum pulse">
                <path d="M3 17h6l3-9 6 18 3-9h8" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"></path>
              </svg>
            </div>
          </div>
          <div class="momentum-flow-bar__percent momentum-flow-bar__percent--orange" data-mfb="orangePercent">50%</div>
        </div>
        <div class="momentum-flow-bar__labels">
          <div class="momentum-flow-bar__label momentum-flow-bar__label--left" data-mfb="leftLabel">Neutral</div>
          <div class="momentum-flow-bar__label momentum-flow-bar__label--center" data-mfb="centerLabel">Confidence: 0.00</div>
          <div class="momentum-flow-bar__label momentum-flow-bar__label--right" data-mfb="rightLabel">Neutral</div>
        </div>
      </div>
    `;
    this.blueBar = this.root.querySelector('[data-mfb="blueBar"]');
    this.orangeBar = this.root.querySelector('[data-mfb="orangeBar"]');
    this.applyConfig();
  }

  applyConfig() {
    this.root.classList.toggle('momentum-flow-bar--minimal', this.config.variant === 'minimal');
    this.root.classList.toggle('momentum-flow-bar--compact', this.config.variant === 'compact');
    this.root.querySelectorAll('.momentum-flow-bar__percent').forEach(el => {
      el.hidden = !this.config.showPercentages || this.config.variant === 'minimal';
    });
    const labels = this.root.querySelector('.momentum-flow-bar__labels');
    if (labels) labels.hidden = !this.config.showLabels || this.config.variant === 'minimal';
    const center = this.root.querySelector('[data-mfb="centerLabel"]');
    if (center) center.hidden = !this.config.showConfidence || this.config.variant === 'minimal';
  }

  triggerPulse(output) {
    const pulse = output.pulse || '';
    const pulseKey = pulse ? (output.pulseKey || `${pulse}:${output.state}:${output.pulseTeam || ''}`) : '';
    this.root.classList.remove('is-pulsing', 'is-pulse-goal');
    if (!this.config.pulseEnabled || !pulse || pulseKey === this.lastPulseKey) return;
    this.lastPulseKey = pulseKey;
    requestAnimationFrame(() => {
      this.root.classList.add('is-pulsing');
      if (pulse === 'GOAL_BURST') this.root.classList.add('is-pulse-goal');
      window.setTimeout(() => {
        this.root.classList.remove('is-pulsing', 'is-pulse-goal');
      }, 620);
    });
  }

  setText(key, value) {
    const el = this.root.querySelector(`[data-mfb="${key}"]`);
    if (el) el.textContent = value;
  }
}

class MomentumControlWheel {
  constructor(root, config = {}) {
    this.root = root;
    this.config = sanitizeWheelConfig({ ...DEFAULT_MOMENTUM_WHEEL_CONFIG, ...config });
    this.refs = {};
    this.segmentRefs = [];
    this.tickRefs = [];
    this.effects = [];
    this.lastSignal = null;
    this.lastVisualKey = '';
    this.lastTimerKey = '';
    this.lastPerformanceFilterMode = null;
    this.renderShell();
  }

  setConfig(config = {}) {
    const previousWheel = this.config.momentumControlWheel || {};
    const nextWheel = config.momentumControlWheel || {};
    this.config = sanitizeWheelConfig({
      ...this.config,
      ...config,
      colorOverrides: {
        ...(this.config.colorOverrides || {}),
        ...(config.colorOverrides || {}),
      },
      debug: {
        ...(this.config.debug || {}),
        ...(config.debug || config.debugWheel || {}),
      },
      momentumControlWheel: {
        ...previousWheel,
        ...nextWheel,
        visual: mergeWheelVisualConfig(previousWheel.visual, nextWheel.visual),
        response: mergeWheelResponseConfig(previousWheel.response, nextWheel.response),
      },
    });
    this.applyConfigToRoot();
    this.applyVariantVisibility();
    this.lastVisualKey = '';
    this.lastTimerKey = '';
    if (this.lastSignal) this.update(this.lastSignal);
  }

  destroy() {
    for (const timer of this.effects) {
      window.clearTimeout(timer);
      window.clearInterval(timer);
    }
    this.effects = [];
    if (this.root) this.root.innerHTML = '';
    this.refs = {};
    this.segmentRefs = [];
    this.tickRefs = [];
    this.lastSignal = null;
    this.lastVisualKey = '';
    this.lastTimerKey = '';
  }

  update(rawOutput) {
    if (!this.config.enabled || !this.root) return;
    const signal = adaptMomentumControlWheelSignal(rawOutput);
    const visualKey = momentumWheelPerfSignalKey(signal);
    const timerKey = String(signal.time || '');
    this.lastSignal = signal;
    _overlayPerf.lastSignalKey = visualKey;
    if (this.config.visual !== 'wheel') {
      this.lastTimerKey = timerKey;
      overlayPerfCount('wheel.hiddenSkip');
      return;
    }
    overlayPerfCount('wheel.update');
    if (this.lastVisualKey === visualKey) {
      overlayPerfCount('wheel.duplicateSignal');
      if (this.lastTimerKey !== timerKey) {
        this.lastTimerKey = timerKey;
        overlayPerfCount('wheel.timerOnlyUpdate');
        this.updateTimeOnly(signal);
        return;
      }
      overlayPerfCount('wheel.skippedDuplicate');
      return;
    }
    this.lastVisualKey = visualKey;
    this.lastTimerKey = timerKey;
    this.applyConfigToRoot();
    this.applyVariantVisibility();
    const response = this.config.momentumControlWheel.response;
    const visualSignal = applyMomentumWheelDisplayResponse(signal, response);

    const blue = signal.bluePercent;
    const orange = signal.orangePercent;
    const seam = calculateSeamAngle(blue, orange);
    const state = signal.state;
    const stateConfig = momentumWheelStateConfig(visualSignal);
    const reduced = Boolean(signal.reducedMotion || this.config.reducedMotion);
    const performance = Boolean(signal.performanceMode || this.config.performanceMode);
    this.applyPerformanceRenderingMode(performance || reduced);
    const recentEnergy = momentumWheelClamp(visualSignal.recentEventEnergy, 0, 1);
    const recentTeam = signal.recentEventTeam || '';

    this.root.className = [
      'momentum-control-wheel',
      `momentum-control-wheel--${this.config.variant}`,
      `mcw-state-${state}`,
      recentEnergy > 0.05 ? 'is-recent-event' : '',
      recentTeam === 'blue' ? 'is-recent-blue' : '',
      recentTeam === 'orange' ? 'is-recent-orange' : '',
      reduced ? 'is-reduced-motion' : '',
      performance ? 'is-performance-mode' : '',
      state === 'dominant-blue' || state === 'dominant-orange' ? 'is-dominant' : '',
    ].filter(Boolean).join(' ');

    this.refs.svg?.setAttribute('data-state', state);
    this.refs.svg?.setAttribute('data-confidence', signal.confidence);
    this.refs.svg?.setAttribute('data-variant', this.config.variant);
    this.refs.svg?.setAttribute('data-theme', this.config.theme);
    this.refs.svg?.setAttribute('data-reduced-motion', String(reduced));
    this.refs.svg?.setAttribute('data-performance-mode', String(performance));
    this.refs.svg?.setAttribute('data-recent-event', recentEnergy > 0.05 ? signal.recentEventType || 'event' : 'none');
    this.refs.svg?.style.setProperty('--mcw-blue-pressure', String(visualSignal.bluePressure));
    this.refs.svg?.style.setProperty('--mcw-orange-pressure', String(visualSignal.orangePressure));
    this.refs.svg?.style.setProperty('--mcw-volatility', String(visualSignal.volatility));
    this.refs.svg?.style.setProperty('--mcw-recent-event-energy', String(recentEnergy));
    this.refs.svg?.style.setProperty('--mcw-recent-blue-energy', String(recentTeam === 'blue' ? recentEnergy : 0));
    this.refs.svg?.style.setProperty('--mcw-recent-orange-energy', String(recentTeam === 'orange' ? recentEnergy : 0));
    this.refs.svg?.style.setProperty('--mcw-center-wash-opacity', String(stateConfig.centerOpacity));
    this.refs.svg?.style.setProperty('--mcw-outer-aura-opacity', String(stateConfig.auraOpacity));
    this.refs.svg?.style.setProperty('--mcw-blue-aura-base', String(stateConfig.blueAuraBase));
    this.refs.svg?.style.setProperty('--mcw-blue-aura-peak', String(stateConfig.blueAuraPeak));
    this.refs.svg?.style.setProperty('--mcw-orange-aura-base', String(stateConfig.orangeAuraBase));
    this.refs.svg?.style.setProperty('--mcw-orange-aura-peak', String(stateConfig.orangeAuraPeak));
    this.refs.svg?.style.setProperty('--mcw-contest-aura-base', String(stateConfig.contestAuraBase));
    this.refs.svg?.style.setProperty('--mcw-contest-aura-peak', String(stateConfig.contestAuraPeak));
    const visual = this.config.momentumControlWheel.visual;
    const pulseRatio = response.timing.pulseDuration / DEFAULT_WHEEL_RESPONSE_CONFIG.timing.pulseDuration;
    const pulseMs = stateConfig.auraPulseMs * (1.28 - visual.aura.reactiveness * 0.48) * pulseRatio / visual.aura.pulseSpeed;
    this.refs.svg?.style.setProperty('--mcw-aura-pulse-ms', `${Math.round(pulseMs)}ms`);

    this.updateText(signal, stateConfig);
    this.updateSegments(signal, seam);
    this.updateTicks(signal);
    this.updateEffects(visualSignal, seam, reduced, performance, stateConfig);
  }

  renderShell() {
    this.root.innerHTML = '';
    const svg = svgEl('svg', {
      id: 'momentum-control-wheel',
      class: 'momentum-control-wheel-svg',
      viewBox: '0 0 1024 1024',
      role: 'img',
      'aria-label': 'Momentum Control Wheel',
    });
    this.refs.svg = svg;
    this.root.appendChild(svg);

    this.buildDefs(svg);
    this.buildLayerGroups(svg);
    this.buildBackground();
    this.buildOuterAura();
    this.buildOuterEnergy();
    this.buildOuterSparks();
    this.buildFrame();
    this.buildSegments();
    this.buildTicks();
    this.buildCenter();
    this.buildContestLine();
    this.buildText();
    this.buildBadge();
    this.buildDebug();

    this.applyConfigToRoot();
    this.applyVariantVisibility();
    this.update({
      time: '--:--',
      bluePercent: 50,
      orangePercent: 50,
      state: 'neutral',
      confidence: 'low',
      volatility: 0,
      showOOFBadge: true,
    });
  }

  applyConfig() {
    this.applyConfigToRoot();
    this.applyVariantVisibility();
  }

  applyConfigToRoot() {
    if (!this.root) return;
    this.root.hidden = this.config.visual !== 'wheel';
    this.root.style.setProperty('--mcw-scale', String(this.config.scale));
    this.root.style.setProperty('--mcw-opacity', String(this.config.opacity));
    this.root.style.setProperty('--mcw-glow-intensity', String(this.config.glowIntensity));
    this.root.style.setProperty('--mcw-segment-brightness', String(this.config.segmentBrightness));
    this.root.style.setProperty('--mcw-inactive-opacity', String(this.config.inactiveSegmentVisibility));
    this.root.style.setProperty('--mcw-seam-intensity', String(this.config.seamIntensity));
    this.root.style.setProperty('--mcw-static-aura-intensity', String(this.config.staticAuraIntensity));
    this.root.style.setProperty('--mcw-volatile-effects', String(this.config.volatileEffects));
    this.root.style.setProperty('--mcw-dominant-pulse', String(this.config.dominantPulse));
    this.root.style.setProperty('--mcw-timer-scale', String(this.config.timerScale));
    this.root.style.setProperty('--mcw-label-scale', String(this.config.labelScale));

    const visual = this.config.momentumControlWheel.visual;
    const response = this.config.momentumControlWheel.response;
    const timing = response.timing;
    const transitionMs = Math.round(momentumWheelClamp(
      (500 - response.transitionSharpness * 240 + response.smoothing * 220) / timing.reactionSpeed,
      80,
      900,
    ));
    const effectTransitionMs = Math.round(momentumWheelClamp(
      (420 - response.transitionSharpness * 170 + response.smoothing * 260) / timing.decaySpeed,
      80,
      900,
    ));
    const afterglowMs = Math.round(timing.afterglowDuration * 1000);
    const eventBurstRatio = timing.eventBurstDuration / DEFAULT_WHEEL_RESPONSE_CONFIG.timing.eventBurstDuration;
    this.root.style.setProperty('--mcw-segment-brightness', String(visual.segments.brightness));
    this.root.style.setProperty('--mcw-inactive-opacity', String(visual.inactiveSegments.opacity));
    this.root.style.setProperty('--mcw-seam-intensity', String(visual.seam.intensity));
    this.root.style.setProperty('--mcw-static-aura-intensity', String(visual.aura.intensity));
    this.root.style.setProperty('--mcw-volatile-effects', String(Math.max(visual.volatileAura.intensity, visual.sparks.intensity)));
    this.root.style.setProperty('--mcw-segment-saturation', String(visual.segments.saturation));
    this.root.style.setProperty('--mcw-segment-glow', String(visual.segments.glow));
    this.root.style.setProperty('--mcw-blue-segment-brightness', String(visual.blueSegments.brightness));
    this.root.style.setProperty('--mcw-blue-segment-saturation', String(visual.blueSegments.saturation));
    this.root.style.setProperty('--mcw-blue-segment-glow', String(visual.blueSegments.glow));
    this.root.style.setProperty('--mcw-blue-segment-opacity', String(visual.blueSegments.opacity));
    this.root.style.setProperty('--mcw-orange-segment-brightness', String(visual.orangeSegments.brightness));
    this.root.style.setProperty('--mcw-orange-segment-saturation', String(visual.orangeSegments.saturation));
    this.root.style.setProperty('--mcw-orange-segment-glow', String(visual.orangeSegments.glow));
    this.root.style.setProperty('--mcw-orange-segment-opacity', String(visual.orangeSegments.opacity));
    this.root.style.setProperty('--mcw-inactive-segment-opacity', String(visual.inactiveSegments.opacity));
    this.root.style.setProperty('--mcw-inactive-segment-brightness', String(visual.inactiveSegments.brightness));
    this.root.style.setProperty('--mcw-aura-intensity', String(visual.aura.intensity));
    this.root.style.setProperty('--mcw-aura-pulse', String(visual.aura.pulse));
    this.root.style.setProperty('--mcw-seam-flare', String(visual.seam.flare));
    this.root.style.setProperty('--mcw-seam-flicker', String(visual.seam.flicker));
    this.root.style.setProperty('--mcw-front-line-intensity', String(visual.frontLine.intensity));
    this.root.style.setProperty('--mcw-front-line-core-size', String(visual.frontLine.coreSize));
    this.root.style.setProperty('--mcw-front-line-glow-size', String(visual.frontLine.glowSize));
    this.root.style.setProperty('--mcw-front-line-opacity', String(visual.frontLine.opacity));
    this.root.style.setProperty('--mcw-front-line-trail-strength', String(visual.frontLine.trailStrength));
    this.root.style.setProperty('--mcw-front-line-trail-duration', String(visual.frontLine.trailDuration));
    this.root.style.setProperty('--mcw-front-line-layer-priority', String(visual.frontLine.layerPriority));
    this.root.style.setProperty('--mcw-aura-reactiveness', String(visual.aura.reactiveness));
    this.root.style.setProperty('--mcw-aura-pulse-strength', String(visual.aura.pulse));
    this.root.style.setProperty('--mcw-aura-pulse-speed', String(visual.aura.pulseSpeed));
    this.root.style.setProperty('--mcw-blue-aura-strength', String(visual.aura.blueStrength));
    this.root.style.setProperty('--mcw-orange-aura-strength', String(visual.aura.orangeStrength));
    this.root.style.setProperty('--mcw-volatile-purple-aura-strength', String(visual.aura.volatilePurpleStrength));
    this.root.style.setProperty('--mcw-volatile-aura-intensity', String(visual.volatileAura.intensity));
    this.root.style.setProperty('--mcw-volatile-aura-saturation', String(visual.volatileAura.saturation));
    this.root.style.setProperty('--mcw-spark-intensity', String(visual.sparks.intensity));
    this.root.style.setProperty('--mcw-spark-saturation', String(visual.sparks.saturation));
    this.root.style.setProperty('--mcw-spark-opacity', String(visual.sparks.opacity));
    this.root.style.setProperty('--mcw-spark-reactiveness', String(visual.sparks.reactiveness));
    this.root.style.setProperty('--mcw-seam-spark-intensity', String(visual.seamSparks.intensity));
    this.root.style.setProperty('--mcw-seam-spark-opacity', String(visual.seamSparks.opacity));
    this.root.style.setProperty('--mcw-seam-spark-travel-distance', String(visual.seamSparks.travelDistance));
    this.root.style.setProperty('--mcw-seam-spark-duration', String(visual.seamSparks.duration));
    this.root.style.setProperty('--mcw-seam-spark-density', String(visual.seamSparks.density));
    this.root.style.setProperty('--mcw-volatile-seam-spark-multiplier', String(visual.seamSparks.volatileMultiplier));
    this.root.style.setProperty('--mcw-outer-arc-spark-intensity', String(visual.outerSparks.intensity));
    this.root.style.setProperty('--mcw-outer-arc-spark-opacity', String(visual.outerSparks.opacity));
    this.root.style.setProperty('--mcw-outer-arc-spark-speed', String(visual.outerSparks.speed));
    this.root.style.setProperty('--mcw-pressure-arc-spark-multiplier', String(visual.outerSparks.pressureMultiplier));
    this.root.style.setProperty('--mcw-control-edge-spark-multiplier', String(visual.outerSparks.controlMultiplier));
    this.root.style.setProperty('--mcw-dominant-edge-spark-multiplier', String(visual.outerSparks.dominantMultiplier));
    this.root.style.setProperty('--mcw-shot-pulse-strength', String(visual.eventReactions.shotPulseStrength));
    this.root.style.setProperty('--mcw-save-flash-strength', String(visual.eventReactions.saveFlashStrength));
    this.root.style.setProperty('--mcw-epic-save-afterglow', String(visual.eventReactions.epicSaveAfterglow));
    this.root.style.setProperty('--mcw-goal-full-ring-pulse-strength', String(visual.eventReactions.goalFullRingPulseStrength));
    this.root.style.setProperty('--mcw-demo-jagged-spark-strength', String(visual.eventReactions.demoJaggedSparkStrength));
    this.root.style.setProperty('--mcw-state-neutral-intensity', String(visual.stateMultipliers.neutral));
    this.root.style.setProperty('--mcw-state-pressure-intensity', String(visual.stateMultipliers.pressure));
    this.root.style.setProperty('--mcw-state-control-intensity', String(visual.stateMultipliers.control));
    this.root.style.setProperty('--mcw-state-volatile-intensity', String(visual.stateMultipliers.volatile));
    this.root.style.setProperty('--mcw-state-dominant-intensity', String(visual.stateMultipliers.dominant));
    this.root.style.setProperty('--mcw-center-wash-strength', String(visual.centerWash.intensity));
    this.root.style.setProperty('--mcw-center-wash-saturation', String(visual.centerWash.saturation));
    this.root.style.setProperty('--mcw-center-wash-blue-strength', String(visual.centerWash.blueStrength));
    this.root.style.setProperty('--mcw-center-wash-orange-strength', String(visual.centerWash.orangeStrength));
    this.root.style.setProperty('--mcw-center-wash-purple-strength', String(visual.centerWash.purpleStrength));
    this.root.style.setProperty('--mcw-center-text-brightness', String(visual.centerText.brightness));
    this.root.style.setProperty('--mcw-confidence-text-brightness', String(visual.centerText.confidenceBrightness));
    this.root.style.setProperty('--mcw-timer-scale', String(this.config.timerScale * visual.centerText.scale));
    this.root.style.setProperty('--mcw-label-scale', String(this.config.labelScale * visual.centerText.scale));
    this.root.style.setProperty('--mcw-inner-tick-opacity', String(visual.innerTicks.opacity));
    this.root.style.setProperty('--mcw-inner-tick-brightness', String(visual.innerTicks.brightness));
    this.root.style.setProperty('--mcw-inner-tick-saturation', String(visual.innerTicks.saturation));
    this.root.style.setProperty('--mcw-frame-brightness', String(visual.frame.brightness));
    this.root.style.setProperty('--mcw-frame-opacity', String(visual.frame.opacity));
    this.root.style.setProperty('--mcw-frame-saturation', String(visual.frame.saturation));
    this.root.style.setProperty('--mcw-badge-opacity', String(visual.badge.opacity));
    this.root.style.setProperty('--mcw-response-pressure', String(response.pressureSensitivity));
    this.root.style.setProperty('--mcw-response-volatility', String(response.volatilitySensitivity));
    this.root.style.setProperty('--mcw-response-confidence', String(response.confidenceInfluence));
    this.root.style.setProperty('--mcw-response-event-reactiveness', String(response.eventReactiveness));
    this.root.style.setProperty('--mcw-transition-sharpness', String(response.transitionSharpness));
    this.root.style.setProperty('--mcw-smoothing', String(response.smoothing));
    this.root.style.setProperty('--mcw-reaction-speed', String(timing.reactionSpeed));
    this.root.style.setProperty('--mcw-hold-duration', String(timing.holdDuration));
    this.root.style.setProperty('--mcw-decay-speed', String(timing.decaySpeed));
    this.root.style.setProperty('--mcw-pulse-duration', String(timing.pulseDuration));
    this.root.style.setProperty('--mcw-afterglow-duration', String(timing.afterglowDuration));
    this.root.style.setProperty('--mcw-event-burst-duration', String(timing.eventBurstDuration));
    this.root.style.setProperty('--mcw-response-transition-ms', `${transitionMs}ms`);
    this.root.style.setProperty('--mcw-response-effect-transition-ms', `${effectTransitionMs}ms`);
    this.root.style.setProperty('--mcw-afterglow-duration-ms', `${afterglowMs}ms`);
    this.root.style.setProperty('--mcw-neutral-spark-duration-ms', `${Math.round(780 * eventBurstRatio)}ms`);
    this.root.style.setProperty('--mcw-control-spark-duration-ms', `${Math.round(1040 * eventBurstRatio)}ms`);
    this.root.style.setProperty('--mcw-volatile-spark-duration-ms', `${Math.round(620 * eventBurstRatio)}ms`);
    this.root.style.setProperty('--mcw-dominant-spark-duration-ms', `${Math.round(1680 * eventBurstRatio)}ms`);
    this.root.style.setProperty('--mcw-contest-flicker-duration-ms', `${Math.round(760 * eventBurstRatio)}ms`);
    this.root.style.setProperty('--mcw-render-performance-safe', this.config.performanceMode ? '1' : '0');

    const theme = resolveWheelTheme(this.config);
    this.root.style.setProperty('--mcw-blue-core', theme.blueCore);
    this.root.style.setProperty('--mcw-blue-edge', theme.blueEdge);
    this.root.style.setProperty('--mcw-blue-glow', theme.blueGlow);
    this.root.style.setProperty('--mcw-orange-core', theme.orangeCore);
    this.root.style.setProperty('--mcw-orange-edge', theme.orangeEdge);
    this.root.style.setProperty('--mcw-orange-glow', theme.orangeGlow);
    this.root.style.setProperty('--mcw-frame', theme.frame);
    this.root.style.setProperty('--mcw-frame-edge', theme.frameEdge);
    this.root.style.setProperty('--mcw-center', theme.center);
    this.root.style.setProperty('--mcw-text-primary', theme.textPrimary);
    this.root.style.setProperty('--mcw-text-secondary', theme.textSecondary);
    this.root.style.setProperty('--mcw-text-muted', theme.textMuted);
    this.root.style.setProperty('--mcw-seam', theme.seam);

    this.refs.svg?.setAttribute('data-variant', this.config.variant);
    this.refs.svg?.setAttribute('data-theme', this.config.theme);
    this.refs.svg?.setAttribute('data-reduced-motion', String(this.config.reducedMotion));
    this.refs.svg?.setAttribute('data-performance-mode', String(this.config.performanceMode));
  }

  applyVariantVisibility() {
    if (!this.root) return;
    this.refs.svg?.classList.toggle('momentum-control-wheel-svg--minimal', this.config.variant === 'minimal');
    this.refs.svg?.classList.toggle('momentum-control-wheel-svg--compact', this.config.variant === 'compact');
    this.refs.svg?.classList.toggle('momentum-control-wheel-svg--debug', this.config.variant === 'debug' && this.config.debug.enabled);
    if (this.refs.confidenceLabel) this.refs.confidenceLabel.hidden = !this.config.showConfidence;
    if (this.refs.confidenceValue) this.refs.confidenceValue.hidden = !this.config.showConfidence;
    if (this.refs.state) this.refs.state.hidden = !this.config.showStateLabel;
    if (this.refs['oof-badge']) this.refs['oof-badge'].style.display = this.config.showOOFBadge ? '' : 'none';
    const hideWaveform = this.config.performanceMode || this.config.reducedMotion || this.config.variant === 'minimal';
    if (this.refs.waveform) this.refs.waveform.style.display = this.config.showTelemetryWaveform && !hideWaveform ? '' : 'none';
    if (this.refs['debug-overlays']) this.refs['debug-overlays'].style.display = this.config.variant === 'debug' && this.config.debug.enabled ? '' : 'none';
  }

  applyPerformanceRenderingMode(disableFilters) {
    if (this.lastPerformanceFilterMode === disableFilters) return;
    this.lastPerformanceFilterMode = disableFilters;
    const setFilter = (el, value) => {
      if (!el) return;
      if (disableFilters) {
        if (typeof el.removeAttribute === 'function') {
          el.removeAttribute('filter');
        } else if (el.attrs) {
          delete el.attrs.filter;
        }
      } else if (value) {
        el.setAttribute('filter', value);
      }
    };
    setFilter(this.refs.bgRadialShadow, 'url(#mcw-soft-blur)');
    setFilter(this.refs.auraBlue, 'url(#mcw-soft-blur)');
    setFilter(this.refs.auraOrange, 'url(#mcw-soft-blur)');
    setFilter(this.refs.auraContest, 'url(#mcw-soft-blur)');
    setFilter(this.refs.contestCore, 'url(#mcw-hot-glow)');
    setFilter(this.refs.contestGlow, 'url(#mcw-soft-blur)');
    for (const ref of this.segmentRefs || []) {
      setFilter(ref.cap, 'url(#mcw-hot-glow)');
    }
  }

  buildDefs(svg) {
    const defs = svgEl('defs');
    this.refs.defs = defs;
    svg.appendChild(defs);

    defs.appendChild(linearGradient('mcw-blue-segment-gradient', [
      ['0%', 'var(--mcw-blue-edge)'],
      ['38%', 'var(--mcw-blue-core)'],
      ['100%', 'var(--mcw-blue-glow)'],
    ]));
    defs.appendChild(linearGradient('mcw-orange-segment-gradient', [
      ['0%', 'var(--mcw-orange-edge)'],
      ['38%', 'var(--mcw-orange-core)'],
      ['100%', 'var(--mcw-orange-glow)'],
    ]));
    defs.appendChild(radialGradient('mcw-bg-vignette-gradient', [
      ['0%', 'rgba(8,17,31,0.64)'],
      ['54%', 'rgba(5,10,18,0.22)'],
      ['100%', 'rgba(5,10,18,0)'],
    ]));
    defs.appendChild(radialGradient('mcw-blue-wash-gradient', [
      ['0%', 'rgba(56,178,246,0.78)'],
      ['100%', 'rgba(11,99,255,0)'],
    ]));
    defs.appendChild(radialGradient('mcw-orange-wash-gradient', [
      ['0%', 'rgba(249,115,22,0.78)'],
      ['100%', 'rgba(255,78,0,0)'],
    ]));
    defs.appendChild(radialGradient('mcw-purple-wash-gradient', [
      ['0%', 'rgba(240,184,255,0.72)'],
      ['100%', 'rgba(180,92,255,0)'],
    ]));
    const clip = svgEl('clipPath', { id: 'mcw-center-disc-clip' });
    clip.appendChild(svgEl('circle', { cx: '512', cy: '512', r: '222' }));
    defs.appendChild(clip);

    const blur = svgEl('filter', { id: 'mcw-soft-blur', x: '-70%', y: '-70%', width: '240%', height: '240%' });
    blur.appendChild(svgEl('feGaussianBlur', { stdDeviation: '18' }));
    defs.appendChild(blur);

    const glow = svgEl('filter', { id: 'mcw-hot-glow', x: '-90%', y: '-90%', width: '280%', height: '280%' });
    glow.appendChild(svgEl('feGaussianBlur', { stdDeviation: '8', result: 'blur' }));
    glow.appendChild(svgEl('feMerge', {}, [
      svgEl('feMergeNode', { in: 'blur' }),
      svgEl('feMergeNode', { in: 'SourceGraphic' }),
    ]));
    defs.appendChild(glow);

    const pattern = svgEl('pattern', {
      id: 'mcw-honeycomb-pattern',
      width: '34',
      height: '29',
      patternUnits: 'userSpaceOnUse',
    });
    pattern.appendChild(svgEl('path', {
      d: 'M8.5 1 L25.5 1 L34 14.5 L25.5 28 L8.5 28 L0 14.5 Z',
      fill: 'none',
      stroke: '#9CA3AF',
      'stroke-width': '1',
      opacity: '0.55',
    }));
    defs.appendChild(pattern);
  }

  buildLayerGroups(svg) {
    const groupIds = [
      'background',
      'outer-aura',
      'outer-energy-streaks',
      'outer-sparks',
      'outer-mechanical-frame',
      'segment-ring-underlay',
      'segment-ring-active',
      'segment-ring-bevels',
      'inner-tick-ring',
      'center-disc',
      'center-color-washes',
      'center-texture',
      'center-rim',
      'contested-front-line',
      'text-layer',
      'oof-badge',
      'debug-overlays',
    ];
    for (const id of groupIds) {
      const g = svgEl('g', { id });
      this.refs[id] = g;
      svg.appendChild(g);
    }
  }

  buildBackground() {
    this.refs.background.appendChild(svgEl('rect', {
      id: 'bg-transparent-hitbox',
      x: '0',
      y: '0',
      width: '1024',
      height: '1024',
      fill: 'transparent',
    }));
    this.refs.background.appendChild(svgEl('circle', {
      id: 'bg-vignette',
      cx: '512',
      cy: '512',
      r: '470',
      fill: 'url(#mcw-bg-vignette-gradient)',
      opacity: '0.5',
    }));
    this.refs.bgRadialShadow = svgEl('circle', {
      id: 'bg-radial-shadow',
      cx: '512',
      cy: '540',
      r: '420',
      fill: '#050A12',
      opacity: '0.18',
      filter: 'url(#mcw-soft-blur)',
    });
    this.refs.background.appendChild(this.refs.bgRadialShadow);
    this.refs.background.appendChild(svgEl('g', { id: 'bg-subtle-noise', opacity: '0.04' }, [
      svgEl('circle', { cx: '302', cy: '266', r: '2', fill: '#F5F7FB' }),
      svgEl('circle', { cx: '706', cy: '338', r: '1.5', fill: '#F5F7FB' }),
      svgEl('circle', { cx: '594', cy: '736', r: '1.6', fill: '#F5F7FB' }),
    ]));
  }

  buildOuterAura() {
    this.refs.auraBlue = svgEl('circle', {
      id: 'outer-aura-blue',
      class: 'mcw-aura mcw-aura-blue',
      cx: '512',
      cy: '512',
      r: '430',
      fill: 'none',
      stroke: '#38B2F6',
      'stroke-width': '32',
      opacity: '0.28',
      filter: 'url(#mcw-soft-blur)',
    });
    this.refs.auraOrange = svgEl('circle', {
      id: 'outer-aura-orange',
      class: 'mcw-aura mcw-aura-orange',
      cx: '512',
      cy: '512',
      r: '430',
      fill: 'none',
      stroke: '#F97316',
      'stroke-width': '32',
      opacity: '0.28',
      filter: 'url(#mcw-soft-blur)',
    });
    this.refs.auraContest = svgEl('circle', {
      id: 'outer-aura-purple-contest',
      class: 'mcw-aura-contest',
      cx: '512',
      cy: '104',
      r: '46',
      fill: '#B45CFF',
      opacity: '0.28',
      filter: 'url(#mcw-soft-blur)',
    });
    this.refs['outer-aura'].appendChild(this.refs.auraBlue);
    this.refs['outer-aura'].appendChild(this.refs.auraOrange);
    this.refs['outer-aura'].appendChild(this.refs.auraContest);
  }

  buildOuterEnergy() {
    const blue = svgEl('g', { id: 'outer-energy-streaks-blue', class: 'mcw-streaks mcw-streaks-blue' });
    const orange = svgEl('g', { id: 'outer-energy-streaks-orange', class: 'mcw-streaks mcw-streaks-orange' });
    for (let i = 0; i < 8; i++) {
      blue.appendChild(radialLine(455, 495, 224 + i * 8, '#38B2F6', 0.36));
      orange.appendChild(radialLine(455, 495, 44 - i * 8, '#F97316', 0.36));
    }
    this.refs['outer-energy-streaks'].appendChild(blue);
    this.refs['outer-energy-streaks'].appendChild(orange);
  }

  buildOuterSparks() {
    const blue = svgEl('g', { id: 'outer-sparks-blue', class: 'mcw-sparks mcw-sparks-blue' });
    const orange = svgEl('g', { id: 'outer-sparks-orange', class: 'mcw-sparks mcw-sparks-orange' });
    const purple = svgEl('g', { id: 'outer-sparks-purple', class: 'mcw-sparks mcw-sparks-purple' });
    const white = svgEl('g', { id: 'outer-sparks-white', class: 'mcw-sparks mcw-sparks-white' });
    appendSparkFixtures(blue, MOMENTUM_WHEEL_SPARKS.blue, 'blue');
    appendSparkFixtures(orange, MOMENTUM_WHEEL_SPARKS.orange, 'orange');
    appendSparkFixtures(purple, MOMENTUM_WHEEL_SPARKS.purple, 'purple');
    appendSparkFixtures(white, MOMENTUM_WHEEL_SPARKS.white, 'white');
    this.refs['outer-sparks'].appendChild(blue);
    this.refs['outer-sparks'].appendChild(orange);
    this.refs['outer-sparks'].appendChild(purple);
    this.refs['outer-sparks'].appendChild(white);
    this.refs.sparkGroups = [blue, orange, purple, white];
  }

  buildFrame() {
    this.refs['outer-mechanical-frame'].appendChild(svgEl('circle', {
      id: 'outer-frame-base',
      cx: '512',
      cy: '512',
      r: '410',
      fill: 'none',
      stroke: '#111827',
      'stroke-width': '28',
    }));
    this.refs['outer-mechanical-frame'].appendChild(svgEl('circle', {
      id: 'outer-frame-highlight',
      cx: '512',
      cy: '512',
      r: '425',
      fill: 'none',
      stroke: '#2B3340',
      'stroke-width': '4',
      opacity: '0.82',
    }));
    this.refs['outer-mechanical-frame'].appendChild(svgEl('circle', {
      id: 'outer-frame-shadow',
      cx: '512',
      cy: '512',
      r: '386',
      fill: 'none',
      stroke: '#050A12',
      'stroke-width': '7',
      opacity: '0.88',
    }));
    const panels = [
      ['outer-frame-panels-top', 'M448 74 H576 L610 116 L582 150 H442 L414 116 Z'],
      ['outer-frame-panels-bottom', 'M438 888 H586 L626 928 L590 968 H434 L398 928 Z'],
      ['outer-frame-panels-left', 'M86 430 L124 402 L142 420 L116 512 L142 604 L124 622 L86 594 Z'],
      ['outer-frame-panels-right', 'M938 430 L900 402 L882 420 L908 512 L882 604 L900 622 L938 594 Z'],
    ];
    for (const [id, d] of panels) {
      this.refs['outer-mechanical-frame'].appendChild(svgEl('path', {
        id,
        d,
        fill: '#08111F',
        stroke: '#2B3340',
        'stroke-width': '2',
        opacity: '0.82',
      }));
    }
    this.refs['outer-mechanical-frame'].appendChild(svgEl('path', {
      id: 'outer-frame-top-notch',
      d: 'M486 96 H538 L512 130 Z',
      fill: '#050A12',
      stroke: '#F5F7FB',
      'stroke-width': '3',
      opacity: '0.78',
    }));
    this.refs['outer-mechanical-frame'].appendChild(svgEl('path', {
      id: 'outer-frame-bottom-notch',
      d: 'M466 906 H558 L582 930 H442 Z',
      fill: '#07111C',
      stroke: '#5E7086',
      'stroke-width': '2',
      opacity: '0.72',
    }));
    for (const [id, d] of [
      ['outer-frame-slot-left-top', 'M176 224 L336 156 L352 172 L192 242 Z'],
      ['outer-frame-slot-right-top', 'M848 224 L688 156 L672 172 L832 242 Z'],
      ['outer-frame-slot-left-bottom', 'M176 800 L336 868 L352 852 L192 782 Z'],
      ['outer-frame-slot-right-bottom', 'M848 800 L688 868 L672 852 L832 782 Z'],
    ]) {
      this.refs['outer-mechanical-frame'].appendChild(svgEl('path', {
        id,
        class: 'mcw-frame-slot',
        d,
        fill: 'none',
        stroke: '#050A12',
        'stroke-width': '12',
        'stroke-linecap': 'round',
        opacity: '0.76',
      }));
    }
    const bolts = svgEl('g', { id: 'outer-frame-bolts' });
    for (const angle of [45, 135, 225, 315]) {
      const p = polar(512, 512, 422, angle);
      bolts.appendChild(svgEl('circle', {
        cx: n(p.x),
        cy: n(p.y),
        r: '5',
        fill: '#2B3340',
        stroke: '#9CA3AF',
        'stroke-width': '1',
        opacity: '0.72',
      }));
    }
    this.refs['outer-mechanical-frame'].appendChild(bolts);
  }

  buildSegments() {
    const underlay = this.refs['segment-ring-underlay'];
    const active = this.refs['segment-ring-active'];
    const blueGroup = svgEl('g', { id: 'segment-ring-blue-active' });
    const orangeGroup = svgEl('g', { id: 'segment-ring-orange-active' });
    const seamGroup = svgEl('g', { id: 'segment-ring-neutral-caps' });
    active.appendChild(blueGroup);
    active.appendChild(orangeGroup);
    active.appendChild(seamGroup);

    const bevels = this.refs['segment-ring-bevels'];
    const bevelGroup = svgEl('g', { id: 'segment-ring-bevel-overlays' });
    bevels.appendChild(svgEl('circle', {
      id: 'segment-ring-inner-shadow',
      cx: '512',
      cy: '512',
      r: '294',
      fill: 'none',
      stroke: '#050A12',
      'stroke-width': '10',
      opacity: '0.72',
    }));
    bevels.appendChild(svgEl('circle', {
      id: 'segment-ring-outer-highlight',
      cx: '512',
      cy: '512',
      r: '406',
      fill: 'none',
      stroke: '#F5F7FB',
      'stroke-width': '2',
      opacity: '0.16',
    }));
    bevels.appendChild(bevelGroup);

    this.segmentRefs = [];
    for (let i = 0; i < 96; i++) {
      const angle = i * 3.75;
      const common = segmentAttrs(angle);
      const inactive = svgEl('rect', {
        ...common,
        class: 'mcw-segment mcw-segment-inactive',
        opacity: '0.22',
      });
      const blue = svgEl('rect', {
        ...common,
        class: 'mcw-segment mcw-segment-blue',
        fill: 'url(#mcw-blue-segment-gradient)',
      });
      const orange = svgEl('rect', {
        ...common,
        class: 'mcw-segment mcw-segment-orange',
        fill: 'url(#mcw-orange-segment-gradient)',
      });
      const cap = svgEl('rect', {
        ...common,
        class: 'mcw-segment mcw-segment-cap',
        fill: '#F5F7FB',
        filter: 'url(#mcw-hot-glow)',
      });
      const bevel = svgEl('rect', {
        x: '507.5',
        y: '113',
        width: '3',
        height: '42',
        rx: '999',
        ry: '999',
        transform: `rotate(${n(angle)} 512 512)`,
        fill: '#F5F7FB',
        opacity: '0.28',
      });
      underlay.appendChild(inactive);
      blueGroup.appendChild(blue);
      orangeGroup.appendChild(orange);
      seamGroup.appendChild(cap);
      bevelGroup.appendChild(bevel);
      this.segmentRefs.push({ angle, blue, orange, cap });
    }
  }

  buildTicks() {
    const ring = this.refs['inner-tick-ring'];
    const base = svgEl('g', { id: 'inner-tick-ring-base' });
    const blue = svgEl('g', { id: 'inner-tick-ring-blue' });
    const orange = svgEl('g', { id: 'inner-tick-ring-orange' });
    const muted = svgEl('g', { id: 'inner-tick-ring-muted' });
    const crosshair = svgEl('g', { id: 'inner-crosshair-lines' });
    ring.appendChild(base);
    ring.appendChild(blue);
    ring.appendChild(orange);
    ring.appendChild(muted);
    ring.appendChild(crosshair);
    this.tickRefs = [];
    for (let i = 0; i < 120; i++) {
      const angle = i * 3;
      const line = radialLine(258, 274, angle, '#9CA3AF', 0.24, 1.5);
      muted.appendChild(line);
      this.tickRefs.push({ angle, line });
    }
    crosshair.appendChild(svgEl('line', { x1: '512', y1: '268', x2: '512', y2: '756', stroke: '#9CA3AF', 'stroke-width': '1', opacity: '0.1' }));
    crosshair.appendChild(svgEl('line', { x1: '268', y1: '512', x2: '756', y2: '512', stroke: '#9CA3AF', 'stroke-width': '1', opacity: '0.1' }));
  }

  buildCenter() {
    this.refs['center-disc'].appendChild(svgEl('circle', {
      id: 'center-disc-base',
      cx: '512',
      cy: '512',
      r: '230',
      fill: '#050A12',
      stroke: '#111827',
      'stroke-width': '12',
    }));
    this.refs['center-disc'].appendChild(svgEl('circle', {
      id: 'center-disc-inner-shadow',
      cx: '512',
      cy: '512',
      r: '210',
      fill: 'none',
      stroke: '#000',
      'stroke-width': '18',
      opacity: '0.34',
    }));

    this.refs.centerReactive = svgEl('g', {
      id: 'center-reactive-layer',
      class: 'mcw-center-reactive-layer',
      'clip-path': 'url(#mcw-center-disc-clip)',
    });
    this.refs.centerReactive.appendChild(svgEl('ellipse', {
      id: 'center-reactive-blue-field',
      cx: '440',
      cy: '512',
      rx: '124',
      ry: '214',
      fill: 'url(#mcw-blue-wash-gradient)',
    }));
    this.refs.centerReactive.appendChild(svgEl('ellipse', {
      id: 'center-reactive-orange-field',
      cx: '584',
      cy: '512',
      rx: '124',
      ry: '214',
      fill: 'url(#mcw-orange-wash-gradient)',
    }));
    this.refs.centerReactiveSeam = svgEl('line', {
      id: 'center-reactive-seam',
      x1: '512',
      y1: '314',
      x2: '512',
      y2: '710',
      stroke: '#F5F7FB',
      'stroke-width': '4',
      'stroke-linecap': 'round',
    });
    this.refs.centerReactive.appendChild(this.refs.centerReactiveSeam);
    this.refs['center-color-washes'].appendChild(this.refs.centerReactive);

    this.refs.blueWash = svgEl('circle', { id: 'center-disc-blue-wash', cx: '512', cy: '512', r: '230', fill: 'url(#mcw-blue-wash-gradient)', opacity: '0.12' });
    this.refs.orangeWash = svgEl('circle', { id: 'center-disc-orange-wash', cx: '512', cy: '512', r: '230', fill: 'url(#mcw-orange-wash-gradient)', opacity: '0.12' });
    this.refs.purpleWash = svgEl('circle', { id: 'center-disc-purple-contest-wash', cx: '512', cy: '418', r: '180', fill: 'url(#mcw-purple-wash-gradient)', opacity: '0' });
    this.refs['center-color-washes'].appendChild(this.refs.blueWash);
    this.refs['center-color-washes'].appendChild(this.refs.orangeWash);
    this.refs['center-color-washes'].appendChild(this.refs.purpleWash);

    this.refs['center-texture'].appendChild(svgEl('circle', {
      id: 'center-disc-honeycomb',
      cx: '512',
      cy: '512',
      r: '220',
      fill: 'url(#mcw-honeycomb-pattern)',
      opacity: '0.07',
    }));
    this.refs.waveform = svgEl('path', {
      id: 'center-telemetry-waveform',
      class: 'mcw-waveform',
      d: 'M384 512 L488 512 C500 498 524 526 536 512 L640 512',
      fill: 'none',
      stroke: '#9edcff',
      'stroke-width': '4',
      'stroke-linecap': 'round',
      opacity: '0.38',
    });
    this.refs['center-texture'].appendChild(this.refs.waveform);
    this.refs['center-rim'].appendChild(svgEl('ellipse', {
      id: 'center-disc-glass-highlight',
      cx: '512',
      cy: '410',
      rx: '140',
      ry: '44',
      fill: '#F5F7FB',
      opacity: '0.08',
    }));
    this.refs['center-rim'].appendChild(svgEl('circle', {
      id: 'center-disc-rim',
      cx: '512',
      cy: '512',
      r: '238',
      fill: 'none',
      stroke: '#F5F7FB',
      'stroke-width': '2',
      opacity: '0.22',
    }));
  }

  buildContestLine() {
    const g = this.refs['contested-front-line'];
    this.refs.contestCore = svgEl('circle', { id: 'contest-top-core', r: '12', fill: '#F5F7FB', filter: 'url(#mcw-hot-glow)' });
    this.refs.contestGlow = svgEl('circle', { id: 'contest-top-purple-glow', r: '48', fill: '#B45CFF', opacity: '0.34', filter: 'url(#mcw-soft-blur)' });
    this.refs.contestBeam = svgEl('line', { id: 'contest-top-vertical-beam', stroke: '#F0B8FF', 'stroke-width': '2', opacity: '0.66' });
    this.refs.contestCracks = svgEl('g', { id: 'contest-top-electric-cracks' });
    this.refs.originSeam = svgEl('circle', { id: 'contest-bottom-seam', r: '6', fill: '#F5F7FB', opacity: '0.22' });
    g.appendChild(this.refs.contestGlow);
    g.appendChild(this.refs.contestBeam);
    g.appendChild(this.refs.contestCracks);
    g.appendChild(this.refs.contestCore);
    g.appendChild(this.refs.originSeam);
  }

  buildText() {
    const text = this.refs['text-layer'];
    this.refs.time = svgEl('text', {
      id: 'text-time',
      x: '512',
      y: '448',
      'text-anchor': 'middle',
      'dominant-baseline': 'middle',
      class: 'mcw-text mcw-time',
    }, ['--:--']);
    this.refs.state = svgEl('text', {
      id: 'text-state',
      x: '512',
      y: '554',
      'text-anchor': 'middle',
      class: 'mcw-text mcw-state-label',
    }, ['NEUTRAL']);
    this.refs.confidenceLabel = svgEl('text', {
      id: 'text-confidence-label',
      x: '512',
      y: '616',
      'text-anchor': 'middle',
      class: 'mcw-text mcw-confidence-label',
    }, ['MOMENTUM']);
    this.refs.confidenceValue = svgEl('text', {
      id: 'text-confidence-value',
      x: '512',
      y: '658',
      'text-anchor': 'middle',
      class: 'mcw-text mcw-confidence-value',
    }, ['EVEN 50%']);
    text.appendChild(this.refs.time);
    text.appendChild(this.refs.state);
    text.appendChild(this.refs.confidenceLabel);
    text.appendChild(this.refs.confidenceValue);
  }

  buildBadge() {
    const badge = this.refs['oof-badge'];
    this.refs.badgeShell = svgEl('rect', {
      id: 'oof-badge-shell',
      x: '450',
      y: '872',
      width: '124',
      height: '44',
      rx: '14',
      fill: '#08111F',
      stroke: '#9CA3AF',
      'stroke-width': '1.5',
      opacity: '0.72',
    });
    this.refs.badgeFill = svgEl('rect', {
      id: 'oof-badge-fill',
      x: '454',
      y: '876',
      width: '116',
      height: '36',
      rx: '12',
      fill: '#050A12',
      opacity: '0.72',
    });
    this.refs.badgeBorder = svgEl('rect', {
      id: 'oof-badge-border',
      x: '450',
      y: '872',
      width: '124',
      height: '44',
      rx: '14',
      fill: 'none',
      stroke: '#F5F7FB',
      'stroke-width': '1',
      opacity: '0.2',
    });
    this.refs.badgeText = svgEl('text', {
      id: 'oof-badge-text',
      x: '512',
      y: '894',
      'text-anchor': 'middle',
      'dominant-baseline': 'middle',
      class: 'mcw-text mcw-badge-text',
    }, ['OOF']);
    badge.appendChild(this.refs.badgeShell);
    badge.appendChild(this.refs.badgeFill);
    badge.appendChild(this.refs.badgeBorder);
    badge.appendChild(this.refs.badgeText);
  }

  buildDebug() {
    const debug = this.refs['debug-overlays'];
    debug.setAttribute('opacity', '0.86');
    debug.style.display = 'none';
    this.refs.debugSeam = svgEl('line', {
      id: 'debug-seam-angle',
      x1: '512',
      y1: '512',
      x2: '512',
      y2: '88',
      stroke: '#F0B8FF',
      'stroke-width': '2',
      'stroke-dasharray': '8 8',
      opacity: '0.62',
    });
    this.refs.debugOrigin = svgEl('circle', {
      id: 'debug-origin-marker',
      cx: '512',
      cy: '898',
      r: '10',
      fill: 'none',
      stroke: '#F5F7FB',
      'stroke-width': '2',
    });
    this.refs.debugText = svgEl('text', {
      id: 'debug-wheel-readout',
      x: '512',
      y: '88',
      'text-anchor': 'middle',
      class: 'mcw-text mcw-debug-text',
    }, ['seam 0']);
    debug.appendChild(this.refs.debugSeam);
    debug.appendChild(this.refs.debugOrigin);
    debug.appendChild(this.refs.debugText);
  }

  updateTimeOnly(signal) {
    this.setText(this.refs.time, signal.time);
  }

  updateText(signal, stateConfig) {
    this.setText(this.refs.time, signal.time);
    this.setText(this.refs.state, stateConfig.stateText);
    this.setText(this.refs.confidenceLabel, 'MOMENTUM');
    this.setText(this.refs.confidenceValue, momentumShareDisplayLabel(signal));
    this.refs['oof-badge'].style.display = signal.showOOFBadge && this.config.showOOFBadge ? '' : 'none';
  }

  updateSegments(signal, seam) {
    overlayPerfCount('wheel.segmentLoop', this.segmentRefs.length);
    const seamBandDegrees = signal.state === 'volatile' ? 11.25 : 7.5;
    for (const ref of this.segmentRefs) {
      const owner = calculateSegmentOwnership(ref.angle, signal.bluePercent, signal.orangePercent);
      const nearSeam = circularDistance(ref.angle, seam) <= seamBandDegrees / 2;
      overlayPerfCountDisplayMutation(ref.blue, owner === 'blue' ? '' : 'none');
      overlayPerfCountDisplayMutation(ref.orange, owner === 'orange' ? '' : 'none');
      overlayPerfCountDisplayMutation(ref.cap, nearSeam ? '' : 'none');
      ref.blue.style.display = owner === 'blue' ? '' : 'none';
      ref.orange.style.display = owner === 'orange' ? '' : 'none';
      ref.cap.style.display = nearSeam ? '' : 'none';
      ref.blue.classList.toggle('is-low-confidence', signal.confidence === 'low');
      ref.orange.classList.toggle('is-low-confidence', signal.confidence === 'low');
    }
  }

  updateTicks(signal) {
    overlayPerfCount('wheel.tickLoop', this.tickRefs.length);
    for (const ref of this.tickRefs) {
      const owner = calculateSegmentOwnership(ref.angle, signal.bluePercent, signal.orangePercent);
      const stroke = owner === 'blue' ? '#38B2F6' : '#F97316';
      const opacity = signal.confidence === 'low' ? '0.22' : '0.30';
      if (overlayPerfAttrWillChange(ref.line, 'stroke', stroke)) overlayPerfCount('dom.attrMutation');
      if (overlayPerfAttrWillChange(ref.line, 'opacity', opacity)) overlayPerfCount('dom.attrMutation');
      ref.line.setAttribute('stroke', stroke);
      ref.line.setAttribute('opacity', opacity);
    }
  }

  updateEffects(signal, seam, reduced, performance, stateConfig) {
    const visual = this.config.momentumControlWheel.visual;
    const frontLine = visual.frontLine;
    const seamPoint = polar(512, 512, 396, seam);
    const seamAuraPoint = polar(512, 512, 430, seam);
    const beamStart = polar(512, 512, 246, seam);
    const beamEnd = polar(512, 512, 408, seam);
    const origin = polar(512, 512, 386, 180);
    if (performance || reduced) {
      removeSvgAttr(this.refs.contestCore, 'filter');
      removeSvgAttr(this.refs.contestGlow, 'filter');
    } else {
      this.refs.contestCore?.setAttribute('filter', 'url(#mcw-hot-glow)');
      this.refs.contestGlow?.setAttribute('filter', 'url(#mcw-soft-blur)');
    }
    setAttrs(this.refs.contestCore, { cx: n(seamPoint.x), cy: n(seamPoint.y), r: n(12 * frontLine.coreSize) });
    setAttrs(this.refs.contestGlow, {
      cx: n(seamPoint.x),
      cy: n(seamPoint.y),
      r: n(48 * frontLine.glowSize),
      opacity: n(stateConfig.contestOpacity * frontLine.opacity),
    });
    setAttrs(this.refs.contestBeam, {
      x1: n(beamStart.x),
      y1: n(beamStart.y),
      x2: n(beamEnd.x),
      y2: n(beamEnd.y),
      'stroke-width': n(1.5 + frontLine.trailStrength * 2.5),
    });
    setAttrs(this.refs.auraContest, { cx: n(seamAuraPoint.x), cy: n(seamAuraPoint.y) });
    setAttrs(this.refs.originSeam, { cx: n(origin.x), cy: n(origin.y) });
    this.refs.centerReactive?.setAttribute('transform', `rotate(${n(seam)} 512 512)`);
    for (const group of this.refs.sparkGroups || []) {
      group.setAttribute('transform', `rotate(${n(seam)} 512 512)`);
    }
    if (this.refs.debugSeam) {
      setAttrs(this.refs.debugSeam, { x1: '512', y1: '512', x2: n(seamPoint.x), y2: n(seamPoint.y) });
    }
    if (this.refs.debugText) {
      this.refs.debugText.textContent = `seam ${Math.round(seam)} / ${Math.round(signal.bluePercent)}-${Math.round(signal.orangePercent)}`;
    }

    const contestCracksHasChildren = Boolean(
      this.refs.contestCracks?.childNodes?.length
      || this.refs.contestCracks?.children?.length
      || this.refs.contestCracks?.innerHTML,
    );
    if (contestCracksHasChildren) {
      this.refs.contestCracks.innerHTML = '';
      overlayPerfCount('dom.innerHTMLClear');
    }
    if (!reduced && !performance && signal.state === 'volatile') {
      for (let i = 0; i < 4; i++) {
        const a = seam + (i - 1.5) * 5.5;
        const p1 = polar(512, 512, 408 + i * 3, a);
        const p2 = polar(512, 512, 434 + i * 5, a + (i % 2 ? 3 : -3));
        this.refs.contestCracks.appendChild(svgEl('path', {
          d: `M ${n(p1.x)} ${n(p1.y)} L ${n((p1.x + p2.x) / 2)} ${n((p1.y + p2.y) / 2 - 8)} L ${n(p2.x)} ${n(p2.y)}`,
          stroke: '#F0B8FF',
          'stroke-width': '2',
          fill: 'none',
          opacity: '0.7',
          filter: 'url(#mcw-hot-glow)',
        }));
        overlayPerfCount('dom.nodeAppend');
      }
    }

    const blueOpacity = stateConfig.blueWash;
    const orangeOpacity = stateConfig.orangeWash;
    const purpleOpacity = stateConfig.purpleWash;
    setAttrs(this.refs.blueWash, { opacity: blueOpacity });
    setAttrs(this.refs.orangeWash, { opacity: orangeOpacity });
    setAttrs(this.refs.purpleWash, { opacity: purpleOpacity });
    this.root.style.setProperty('--mcw-blue-wash-opacity', String(blueOpacity));
    this.root.style.setProperty('--mcw-orange-wash-opacity', String(orangeOpacity));
    this.root.style.setProperty('--mcw-purple-wash-opacity', String(purpleOpacity));

    const reactive = signal.state !== 'neutral' || signal.recentEventEnergy > 0.08;
    const sparks = signal.state !== 'neutral' || signal.recentEventEnergy > 0.2;
    this.refs['outer-sparks'].style.display = reduced || performance || !sparks ? 'none' : '';
    this.refs['outer-energy-streaks'].style.display = reduced || performance || !reactive ? 'none' : '';
  }

  setText(el, value) {
    if (el) el.textContent = value;
  }
}

window.MomentumFlowBarWidget = MomentumFlowBarWidget;
window.MomentumControlWheelWidget = MomentumControlWheel;
window.MomentumControlWheel = MomentumControlWheel;
window.MomentumControlWheelMath = {
  sanitizeWheelConfig,
  sanitizeWheelVisualConfig,
  sanitizeWheelResponseConfig,
  sanitizeMomentumWidgetConfig,
  sanitizeMomentumHostConfig,
  buildMomentumWheelPresetExport,
  parseMomentumWheelPresetJSON,
  clampNumber,
  isValidHexColor,
  sanitizeVariant,
  sanitizeTheme,
  sanitizeColorOverrides,
  normalizeMomentumControlWheelSignal,
  applyMomentumWheelDisplayResponse,
  normalizeMomentumWheelDisplayValues,
  momentumShareDisplayLabel,
  momentumWheelClamp,
  polar,
  angularDistanceClockwise,
  circularDistance,
  calculateSeamAngle,
  calculateSegmentOwnership,
  momentumWheelRecentEvent,
  momentumControlWheelSeamAngle,
  momentumControlWheelSegmentOwnership,
};
window.adaptMomentumFlowOutput = adaptMomentumFlowOutput;

window.pluginInit_overlay = () => {
  const prefs = loadMomentumFlowPrefs();
  const widgetConfig = mergeMomentumWidgetConfig(DEFAULT_MFB_CONFIG, prefs.widget);
  const hostConfig = { ...DEFAULT_HOST_CONFIG, ...prefs.host };
  const host = document.getElementById('momentum-flow-bar-widget');
  if (host && !_momentumFlowBar) {
    _momentumFlowBar = new MomentumFlowBarWidget(host, widgetConfig);
  } else {
    _momentumFlowBar?.setConfig(widgetConfig);
  }
  const wheelHost = document.getElementById('momentum-control-wheel-widget');
  if (wheelHost && !_momentumControlWheel) {
    _momentumControlWheel = new MomentumControlWheel(wheelHost, widgetConfig);
  } else {
    _momentumControlWheel?.setConfig(widgetConfig);
  }
  applyMomentumFlowPrefs(widgetConfig, hostConfig, { persist: false, push: false });
  if (!isOverlayHudWindow()) {
    queuePushMomentumFlowPrefs(widgetConfig, hostConfig, 0);
  }

  wireSelect('mfb-visual', value => updateMomentumFlowPrefs({ widget: { visual: value } }));
  wireSelect('mfb-variant', value => updateMomentumFlowPrefs({ widget: { variant: value } }));
  wireSelect('mfb-theme', value => updateMomentumFlowPrefs({ widget: { theme: value } }));
  wireSelect('mfb-position', value => updateMomentumFlowPrefs({ host: { position: value } }));
  wireRange('mfb-opacity', value => updateMomentumFlowPrefs({ host: { opacity: Number(value) } }));
  wireRange('mfb-scale', value => updateMomentumFlowPrefs({ host: { scale: Number(value) } }));
  wireCheck('mfb-show-confidence', checked => updateMomentumFlowPrefs({ widget: { showConfidence: checked } }));
  wireCheck('mfb-show-labels', checked => updateMomentumFlowPrefs({ widget: { showLabels: checked } }));
  wireCheck('mfb-show-percentages', checked => updateMomentumFlowPrefs({ widget: { showPercentages: checked } }));
  wireCheck('mfb-show-oof-badge', checked => updateMomentumFlowPrefs({ widget: { showOOFBadge: checked } }));
  wireCheck('mfb-reduced-motion', checked => updateMomentumFlowPrefs({ widget: { reducedMotion: checked } }));
  wireCheck('mfb-performance-mode', checked => updateMomentumFlowPrefs({ widget: { performanceMode: checked } }));
  wireCheck('mfb-pulse-enabled', checked => updateMomentumFlowPrefs({ widget: { pulseEnabled: checked } }));
  const performanceSafePreset = document.getElementById('mfb-performance-safe-preset');
  if (performanceSafePreset && !performanceSafePreset.dataset.wired) {
    performanceSafePreset.dataset.wired = '1';
    performanceSafePreset.addEventListener('click', () => {
      updateMomentumFlowPrefs({
        widget: {
          visual: 'wheel',
          variant: 'compact',
          theme: 'performance-safe',
          performanceMode: true,
          reducedMotion: false,
          showTelemetryWaveform: false,
          glowIntensity: 0.18,
          seamIntensity: 0.78,
          staticAuraIntensity: 0.12,
          volatileEffects: 0,
          dominantPulse: 0,
          momentumControlWheel: {
            visual: {
              segments: { brightness: 0.84, saturation: 0.88, glow: 0.08 },
              blueSegments: { brightness: 0.9, saturation: 0.88, glow: 0.08, opacity: 0.86 },
              orangeSegments: { brightness: 0.9, saturation: 0.88, glow: 0.08, opacity: 0.86 },
              inactiveSegments: { opacity: 0.22, brightness: 0.72 },
              seam: { intensity: 0.78, flare: 0.16, flicker: 0 },
              frontLine: { intensity: 0.78, coreSize: 0.9, glowSize: 0.62, opacity: 0.9, trailStrength: 0.06, trailDuration: 0.35 },
              aura: { intensity: 0.1, pulse: 0, pulseSpeed: 1, reactiveness: 0 },
              volatileAura: { intensity: 0.06, saturation: 0.85 },
              sparks: { intensity: 0, opacity: 0, reactiveness: 0 },
              seamSparks: { intensity: 0, opacity: 0 },
              outerSparks: { intensity: 0, opacity: 0 },
              centerWash: { intensity: 0.22, saturation: 0.85 },
              innerTicks: { opacity: 0.22, brightness: 0.9, saturation: 0.85 },
              frame: { brightness: 0.78, saturation: 0.85, opacity: 0.92 },
            },
            response: {
              pressureSensitivity: 0.75,
              volatilitySensitivity: 0.35,
              confidenceInfluence: 0.85,
              smoothing: 0.45,
              transitionSharpness: 0.5,
              eventReactiveness: 0,
              timing: {
                reactionSpeed: 1,
                holdDuration: 0.75,
                decaySpeed: 1.25,
                pulseDuration: 0.35,
                afterglowDuration: 0.35,
                eventBurstDuration: 0.35,
              },
            },
          },
        },
      });
      setConfigJSONStatus('Applied performance-safe visual preset.');
    });
  }
  wireRange('mfb-glow-intensity', value => updateMomentumFlowPrefs({
    widget: {
      glowIntensity: Number(value),
      momentumControlWheel: { visual: {
        segments: { glow: Number(value) },
        aura: { intensity: Number(value) },
      } },
    },
  }));
  wireRange('mfb-reactiveness', value => updateMomentumFlowPrefs({
    widget: { momentumControlWheel: { visual: {
      aura: { reactiveness: Number(value), pulse: Number(value) },
      sparks: { reactiveness: Number(value) },
    } } },
  }));
  wireRange('mfb-spark-intensity', value => updateMomentumFlowPrefs({
    widget: { momentumControlWheel: { visual: { sparks: { intensity: Number(value) } } } },
  }));
  wireRange('mfb-seam-intensity', value => updateMomentumFlowPrefs({
    widget: {
      seamIntensity: Number(value),
      momentumControlWheel: { visual: { seam: { intensity: Number(value) } } },
    },
  }));
  wireRange('mfb-volatile-effects', value => updateMomentumFlowPrefs({
    widget: {
      volatileEffects: Number(value),
      momentumControlWheel: { visual: { volatileAura: { intensity: Number(value) } } },
    },
  }));
  wireCheck('mfb-color-overrides-enabled', checked => updateMomentumFlowPrefs({ widget: { colorOverrides: { enabled: checked } } }));
  wireColor('mfb-color-blue', value => updateMomentumFlowPrefs({ widget: { colorOverrides: { blue: value } } }));
  wireColor('mfb-color-orange', value => updateMomentumFlowPrefs({ widget: { colorOverrides: { orange: value } } }));
  wireColor('mfb-color-frame', value => updateMomentumFlowPrefs({ widget: { colorOverrides: { frame: value } } }));
  wireColor('mfb-color-text', value => updateMomentumFlowPrefs({ widget: { colorOverrides: { text: value } } }));
  wireWheelVisualRangeControls();
  wireWheelResponseRangeControls();
  wireAdvancedVisualSections();

  document.querySelectorAll('[data-mfb-demo]').forEach(button => {
    if (button.dataset.wired) return;
    button.dataset.wired = '1';
    button.addEventListener('click', () => {
      _demoModeUntil = Date.now() + 4500;
      const out = demoOutput(button.dataset.mfbDemo);
      _lastMomentumOutput = out;
      renderOverlayMomentum(out);
    });
  });

  const reset = document.getElementById('momentum-reset');
  if (reset && !reset.dataset.wired) {
    reset.dataset.wired = '1';
    reset.addEventListener('click', async () => {
      await fetch('/api/overlay/momentum/reset', { method: 'POST' });
      await refreshOverlayMomentum();
    });
  }

  const resetStyle = document.getElementById('mfb-reset-style');
  if (resetStyle && !resetStyle.dataset.wired) {
    resetStyle.dataset.wired = '1';
    resetStyle.addEventListener('click', () => {
      resetMomentumFlowPrefs();
      setConfigJSONStatus('Reset visual defaults.');
    });
  }

  const copyConfig = document.getElementById('mcw-copy-config-json');
  if (copyConfig && !copyConfig.dataset.wired) {
    copyConfig.dataset.wired = '1';
    copyConfig.addEventListener('click', () => {
      copyCurrentOverlayConfigJSON();
    });
  }

  const exportConfig = document.getElementById('mcw-export-config-json');
  if (exportConfig && !exportConfig.dataset.wired) {
    exportConfig.dataset.wired = '1';
    exportConfig.addEventListener('click', () => {
      exportCurrentOverlayConfigJSON();
    });
  }

  const importConfig = document.getElementById('mcw-import-config-json');
  if (importConfig && !importConfig.dataset.wired) {
    importConfig.dataset.wired = '1';
    importConfig.addEventListener('click', () => {
      importOverlayConfigJSON();
    });
  }

  wireOverlayPerfControls();

  const clearTimeline = document.getElementById('momentum-timeline-clear');
  if (clearTimeline && !clearTimeline.dataset.wired) {
    clearTimeline.dataset.wired = '1';
    clearTimeline.addEventListener('click', () => {
      resetLocalTimeline();
      renderMomentumTimeline();
    });
  }

  clearInterval(_overlayMomentumTimer);
  refreshOverlayMomentum();
  _overlayMomentumTimer = setInterval(refreshOverlayMomentum, 1000);
};

function loadMomentumFlowPrefs() {
  try {
    const raw = window.localStorage?.getItem(MFB_PREFS_KEY);
    if (!raw) return { widget: {}, host: {} };
    const parsed = JSON.parse(raw);
    return {
      widget: parsed && typeof parsed.widget === 'object' ? parsed.widget : {},
      host: parsed && typeof parsed.host === 'object' ? parsed.host : {},
    };
  } catch {
    return { widget: {}, host: {} };
  }
}

function saveMomentumFlowPrefs(widgetConfig, hostConfig) {
  try {
    window.localStorage?.setItem(MFB_PREFS_KEY, JSON.stringify({
      widget: widgetConfig,
      host: hostConfig,
    }));
  } catch {
    // Local overlay preferences are optional; failing to persist should not
    // interrupt live HUD rendering.
  }
}

function mergeMomentumWidgetConfig(base = {}, patch = {}) {
  const baseWheel = base.momentumControlWheel || {};
  const patchWheel = patch.momentumControlWheel || {};
  return {
    ...base,
    ...patch,
    colorOverrides: {
      ...(base.colorOverrides || {}),
      ...(patch.colorOverrides || {}),
    },
    debugWheel: {
      ...(base.debugWheel || {}),
      ...(patch.debugWheel || {}),
    },
    momentumControlWheel: {
      ...baseWheel,
      ...patchWheel,
      visual: mergeWheelVisualConfig(baseWheel.visual, patchWheel.visual),
      response: mergeWheelResponseConfig(baseWheel.response, patchWheel.response),
    },
  };
}

function sanitizeMomentumWidgetConfig(input = {}) {
  const merged = mergeMomentumWidgetConfig(DEFAULT_MFB_CONFIG, input);
  const wheel = sanitizeWheelConfig({
    ...merged,
    reducedMotion: false,
    performanceMode: false,
  });
  return {
    ...merged,
    enabled: merged.enabled !== false,
    visual: wheel.visual,
    variant: wheel.variant,
    showConfidence: merged.showConfidence !== false,
    showLabels: merged.showLabels !== false,
    showPercentages: merged.showPercentages !== false,
    smoothTransitions: merged.smoothTransitions !== false,
    pulseEnabled: merged.pulseEnabled !== false,
    lowConfidenceDimThreshold: clampNumber(merged.lowConfidenceDimThreshold, 0, 1, DEFAULT_MFB_CONFIG.lowConfidenceDimThreshold),
    showOOFBadge: wheel.showOOFBadge,
    showStateLabel: wheel.showStateLabel,
    showTelemetryWaveform: wheel.showTelemetryWaveform,
    reducedMotion: Boolean(merged.reducedMotion),
    performanceMode: Boolean(merged.performanceMode),
    theme: wheel.theme,
    glowIntensity: wheel.glowIntensity,
    segmentBrightness: wheel.segmentBrightness,
    inactiveSegmentVisibility: wheel.inactiveSegmentVisibility,
    seamIntensity: wheel.seamIntensity,
    staticAuraIntensity: wheel.staticAuraIntensity,
    volatileEffects: wheel.volatileEffects,
    dominantPulse: wheel.dominantPulse,
    timerScale: wheel.timerScale,
    labelScale: wheel.labelScale,
    forceHighContrastText: wheel.forceHighContrastText,
    colorOverrides: wheel.colorOverrides,
    debugWheel: sanitizeDebugConfig(merged.debugWheel || merged.debug),
    momentumControlWheel: wheel.momentumControlWheel,
  };
}

function mergeWheelVisualConfig(base = {}, patch = {}) {
  const out = {};
  const source = { ...(base || {}), ...(patch || {}) };
  for (const key of Object.keys(source)) {
    const baseValue = base?.[key];
    const patchValue = patch?.[key];
    if (isPlainObject(baseValue) || isPlainObject(patchValue)) {
      out[key] = {
        ...(isPlainObject(baseValue) ? baseValue : {}),
        ...(isPlainObject(patchValue) ? patchValue : {}),
      };
    } else {
      out[key] = patchValue !== undefined ? patchValue : baseValue;
    }
  }
  return out;
}

function mergeWheelResponseConfig(base = {}, patch = {}) {
  return {
    ...(base || {}),
    ...(patch || {}),
    timing: {
      ...(base?.timing || {}),
      ...(patch?.timing || {}),
    },
  };
}

function isPlainObject(value) {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

function isOverlayHudWindow() {
  try {
    const params = new URLSearchParams(window.location?.search || '');
    return params.get('hud') === '1' || params.get('overlay') === '1' || document.body?.classList.contains('overlay-hud-mode');
  } catch {
    return false;
  }
}

function applyMomentumFlowPrefs(widgetConfig, hostConfig, options = {}) {
  _currentMomentumWidgetConfig = sanitizeMomentumWidgetConfig(mergeMomentumWidgetConfig(DEFAULT_MFB_CONFIG, widgetConfig || {}));
  _currentMomentumHostConfig = { ...DEFAULT_HOST_CONFIG, ...(hostConfig || {}) };
  _momentumFlowBar?.setConfig(_currentMomentumWidgetConfig);
  _momentumControlWheel?.setConfig(_currentMomentumWidgetConfig);
  updateMomentumVisualMode(_currentMomentumWidgetConfig.visual);
  applyHostConfig(_currentMomentumHostConfig);
  syncMomentumFlowControls(_currentMomentumWidgetConfig, _currentMomentumHostConfig);
  if (options.persist !== false) {
    saveMomentumFlowPrefs(_currentMomentumWidgetConfig, _currentMomentumHostConfig);
  }
  if (options.push) {
    queuePushMomentumFlowPrefs(_currentMomentumWidgetConfig, _currentMomentumHostConfig);
  }
}

function queuePushMomentumFlowPrefs(widgetConfig, hostConfig, delay = 120) {
  if (isOverlayHudWindow()) return;
  clearTimeout(_prefsPushTimer);
  _prefsPushTimer = setTimeout(() => {
    pushMomentumFlowPrefs(widgetConfig, hostConfig);
  }, delay);
}

async function pushMomentumFlowPrefs(widgetConfig, hostConfig) {
  if (isOverlayHudWindow()) return;
  try {
    const res = await fetch('/api/overlay/prefs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        widget: widgetConfig || {},
        host: hostConfig || {},
      }),
    });
    if (!res.ok) return;
    const prefs = await res.json();
    if (prefs?.updatedAt) _lastServerPrefsAt = Math.max(_lastServerPrefsAt, Number(prefs.updatedAt) || 0);
  } catch {
    // The lab preview still works from local preferences if the shared overlay
    // preference bridge is temporarily unavailable.
  }
}

function applyServerMomentumPrefs(prefs) {
  if (!prefs || typeof prefs !== 'object') return;
  const updatedAt = Number(prefs.updatedAt || 0);
  if (!updatedAt || updatedAt <= _lastServerPrefsAt) return;
  _lastServerPrefsAt = updatedAt;
  const widgetConfig = mergeMomentumWidgetConfig(DEFAULT_MFB_CONFIG, prefs.widget || {});
  const hostConfig = { ...DEFAULT_HOST_CONFIG, ...(prefs.host || {}) };
  applyMomentumFlowPrefs(widgetConfig, hostConfig, {
    persist: !isOverlayHudWindow(),
    push: false,
  });
}

async function refreshOverlayPrefsOnly() {
  try {
    overlayPerfCount('fetch.prefs');
    const prefs = await fetch('/api/overlay/prefs').then(r => (r.ok ? r.json() : null));
    applyServerMomentumPrefs(prefs);
  } catch {
    // Hidden F9 overlays may skip expensive momentum rendering, but preference
    // sync is best-effort so a transient fetch issue never breaks the overlay.
  }
}

function updateMomentumFlowPrefs(patch) {
  const baseWidget = _currentMomentumWidgetConfig || mergeMomentumWidgetConfig(DEFAULT_MFB_CONFIG, loadMomentumFlowPrefs().widget);
  const baseHost = _currentMomentumHostConfig || { ...DEFAULT_HOST_CONFIG, ...loadMomentumFlowPrefs().host };
  const widgetConfig = mergeMomentumWidgetConfig(baseWidget, patch.widget || {});
  const hostConfig = { ...DEFAULT_HOST_CONFIG, ...baseHost, ...(patch.host || {}) };
  applyMomentumFlowPrefs(widgetConfig, hostConfig, { persist: true, push: true });
}

function resetMomentumFlowPrefs() {
  const widgetConfig = mergeMomentumWidgetConfig({}, DEFAULT_MFB_CONFIG);
  const hostConfig = { ...DEFAULT_HOST_CONFIG };
  applyMomentumFlowPrefs(widgetConfig, hostConfig, { persist: true, push: true });
  if (_lastDisplayOutput) renderOverlayMomentum(_lastDisplayOutput);
}

function sanitizeMomentumHostConfig(input = {}) {
  const source = input && typeof input === 'object' ? input : {};
  const position = ['top-center', 'top-left', 'top-right', 'lower-center', 'side-left'].includes(source.position)
    ? source.position
    : DEFAULT_HOST_CONFIG.position;
  return {
    position,
    opacity: clampNumber(source.opacity, 0.35, 1, DEFAULT_HOST_CONFIG.opacity),
    scale: clampNumber(source.scale, 0.75, 1.25, DEFAULT_HOST_CONFIG.scale),
  };
}

function buildMomentumWheelPresetExport(widgetConfig, hostConfig, metadata = {}) {
  const widget = sanitizeMomentumWidgetConfig(mergeMomentumWidgetConfig(DEFAULT_MFB_CONFIG, widgetConfig || {}));
  const host = sanitizeMomentumHostConfig(hostConfig || {});
  return {
    version: 1,
    presetType: 'momentum-control-wheel',
    name: metadata.name || 'Custom Momentum Wheel',
    description: metadata.description || 'User-tuned MomentumControlWheel preset',
    exportedAt: new Date().toISOString(),
    widget: {
      visual: widget.visual,
      variant: widget.variant,
      theme: widget.theme,
      showConfidence: widget.showConfidence,
      showOOFBadge: widget.showOOFBadge,
      showStateLabel: widget.showStateLabel,
      reducedMotion: widget.reducedMotion,
      performanceMode: widget.performanceMode,
      glowIntensity: widget.glowIntensity,
      seamIntensity: widget.seamIntensity,
      volatileEffects: widget.volatileEffects,
      colorOverrides: widget.colorOverrides,
      momentumControlWheel: {
        visual: widget.momentumControlWheel.visual,
        response: widget.momentumControlWheel.response,
        timing: widget.momentumControlWheel.response.timing,
      },
    },
    host,
  };
}

function parseMomentumWheelPresetJSON(raw) {
  const parsed = JSON.parse(raw);
  if (!parsed || typeof parsed !== 'object') throw new Error('Preset JSON must be an object.');
  if (parsed.presetType && parsed.presetType !== 'momentum-control-wheel') {
    throw new Error('Preset type is not momentum-control-wheel.');
  }
  const sourceWidget = parsed.widget && typeof parsed.widget === 'object' ? parsed.widget : {};
  const sourceWheel = sourceWidget.momentumControlWheel && typeof sourceWidget.momentumControlWheel === 'object'
    ? sourceWidget.momentumControlWheel
    : {};
  const response = mergeWheelResponseConfig(sourceWheel.response || {}, sourceWheel.timing ? { timing: sourceWheel.timing } : {});
  const widget = sanitizeMomentumWidgetConfig(mergeMomentumWidgetConfig(DEFAULT_MFB_CONFIG, {
    ...sourceWidget,
    momentumControlWheel: {
      ...sourceWheel,
      response,
    },
  }));
  const host = sanitizeMomentumHostConfig(parsed.host || {});
  return { widget, host };
}

function currentMomentumWheelPresetJSON() {
  const widget = _currentMomentumWidgetConfig || mergeMomentumWidgetConfig(DEFAULT_MFB_CONFIG, loadMomentumFlowPrefs().widget);
  const host = _currentMomentumHostConfig || { ...DEFAULT_HOST_CONFIG, ...loadMomentumFlowPrefs().host };
  return JSON.stringify(buildMomentumWheelPresetExport(widget, host), null, 2);
}

function setConfigJSONStatus(message) {
  const el = document.getElementById('mcw-config-json-status');
  if (el) el.textContent = message || '';
}

async function copyCurrentOverlayConfigJSON() {
  const json = currentMomentumWheelPresetJSON();
  try {
    await navigator.clipboard?.writeText(json);
    setConfigJSONStatus('Copied sanitized MomentumControlWheel preset JSON.');
  } catch {
    const area = document.createElement('textarea');
    area.value = json;
    area.style.position = 'fixed';
    area.style.left = '-9999px';
    document.body.appendChild(area);
    area.select();
    document.execCommand('copy');
    document.body.removeChild(area);
    setConfigJSONStatus('Copied sanitized MomentumControlWheel preset JSON.');
  }
  return json;
}

function exportCurrentOverlayConfigJSON() {
  const json = currentMomentumWheelPresetJSON();
  const blob = new Blob([json], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `oof-rl-momentum-control-wheel-${new Date().toISOString().slice(0, 10)}.json`;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
  setConfigJSONStatus('Exported sanitized MomentumControlWheel preset JSON.');
  return json;
}

function importOverlayConfigJSON() {
  const raw = window.prompt('Paste MomentumControlWheel preset JSON. Display config only; values are sanitized before applying.');
  if (!raw) return;
  try {
    const parsed = parseMomentumWheelPresetJSON(raw);
    applyMomentumFlowPrefs(parsed.widget, parsed.host, { persist: true, push: true });
    if (_lastDisplayOutput) renderOverlayMomentum(_lastDisplayOutput);
    setConfigJSONStatus('Imported sanitized MomentumControlWheel preset JSON.');
  } catch (err) {
    setConfigJSONStatus(`Import failed: ${err.message || err}`);
  }
}

function applyHostConfig(config = DEFAULT_HOST_CONFIG) {
  const host = document.getElementById('overlay-widget-host');
  if (!host) return;
  const position = ['top-center', 'top-left', 'top-right', 'lower-center', 'side-left'].includes(config.position)
    ? config.position
    : DEFAULT_HOST_CONFIG.position;
  host.className = `overlay-widget-host overlay-widget-host--${position}`;
  host.style.setProperty('--mfb-host-opacity', String(clampRange(config.opacity, 0.35, 1, 1)));
  host.style.setProperty('--mfb-host-scale', String(clampRange(config.scale, 0.75, 1.25, 1)));
}

function syncMomentumFlowControls(widgetConfig, hostConfig) {
  setControlValue('mfb-visual', widgetConfig.visual);
  setControlValue('mfb-variant', widgetConfig.variant);
  setControlValue('mfb-theme', widgetConfig.theme);
  setControlValue('mfb-position', hostConfig.position);
  setControlValue('mfb-opacity', hostConfig.opacity);
  setControlValue('mfb-scale', hostConfig.scale);
  setControlChecked('mfb-show-confidence', widgetConfig.showConfidence);
  setControlChecked('mfb-show-labels', widgetConfig.showLabels);
  setControlChecked('mfb-show-percentages', widgetConfig.showPercentages);
  setControlChecked('mfb-show-oof-badge', widgetConfig.showOOFBadge);
  setControlChecked('mfb-reduced-motion', widgetConfig.reducedMotion);
  setControlChecked('mfb-performance-mode', widgetConfig.performanceMode);
  setControlChecked('mfb-pulse-enabled', widgetConfig.pulseEnabled);
  setControlValue('mfb-glow-intensity', widgetConfig.glowIntensity);
  const visual = sanitizeWheelVisualConfig(widgetConfig.momentumControlWheel?.visual);
  const response = sanitizeWheelResponseConfig(widgetConfig.momentumControlWheel?.response);
  setControlValue('mfb-reactiveness', visual.aura.reactiveness);
  setControlValue('mfb-spark-intensity', visual.sparks.intensity);
  setControlValue('mfb-seam-intensity', widgetConfig.seamIntensity);
  setControlValue('mfb-volatile-effects', widgetConfig.volatileEffects);
  setControlChecked('mfb-color-overrides-enabled', widgetConfig.colorOverrides?.enabled);
  setControlValue('mfb-color-blue', widgetConfig.colorOverrides?.blue || MOMENTUM_WHEEL_THEMES['oof-default'].blueCore);
  setControlValue('mfb-color-orange', widgetConfig.colorOverrides?.orange || MOMENTUM_WHEEL_THEMES['oof-default'].orangeCore);
  setControlValue('mfb-color-frame', widgetConfig.colorOverrides?.frame || MOMENTUM_WHEEL_THEMES['oof-default'].frame);
  setControlValue('mfb-color-text', widgetConfig.colorOverrides?.text || MOMENTUM_WHEEL_THEMES['oof-default'].textPrimary);
  for (const [id, path] of WHEEL_VISUAL_RANGE_CONTROLS) {
    setControlValue(id, wheelVisualValue(visual, path));
  }
  for (const [id, path] of WHEEL_RESPONSE_RANGE_CONTROLS) {
    setControlValue(id, wheelResponseValue(response, path));
  }
}

function updateMomentumVisualMode(visual) {
  const bar = document.getElementById('momentum-flow-bar-widget');
  const wheel = document.getElementById('momentum-control-wheel-widget');
  const showWheel = visual === 'wheel';
  if (bar) {
    bar.hidden = showWheel;
    bar.style.display = showWheel ? 'none' : '';
    bar.setAttribute('aria-hidden', showWheel ? 'true' : 'false');
  }
  if (wheel) {
    wheel.hidden = !showWheel;
    wheel.style.display = showWheel ? '' : 'none';
    wheel.setAttribute('aria-hidden', showWheel ? 'false' : 'true');
  }
}

function isOverlayLabViewActive() {
  if (isOverlayHudWindow()) return true;
  const root = document.getElementById('view-overlay');
  return Boolean(root?.classList.contains('active') || window.oofActiveViewName === 'overlay');
}

function isOverlayLabPreviewPaused() {
  // Only the Lab preview may pause itself. The F9 HUD must stay live whenever
  // its webview exists; native show/hide state is not a safe render gate.
  if (isOverlayHudWindow()) return false;
  if (document.hidden) return true;
  return !isOverlayLabViewActive();
}

function applyOverlayLabPreviewPaused(paused) {
  const root = document.getElementById('view-overlay');
  if (!root || isOverlayHudWindow()) return;
  root.dataset.previewPaused = paused ? 'true' : 'false';
  const host = document.getElementById('overlay-widget-host');
  if (host) host.dataset.previewPaused = paused ? 'true' : 'false';
}

function wireSelect(id, onChange) {
  const el = document.getElementById(id);
  if (!el || el.dataset.wired) return;
  el.dataset.wired = '1';
  el.addEventListener('change', () => onChange(el.value));
}

function wireRange(id, onChange) {
  const el = document.getElementById(id);
  if (!el || el.dataset.wired) return;
  el.dataset.wired = '1';
  el.addEventListener('input', () => onChange(el.value));
}

function wireCheck(id, onChange) {
  const el = document.getElementById(id);
  if (!el || el.dataset.wired) return;
  el.dataset.wired = '1';
  el.addEventListener('change', () => onChange(Boolean(el.checked)));
}

function wireColor(id, onChange) {
  const el = document.getElementById(id);
  if (!el || el.dataset.wired) return;
  el.dataset.wired = '1';
  el.addEventListener('input', () => onChange(el.value));
}

function wireWheelVisualRangeControls() {
  for (const [id, path] of WHEEL_VISUAL_RANGE_CONTROLS) {
    wireRange(id, value => updateMomentumFlowPrefs({
      widget: {
        momentumControlWheel: {
          visual: wheelVisualPatch(path, Number(value)),
        },
      },
    }));
  }
}

function wireWheelResponseRangeControls() {
  for (const [id, path] of WHEEL_RESPONSE_RANGE_CONTROLS) {
    wireRange(id, value => updateMomentumFlowPrefs({
      widget: {
        momentumControlWheel: {
          response: wheelResponsePatch(path, Number(value)),
        },
      },
    }));
  }
}

function wireAdvancedVisualSections() {
  document.querySelectorAll('[data-mcw-advanced-section]').forEach(section => {
    if (section.dataset.wired) return;
    section.dataset.wired = '1';
    section.addEventListener('toggle', () => {
      if (!section.open) return;
      section.parentElement?.querySelectorAll('[data-mcw-advanced-section]').forEach(other => {
        if (other !== section) other.open = false;
      });
    });
  });
}

function wheelVisualPatch(path, value) {
  if (!Array.isArray(path) || path.length !== 2) return {};
  return {
    [path[0]]: {
      [path[1]]: value,
    },
  };
}

function wheelVisualValue(visual, path) {
  if (!visual || !Array.isArray(path) || path.length !== 2) return '';
  return visual[path[0]]?.[path[1]];
}

function wheelResponsePatch(path, value) {
  if (!Array.isArray(path) || path.length < 1 || path.length > 2) return {};
  if (path.length === 1) return { [path[0]]: value };
  return {
    [path[0]]: {
      [path[1]]: value,
    },
  };
}

function wheelResponseValue(response, path) {
  if (!response || !Array.isArray(path) || path.length < 1 || path.length > 2) return '';
  if (path.length === 1) return response[path[0]];
  return response[path[0]]?.[path[1]];
}

function setControlValue(id, value) {
  const el = document.getElementById(id);
  if (el && value !== undefined && value !== null) el.value = String(value);
}

function setControlChecked(id, checked) {
  const el = document.getElementById(id);
  if (el) el.checked = Boolean(checked);
}

function createOverlayPerfState() {
  return {
    enabled: false,
    clientId: overlayPerfClientId(),
    windowStart: 0,
    current: {},
    previous: {},
    totals: {},
    lastSignalKey: '',
    lastPostAt: 0,
  };
}

function overlayPerfPageMeta() {
  try {
    const params = new URLSearchParams(window.location?.search || '');
    const isHud = params.get('hud') === '1' || params.get('overlay') === '1';
    const nativeHud = isHud && params.get('nativeHud') === '1';
    const assetVersion = params.get('assetVersion') || '';
    let clientClass = 'overlay-lab-preview';
    if (isHud && nativeHud && assetVersion) {
      clientClass = 'native-f9-hud';
    } else if (isHud && OVERLAY_PERF_SCHEMA_VERSION < 2) {
      clientClass = 'legacy-hud-client';
    } else if (isHud) {
      clientClass = 'manual-hud-url';
    }
    return { isHud, nativeHud, assetVersion, clientClass };
  } catch {
    return { isHud: false, nativeHud: false, assetVersion: '', clientClass: 'unknown-client' };
  }
}

function overlayPerfClientId() {
  const meta = overlayPerfPageMeta();
  const role = meta.isHud ? 'hud' : 'lab';
  try {
    return `${role}-${Math.random().toString(36).slice(2, 10)}-${Date.now().toString(36)}`;
  } catch {
    return `${role}-${Date.now().toString(36)}`;
  }
}

function overlayPerfFlagEnabled() {
  try {
    if (window.__OOF_OVERLAY_PERF_ENABLED__ === true) return true;
    const params = new URLSearchParams(window.location?.search || '');
    if (params.get('perf') === '1' || params.get('overlayPerf') === '1') return true;
    return window.localStorage?.getItem(OVERLAY_PERF_KEY) === '1';
  } catch {
    return false;
  }
}

function setOverlayPerfEnabled(enabled) {
  _overlayPerf.enabled = Boolean(enabled);
  if (_overlayPerf.enabled && !_overlayPerf.windowStart) {
    overlayPerfRotate(Date.now());
  }
}

function overlayPerfEnabled() {
  return Boolean(_overlayPerf.enabled || overlayPerfFlagEnabled());
}

function overlayPerfRotate(now = Date.now()) {
  const windowStart = Math.floor(now / 1000) * 1000;
  if (!_overlayPerf.windowStart) {
    _overlayPerf.windowStart = windowStart;
  }
  if (windowStart === _overlayPerf.windowStart) return;
  _overlayPerf.previous = { ..._overlayPerf.current };
  _overlayPerf.current = {};
  _overlayPerf.windowStart = windowStart;
}

function overlayPerfCount(key, amount = 1) {
  if (!overlayPerfEnabled() || !key || amount <= 0) return;
  overlayPerfRotate(Date.now());
  _overlayPerf.current[key] = (_overlayPerf.current[key] || 0) + amount;
  _overlayPerf.totals[key] = (_overlayPerf.totals[key] || 0) + amount;
}

function overlayPerfCountDisplayMutation(el, nextDisplay) {
  if (!overlayPerfEnabled() || !el) return;
  if ((el.style?.display || '') !== nextDisplay) overlayPerfCount('dom.displayMutation');
}

function overlayPerfAttrWillChange(el, attrName, value) {
  if (!overlayPerfEnabled() || !el || typeof el.getAttribute !== 'function') return false;
  return el.getAttribute(attrName) !== String(value);
}

function overlayPerfVisibilityState() {
  const meta = overlayPerfPageMeta();
  const isHud = meta.isHud || isOverlayHudWindow();
  const documentHidden = Boolean(document.hidden);
  const visibilityState = String(document.visibilityState || 'unknown');
  const windowFocused = typeof document.hasFocus === 'function' ? document.hasFocus() : false;
  const viewActive = isOverlayLabViewActive();
  const previewPaused = isOverlayLabPreviewPaused();
  const renderActive = isHud ? !documentHidden : !previewPaused;
  const hudVisibleGuess = isHud && !documentHidden;

  let perfRole = isHud ? (meta.nativeHud ? 'f9-hud-window' : 'legacy-or-manual-hud') : 'overlay-lab-preview';
  let perfStatus = renderActive ? 'render-active' : 'render-paused';
  if (isHud && documentHidden) {
    perfStatus = 'hud-document-hidden';
  } else if (!isHud && previewPaused) {
    perfStatus = 'lab-preview-paused';
  } else if (!isHud && viewActive) {
    perfStatus = 'lab-preview-active';
  }

  return {
    isHud,
    previewPaused,
    documentHidden,
    visibilityState,
    windowFocused,
    viewActive,
    renderActive,
    hudVisibleGuess,
    perfRole,
    perfStatus,
    visibilitySource: isHud
      ? (meta.nativeHud ? 'native-hud-query' : 'hud-query-missing-native-marker')
      : 'document-and-active-view',
    nativeHud: Boolean(meta.nativeHud),
    assetVersion: meta.assetVersion,
    clientClass: meta.clientClass,
  };
}

function overlayPerfSnapshot() {
  overlayPerfRotate(Date.now());
  const widget = _currentMomentumWidgetConfig || {};
  const visibility = overlayPerfVisibilityState();
  const visualDOM = overlayPerfVisualDOMState();
  return {
    clientId: _overlayPerf.clientId,
    perfSchemaVersion: OVERLAY_PERF_SCHEMA_VERSION,
    at: Date.now(),
    ...visibility,
    visual: widget.visual || 'unknown',
    variant: widget.variant || 'unknown',
    performanceMode: Boolean(widget.performanceMode),
    reducedMotion: Boolean(widget.reducedMotion),
    ...visualDOM,
    currentSecond: { ..._overlayPerf.current },
    previousSecond: { ..._overlayPerf.previous },
    totals: { ..._overlayPerf.totals },
    lastSignalKey: _overlayPerf.lastSignalKey,
    nodeCount: overlayPerfNodeCount(),
    url: String(window.location?.href || ''),
  };
}

function overlayPerfNodeCount() {
  try {
    return document.querySelectorAll('#momentum-control-wheel *').length;
  } catch {
    return 0;
  }
}

function overlayPerfVisualDOMState() {
  const state = {
    barHidden: true,
    wheelHidden: true,
    barDisplay: 'missing',
    wheelDisplay: 'missing',
    barNodes: 0,
    wheelNodes: 0,
  };
  try {
    const bar = document.getElementById('momentum-flow-bar-widget');
    const wheel = document.getElementById('momentum-control-wheel-widget');
    if (bar) {
      state.barHidden = Boolean(bar.hidden);
      state.barDisplay = window.getComputedStyle ? window.getComputedStyle(bar).display : (bar.style?.display || '');
      state.barNodes = bar.querySelectorAll('*').length;
    }
    if (wheel) {
      state.wheelHidden = Boolean(wheel.hidden);
      state.wheelDisplay = window.getComputedStyle ? window.getComputedStyle(wheel).display : (wheel.style?.display || '');
      state.wheelNodes = wheel.querySelectorAll('*').length;
    }
  } catch {
    // Optional perf diagnostics should never interrupt overlay rendering.
  }
  return state;
}

function momentumWheelPerfSignalKey(signal = {}) {
  return [
    signal.state || '',
    Math.round(Number(signal.bluePercent || 0) * 10) / 10,
    Math.round(Number(signal.orangePercent || 0) * 10) / 10,
    signal.confidence || '',
    Math.round(Number(signal.volatility || 0) * 100) / 100,
    signal.recentEventType || '',
    signal.recentEventTeam || '',
    Math.round(Number(signal.recentEventEnergy || 0) * 100) / 100,
  ].join('|');
}

function pushOverlayPerfIfDue(force = false) {
  if (!overlayPerfEnabled() || typeof fetch !== 'function') return;
  const now = Date.now();
  if (!force && now - _overlayPerf.lastPostAt < 1000) return;
  _overlayPerf.lastPostAt = now;
  fetch('/api/overlay/perf/frontend', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(overlayPerfSnapshot()),
  }).catch(() => {
    // Perf reporting is optional and must never interrupt overlay rendering.
  });
}

function unregisterOverlayPerfClient() {
  if (!_overlayPerf.clientId) return;
  const meta = overlayPerfPageMeta();
  const payload = {
    clientId: _overlayPerf.clientId,
    unregister: true,
    perfSchemaVersion: OVERLAY_PERF_SCHEMA_VERSION,
    at: Date.now(),
    isHud: meta.isHud,
    nativeHud: meta.nativeHud,
    assetVersion: meta.assetVersion,
    clientClass: meta.clientClass,
    url: String(window.location?.href || ''),
  };
  const body = JSON.stringify(payload);
  try {
    if (typeof navigator !== 'undefined' && typeof navigator.sendBeacon === 'function' && typeof Blob === 'function') {
      const blob = new Blob([body], { type: 'application/json' });
      navigator.sendBeacon('/api/overlay/perf/frontend', blob);
      return;
    }
  } catch {
    // Fall through to fetch keepalive below.
  }
  try {
    if (typeof fetch === 'function') {
      fetch('/api/overlay/perf/frontend', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body,
        keepalive: true,
      }).catch(() => {});
    }
  } catch {
    // Perf unregister is best-effort only.
  }
}

function wireOverlayPerfUnloadUnregister() {
  if (_overlayPerf.unloadWired || typeof window.addEventListener !== 'function') return;
  _overlayPerf.unloadWired = true;
  window.addEventListener('pagehide', unregisterOverlayPerfClient);
  window.addEventListener('beforeunload', unregisterOverlayPerfClient);
}

wireOverlayPerfUnloadUnregister();

window.OOFOverlayPerf = {
  enable() {
    try { window.localStorage?.setItem(OVERLAY_PERF_KEY, '1'); } catch {}
    setOverlayPerfEnabled(true);
    return overlayPerfSnapshot();
  },
  disable() {
    try { window.localStorage?.removeItem(OVERLAY_PERF_KEY); } catch {}
    setOverlayPerfEnabled(false);
    return overlayPerfSnapshot();
  },
  reset() {
    _overlayPerf.current = {};
    _overlayPerf.previous = {};
    _overlayPerf.totals = {};
    _overlayPerf.windowStart = 0;
    _overlayPerf.lastSignalKey = '';
    _overlayPerf.lastPostAt = 0;
    return overlayPerfSnapshot();
  },
  snapshot: overlayPerfSnapshot,
};

function setOverlayPerfStatus(message, kind = '') {
  const el = document.getElementById('overlay-perf-status');
  if (!el) return;
  el.textContent = message || '';
  el.classList.toggle('overlaylab-status--ok', kind === 'ok');
  el.classList.toggle('overlaylab-status--warn', kind === 'warn');
}

function setOverlayPerfDebugText(message) {
  const el = document.getElementById('overlay-perf-debug');
  if (el) el.textContent = message || '';
}

function overlayPerfCounter(map, key) {
  return Number(map?.[key] || 0);
}

function overlayPerfCounterLine(map, keys) {
  return keys.map(key => `${key}: ${overlayPerfCounter(map, key)}`).join(' | ');
}

function overlayPerfPercent(part, total) {
  if (!total) return '0%';
  return `${Math.round((part / total) * 100)}%`;
}

function formatOverlayPerfSnapshot(snapshot) {
  if (!snapshot || typeof snapshot !== 'object') return 'No perf snapshot available.';
  const backendPrevious = snapshot.previousSecond || {};
  const backendTotals = snapshot.totals || {};
  const frontend = snapshot.frontend && typeof snapshot.frontend === 'object' ? snapshot.frontend : {};
  const lines = [
    `Enabled: ${snapshot.enabled ? 'yes' : 'no'}`,
    `Player cache: ${snapshot.playerCacheGuid || 'none'} (${snapshot.playerCacheEntries || 0} entries)`,
    '',
    'Backend previous second:',
    overlayPerfCounterLine(backendPrevious, [
      'oofevents.state.updated',
      'oofevents.ball.hit',
      'oofevents.stat.feed',
      'oofevents.goal.scored',
      'oofevents.clock.updated',
      'normalized.ball_hit',
      'normalized.shot',
      'cache.miss.ballHit',
    ]),
    '',
    'Backend totals:',
    overlayPerfCounterLine(backendTotals, [
      'oofevents.state.updated',
      'oofevents.ball.hit',
      'oofevents.stat.feed',
      'oofevents.goal.scored',
      'oofevents.clock.updated',
      'normalized.ball_hit',
      'normalized.shot',
      'cache.miss.ballHit',
      'dedupe.explicit.goal',
    ]),
    '',
    'Frontend reports:',
  ];

  const reports = Object.values(frontend);
  if (!reports.length) {
    lines.push('No frontend reports yet. Keep counters enabled and wait for the overlay momentum poll.');
  }
  for (const report of reports) {
    const previous = report.previousSecond || {};
    const totals = report.totals || {};
    const totalUpdates = overlayPerfCounter(totals, 'wheel.update');
    const totalDuplicates = overlayPerfCounter(totals, 'wheel.duplicateSignal');
    const role = report.perfRole || (report.isHud ? 'f9-hud-window' : 'overlay-lab-preview');
    const status = report.perfStatus || (report.previewPaused ? 'render-paused' : 'render-active');
    const clientClass = report.clientClass || (report.isHud ? 'legacy-hud-client' : 'overlay-lab-preview');
    const visibility = report.visibilityState || 'unknown';
    const focus = report.windowFocused ? 'focused' : 'not-focused';
    const renderState = report.renderActive ? 'rendering' : 'not-rendering';
    const hiddenState = report.documentHidden ? 'hidden' : 'visible';
    const schema = report.perfSchemaVersion ? `schema v${report.perfSchemaVersion}` : 'schema unknown/old';
    const hudGuess = report.isHud
      ? ` | hudVisibleGuess: ${report.hudVisibleGuess ? 'yes' : 'no'}`
      : '';
    lines.push(
      '',
      `${report.isHud ? 'F9 HUD' : 'Lab/Preview'} ${report.clientId || 'unknown'} | ${schema} | ${clientClass} | ${role} / ${status} | ${report.visual || 'unknown'} / ${report.variant || 'unknown'} | nodes: ${report.nodeCount || 0}`,
      `visibility: ${visibility} (${hiddenState}) | ${focus} | ${renderState} | viewActive: ${report.viewActive ? 'yes' : 'no'} | previewPaused: ${report.previewPaused ? 'yes' : 'no'}${hudGuess}`,
      `visual DOM: bar hidden=${report.barHidden ? 'yes' : 'no'} display=${report.barDisplay || 'unknown'} nodes=${report.barNodes || 0} | wheel hidden=${report.wheelHidden ? 'yes' : 'no'} display=${report.wheelDisplay || 'unknown'} nodes=${report.wheelNodes || 0}`,
      `previous second: ${overlayPerfCounterLine(previous, ['lab.previewPaused', 'fetch.prefs', 'wheel.update', 'wheel.duplicateSignal', 'wheel.timerOnlyUpdate', 'wheel.skippedDuplicate', 'dom.attrMutation', 'dom.displayMutation', 'wheel.segmentLoop', 'wheel.tickLoop'])}`,
      `totals: ${overlayPerfCounterLine(totals, ['lab.previewPaused', 'fetch.prefs', 'wheel.update', 'wheel.duplicateSignal', 'wheel.timerOnlyUpdate', 'wheel.skippedDuplicate', 'dom.attrMutation', 'dom.displayMutation'])}`,
      `duplicate signal share: ${overlayPerfPercent(totalDuplicates, totalUpdates)}`,
    );
  }

  lines.push('', 'Raw JSON is available through Copy Snapshot JSON.');
  return lines.join('\n');
}

async function requestOverlayPerfSnapshot(query = '') {
  const suffix = query ? `?${query}` : '';
  const snapshot = await fetch(`/api/overlay/perf${suffix}`).then(r => {
    if (!r.ok) throw new Error(`perf request failed (${r.status})`);
    return r.json();
  });
  _lastOverlayPerfSnapshot = snapshot;
  setOverlayPerfDebugText(formatOverlayPerfSnapshot(snapshot));
  return snapshot;
}

async function enableOverlayPerfCapture() {
  try {
    window.OOFOverlayPerf?.enable?.();
    const snapshot = await requestOverlayPerfSnapshot('enable=1');
    pushOverlayPerfIfDue(true);
    setOverlayPerfStatus('Perf counters enabled. Run a scenario for 30-60 seconds, then refresh snapshot.', 'ok');
    return snapshot;
  } catch (err) {
    setOverlayPerfStatus(`Enable failed: ${err.message || err}`, 'warn');
    return null;
  }
}

async function resetOverlayPerfCapture() {
  try {
    window.OOFOverlayPerf?.reset?.();
    const snapshot = await requestOverlayPerfSnapshot('reset=1');
    pushOverlayPerfIfDue(true);
    setOverlayPerfStatus('Perf counters reset. Start the scenario timer now.', 'ok');
    return snapshot;
  } catch (err) {
    setOverlayPerfStatus(`Reset failed: ${err.message || err}`, 'warn');
    return null;
  }
}

async function disableOverlayPerfCapture() {
  try {
    const snapshot = await requestOverlayPerfSnapshot('disable=1');
    window.OOFOverlayPerf?.disable?.();
    setOverlayPerfStatus('Perf counters disabled.', 'ok');
    return snapshot;
  } catch (err) {
    setOverlayPerfStatus(`Disable failed: ${err.message || err}`, 'warn');
    return null;
  }
}

async function refreshOverlayPerfCapture() {
  try {
    pushOverlayPerfIfDue(true);
    const snapshot = await requestOverlayPerfSnapshot();
    setOverlayPerfStatus('Perf snapshot refreshed.', snapshot.enabled ? 'ok' : 'warn');
    return snapshot;
  } catch (err) {
    setOverlayPerfStatus(`Snapshot failed: ${err.message || err}`, 'warn');
    return null;
  }
}

async function copyOverlayPerfSnapshotJSON() {
  const snapshot = _lastOverlayPerfSnapshot || await refreshOverlayPerfCapture();
  if (!snapshot) return '';
  const json = JSON.stringify(snapshot, null, 2);
  try {
    await navigator.clipboard?.writeText(json);
  } catch {
    const area = document.createElement('textarea');
    area.value = json;
    area.style.position = 'fixed';
    area.style.left = '-9999px';
    document.body.appendChild(area);
    area.select();
    document.execCommand('copy');
    document.body.removeChild(area);
  }
  setOverlayPerfStatus('Copied perf snapshot JSON.', 'ok');
  return json;
}

function wireOverlayPerfControls() {
  const controls = [
    ['overlay-perf-enable', enableOverlayPerfCapture],
    ['overlay-perf-reset', resetOverlayPerfCapture],
    ['overlay-perf-snapshot', refreshOverlayPerfCapture],
    ['overlay-perf-copy', copyOverlayPerfSnapshotJSON],
    ['overlay-perf-disable', disableOverlayPerfCapture],
  ];
  for (const [id, handler] of controls) {
    const el = document.getElementById(id);
    if (!el || el.dataset.wired) continue;
    el.dataset.wired = '1';
    el.addEventListener('click', () => {
      handler();
    });
  }
}

async function refreshOverlayMomentum() {
  const root = document.getElementById('view-overlay');
  if (!root) return;
  if (Date.now() < _demoModeUntil) return;
  const previewPaused = isOverlayLabPreviewPaused();
  applyOverlayLabPreviewPaused(previewPaused);
  if (previewPaused) {
    overlayPerfCount('lab.previewPaused');
    pushOverlayPerfIfDue();
    return;
  }
  try {
    overlayPerfCount('fetch.momentum');
    const out = await fetch('/api/overlay/momentum').then(r => r.json());
    setOverlayPerfEnabled(Boolean(out.perfEnabled || overlayPerfFlagEnabled()));
    applyServerMomentumPrefs(out.prefs);
    _lastMomentumOutput = out;
    renderOverlayMomentum(out);
    pushOverlayPerfIfDue();
  } catch (err) {
    const debug = document.getElementById('momentum-debug');
    if (debug) debug.textContent = `Momentum Flow unavailable: ${err.message || err}`;
  }
}

function renderOverlayMomentum(out) {
  overlayPerfCount('renderOverlayMomentum');
  const replayAdjusted = withReplayDisplayState(out);
  const displayOut = replayAdjusted.skipHold
    ? replayAdjusted.output
    : withHeldDisplayState(replayAdjusted.output);
  _momentumFlowBar?.update(displayOut);
  _momentumControlWheel?.update(displayOut);
  updateMomentumVisualMode((_currentMomentumWidgetConfig?.visual || DEFAULT_MFB_CONFIG.visual));
  updateLocalTimeline(displayOut);
  _lastDisplayOutput = displayOut;

  setText('momentum-state', displayOut.state || 'NEUTRAL');
  setText('momentum-confidence', pct(out.confidence));
  setText('momentum-volatility', pct(out.volatility));
  setText('momentum-pulse', out.overlay?.pulse || out.overlaySignals?.pulse || 'None');
  setText('momentum-pulse-team', out.overlay?.pulseTeam || out.overlaySignals?.pulseTeam || 'None');
  setText('momentum-blue-control', num(out.blue?.control));
  setText('momentum-blue-pressure', num(out.blue?.pressure));
  setText('momentum-orange-control', num(out.orange?.control));
  setText('momentum-orange-pressure', num(out.orange?.pressure));
  setText('momentum-last-strong', strongEventSummary(out.debug?.lastStrongEvent));

  const recent = Array.isArray(out.debug?.lastEvents) ? out.debug.lastEvents.slice(-10) : [];
  const reasons = Array.isArray(out.debug?.reasons) ? out.debug.reasons : [];
  const weights = Array.isArray(out.debug?.weightsApplied) ? out.debug.weightsApplied : [];
  const eventCounts = countSummary(out.debug?.eventCounts);
  const sourceCounts = countSummary(out.debug?.sourceCounts);
  const debug = document.getElementById('momentum-debug');
  if (debug) {
    const events = recent.map(ev => `${new Date(ev.time).toLocaleTimeString()}  ${ev.type}  ${ev.team}  ${ev.playerName || ev.playerId || ''}  [${ev.sourceEvent || 'unknown'}]`);
    const applied = weights.map(weightSummary);
    debug.textContent = [
      'Signal counts:',
      eventCounts || 'No normalized signals yet.',
      '',
      'Source counts:',
      sourceCounts || 'No source events yet.',
      '',
      'Recent normalized events:',
      events.length ? events.join('\n') : 'None yet.',
      '',
      'Reasons:',
      reasons.length ? reasons.join('\n') : 'No active weighting reasons.',
      '',
      'Weights applied:',
      applied.length ? applied.join('\n') : 'No active weights.',
    ].join('\n');
  }
}

function updateLocalTimeline(out) {
  const now = Date.now();
  if (!_timelineStart) _timelineStart = now;
  const sample = {
    at: now,
    state: out.state || 'NEUTRAL',
    confidence: clamp01(Number(out.confidence || 0)),
    bluePercent: Number(out.overlay?.momentumBarBluePercent ?? out.overlaySignals?.momentumBarBluePercent ?? 50),
    orangePercent: Number(out.overlay?.momentumBarOrangePercent ?? out.overlaySignals?.momentumBarOrangePercent ?? 50),
  };
  const last = _timelineSamples[_timelineSamples.length - 1];
  if (!last || last.state !== sample.state || Math.abs(last.bluePercent - sample.bluePercent) >= 6 || now - last.at >= 3000) {
    _timelineSamples.push(sample);
    if (_timelineSamples.length > MAX_TIMELINE_SAMPLES) _timelineSamples.shift();
  }

  const ev = out.debug?.lastStrongEvent;
  if (ev && ev.time) {
    const key = `${ev.type}:${ev.team}:${ev.playerId || ev.playerName || ''}:${ev.time}`;
    if (key !== _lastTimelineEventKey) {
      _lastTimelineEventKey = key;
      _timelineEvents.push({
        at: now,
        type: ev.type || 'event',
        team: ev.team || '',
        player: ev.playerName || ev.playerId || 'Unknown',
      });
      if (_timelineEvents.length > MAX_TIMELINE_EVENTS) _timelineEvents.shift();
    }
  }

  renderMomentumTimeline();
}

function renderMomentumTimeline() {
  const bar = document.getElementById('momentum-track-bar');
  const markers = document.getElementById('momentum-markers');
  const feed = document.getElementById('momentum-event-feed');
  if (!bar || !markers || !feed) return;

  if (!_timelineSamples.length) {
    bar.innerHTML = '<div class="momentum-track-segment momentum-track-segment--neutral" style="width:100%"></div>';
    markers.innerHTML = '';
  } else {
    const total = Math.max(1, _timelineSamples[_timelineSamples.length - 1].at - _timelineSamples[0].at);
    bar.innerHTML = _timelineSamples.map((sample, index) => {
      const next = _timelineSamples[index + 1];
      const duration = Math.max(450, (next ? next.at : Date.now()) - sample.at);
      const width = Math.max(2, (duration / total) * 100);
      const state = displayState(sample.state);
      const opacity = 0.35 + sample.confidence * 0.65;
      return `<div class="momentum-track-segment momentum-track-segment--${state.kind}" title="${escapeHtml(state.label)}" style="width:${width.toFixed(2)}%; opacity:${opacity.toFixed(2)}"></div>`;
    }).join('');

    const span = Math.max(1, Date.now() - _timelineSamples[0].at);
    const visibleMarkers = visibleTimelineMarkers(_timelineEvents);
    markers.innerHTML = visibleMarkers.map(event => {
      const left = clampPercent(((event.at - _timelineSamples[0].at) / span) * 100);
      const label = event.label || displayEvent(event.type);
      const teamClass = event.team === 'orange' ? 'momentum-marker--orange' : event.team === 'blue' ? 'momentum-marker--blue' : '';
      const goalClass = event.type === 'goal' ? 'momentum-marker--goal' : '';
      const typeClass = `momentum-marker--${event.type || 'event'}`;
      const details = event.related?.length ? ` (${event.related.map(displayEvent).join(', ')})` : '';
      return `<div class="momentum-marker ${teamClass} ${goalClass} ${typeClass}" style="left:${left.toFixed(2)}%" title="${escapeHtml(label)}${details} - ${escapeHtml(event.player)}">${escapeHtml(event.shortLabel || shortEventLabel(event.type))}</div>`;
    }).join('');
  }

  const visibleFeedEvents = visibleTimelineMarkers(_timelineEvents);
  if (!visibleFeedEvents.length) {
    feed.innerHTML = '<div class="momentum-empty">Waiting for key goals, saves, assists, demos, and pressure shots...</div>';
  } else {
    feed.innerHTML = visibleFeedEvents.slice().reverse().map(event => {
      const time = new Date(event.at).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit', second: '2-digit' });
      const label = event.label || displayEvent(event.type);
      const team = displayTeam(event.team);
      return `<div class="momentum-event-row"><span>${escapeHtml(time)}</span><strong>${escapeHtml(label)}</strong><span>${escapeHtml(team)}${team ? ' - ' : ''}${escapeHtml(event.player)}</span></div>`;
    }).join('');
  }
}

function visibleTimelineMarkers(events) {
  if (!events.length) return [];
  const clusters = [];
  for (const event of events) {
    const previous = clusters[clusters.length - 1];
    if (previous && event.at - previous.endAt <= TIMELINE_MARKER_CLUSTER_MS && relatedTimelineEvents(previous.events, event)) {
      previous.events.push(event);
      previous.endAt = event.at;
      continue;
    }
    clusters.push({ events: [event], endAt: event.at });
  }
  return clusters
    .map(cluster => markerFromCluster(cluster.events))
    .filter(Boolean);
}

function relatedTimelineEvents(cluster, event) {
  if (!cluster.length) return false;
  if (event.type === 'goal' || event.type === 'save' || event.type === 'assist') return true;
  return cluster.some(existing => existing.team === event.team || markerPriority(existing.type) >= markerPriority('save'));
}

function markerFromCluster(cluster) {
  const sorted = cluster.slice().sort((a, b) => markerPriority(b.type) - markerPriority(a.type));
  const primary = sorted[0];
  if (!primary) return null;

  const related = uniqueEventTypes(cluster)
    .filter(type => type !== primary.type && markerPriority(type) >= markerPriority('demo'));
  if (primary.type === 'shot' && !cluster.some(event => event.type === 'demo' || event.type === 'save' || event.type === 'goal')) {
    const recentSameTeamShots = cluster.filter(event => event.type === 'shot' && event.team === primary.team);
    if (recentSameTeamShots.length < 2) return null;
  }

  const hasDemo = cluster.some(event => event.type === 'demo' && event.team === primary.team);
  const label = markerDisplayLabel(primary.type, hasDemo, related);
  const shortLabel = markerShortLabel(primary.type, hasDemo, related);
  return {
    ...primary,
    label,
    shortLabel,
    related,
  };
}

function markerPriority(type) {
  switch (type) {
    case 'goal': return 100;
    case 'save': return 80;
    case 'assist': return 70;
    case 'demo': return 55;
    case 'shot': return 45;
    default: return 0;
  }
}

function uniqueEventTypes(events) {
  const seen = new Set();
  const out = [];
  for (const event of events) {
    if (seen.has(event.type)) continue;
    seen.add(event.type);
    out.push(event.type);
  }
  return out;
}

function markerDisplayLabel(type, hasDemo, related) {
  if (type === 'goal' && related.includes('assist')) return 'Assisted Goal';
  if (type === 'goal') return 'Goal';
  if (type === 'save') return 'Save';
  if (type === 'assist') return 'Assist';
  if (type === 'demo' && related.includes('shot')) return 'Demo Pressure';
  if (type === 'demo') return 'Demo';
  if (type === 'shot' && hasDemo) return 'Demo + Shot';
  if (type === 'shot') return 'Pressure Shot';
  return 'Event';
}

function markerShortLabel(type, hasDemo, related) {
  if (type === 'goal' && related.includes('assist')) return 'G+A';
  if (type === 'goal') return 'G';
  if (type === 'save') return 'SV';
  if (type === 'assist') return 'A';
  if (type === 'demo' && related.includes('shot')) return 'DM+SH';
  if (type === 'demo') return 'DM';
  if (type === 'shot' && hasDemo) return 'DM+SH';
  if (type === 'shot') return 'SH';
  return 'E';
}

function resetLocalTimeline() {
  _timelineStart = 0;
  _timelineSamples = [];
  _timelineEvents = [];
  _lastTimelineEventKey = '';
  _heldDisplayState = null;
  _lastDisplayOutput = null;
  _replayFrozenOutput = null;
  _wasReplayActive = false;
  _postReplayNeutralStarted = 0;
  _postReplayNeutralUntil = 0;
  _postReplaySignalAfter = 0;
  _awaitingPostReplayLiveSignal = false;
  _lastGoalFallbackKey = '';
  _goalFallbackStarted = 0;
  _lastMomentumResetAt = 0;
}

function withReplayDisplayState(out = {}) {
  const now = Date.now();
  const resetAt = Number(out.display?.momentumResetAt || 0);
  if (resetAt && resetAt !== _lastMomentumResetAt) {
    _lastMomentumResetAt = resetAt;
    _replayFrozenOutput = null;
    _wasReplayActive = false;
    _postReplayNeutralStarted = 0;
    _postReplayNeutralUntil = 0;
    _postReplaySignalAfter = 0;
    _awaitingPostReplayLiveSignal = false;
    _goalFallbackStarted = 0;
    _heldDisplayState = null;
    _lastDisplayDominance = null;
    return { output: neutralizedDisplayOutput(out), skipHold: true };
  }

  const replayActive = Boolean(out.display?.replayActive);
  const replayFileMode = Boolean(out.display?.replayFileMode);

  if (replayActive && replayFileMode) {
    _wasReplayActive = false;
    _awaitingPostReplayLiveSignal = false;
    _postReplayNeutralStarted = 0;
    _postReplayNeutralUntil = 0;
    _postReplaySignalAfter = 0;
    _goalFallbackStarted = 0;
    _replayFrozenOutput = null;
    return { output: out, skipHold: false };
  }

  if (replayActive) {
    _wasReplayActive = true;
    _awaitingPostReplayLiveSignal = false;
    _postReplayNeutralStarted = 0;
    _postReplayNeutralUntil = 0;
    _postReplaySignalAfter = 0;
    _goalFallbackStarted = 0;
    if (!_replayFrozenOutput && _lastDisplayOutput && isHoldableDisplayState(_lastDisplayOutput.state)) {
      _replayFrozenOutput = cloneDisplayOutput(_lastDisplayOutput);
    }
    if (_replayFrozenOutput) {
      return { output: {
        ...out,
        ...cloneDisplayOutput(_replayFrozenOutput),
        debug: out.debug,
        display: out.display,
      }, skipHold: true };
    }
    return { output: out, skipHold: true };
  }

  if (_wasReplayActive) {
    _wasReplayActive = false;
    _replayFrozenOutput = null;
    _postReplayNeutralStarted = now;
    _postReplayNeutralUntil = now + POST_REPLAY_MAX_NEUTRAL_MS;
    _postReplaySignalAfter = Number(out.display?.replayChanged || now);
    _awaitingPostReplayLiveSignal = true;
    _heldDisplayState = null;
    _lastDisplayDominance = null;
  }

  const goalKey = goalFallbackKey(out);
  if (goalKey && goalKey !== _lastGoalFallbackKey) {
    _lastGoalFallbackKey = goalKey;
    _goalFallbackStarted = now;
  }

  if (!_awaitingPostReplayLiveSignal && _goalFallbackStarted && now - _goalFallbackStarted >= GOAL_FALLBACK_RESET_MS) {
    _goalFallbackStarted = 0;
    _replayFrozenOutput = null;
    _heldDisplayState = null;
    _lastDisplayDominance = null;
    _postReplayNeutralStarted = now;
    _postReplayNeutralUntil = now + POST_REPLAY_MAX_NEUTRAL_MS;
    _postReplaySignalAfter = now;
    _awaitingPostReplayLiveSignal = true;
  }

  if (_awaitingPostReplayLiveSignal) {
    const minNeutralSatisfied = now - _postReplayNeutralStarted >= POST_REPLAY_MIN_NEUTRAL_MS;
    if (minNeutralSatisfied && hasLiveSignalAfterReplay(out)) {
      _awaitingPostReplayLiveSignal = false;
      _postReplayNeutralStarted = 0;
      _postReplayNeutralUntil = 0;
      _postReplaySignalAfter = 0;
    } else if (now <= _postReplayNeutralUntil) {
      return { output: neutralizedDisplayOutput(out), skipHold: true };
    } else {
      _awaitingPostReplayLiveSignal = false;
      _postReplayNeutralStarted = 0;
      _postReplayNeutralUntil = 0;
      _postReplaySignalAfter = 0;
    }
  }

  return { output: out, skipHold: false };
}

function cloneDisplayOutput(out) {
  return {
    ...out,
    blue: { ...(out.blue || {}) },
    orange: { ...(out.orange || {}) },
    overlay: { ...(out.overlay || out.overlaySignals || {}) },
  };
}

function neutralizedDisplayOutput(out = {}) {
  return {
    ...out,
    state: 'NEUTRAL',
    confidence: 0,
    volatility: 0,
    overlay: {
      ...(out.overlay || {}),
      momentumBarBluePercent: 50,
      momentumBarOrangePercent: 50,
      pulse: '',
      pulseTeam: '',
    },
    overlaySignals: {
      ...(out.overlaySignals || {}),
      momentumBarBluePercent: 50,
      momentumBarOrangePercent: 50,
      pulse: '',
      pulseTeam: '',
    },
  };
}

function hasLiveSignalAfterReplay(out = {}) {
  const changed = Number(_postReplaySignalAfter || out.display?.replayChanged || 0);
  const recent = Array.isArray(out.debug?.lastEvents) ? out.debug.lastEvents : [];
  return recent.some(ev => {
    const t = Number(ev.time || 0);
    return t >= changed && ev.type === 'ball_hit';
  });
}

function goalFallbackKey(out = {}) {
  const ev = out.debug?.lastStrongEvent;
  if (ev?.type === 'goal' && ev.time) {
    return `goal:${ev.team || ''}:${ev.playerId || ev.playerName || ''}:${ev.time}`;
  }
  const signals = out.overlaySignals || out.overlay || {};
  if (signals.pulse === 'GOAL_BURST') {
    return `goal:pulse:${signals.pulseTeam || ''}:${out.display?.replayChanged || ''}:${Math.floor(Date.now() / GOAL_FALLBACK_RESET_MS)}`;
  }
  return '';
}

function withHeldDisplayState(out = {}) {
  const now = Date.now();
  const state = deriveDisplayState(out, now);
  const confidence = clamp01(Number(out.confidence || 0));
  const holdable = isHoldableDisplayState(state);
  const displayOut = { ...out, state };

  if (holdable) {
    const holdMs = DISPLAY_STATE_MIN_HOLD_MS + Math.round(confidence * (DISPLAY_STATE_MAX_HOLD_MS - DISPLAY_STATE_MIN_HOLD_MS));
    _heldDisplayState = {
      state,
      until: now + holdMs,
      confidence,
    };
    return displayOut;
  }

  if (state === 'NEUTRAL' && _heldDisplayState && now <= _heldDisplayState.until && _heldDisplayState.confidence >= 0.25) {
    return { ...displayOut, state: _heldDisplayState.state };
  }

  if (state === 'NEUTRAL' || state === 'VOLATILE') {
    _heldDisplayState = null;
  }
  return displayOut;
}

function deriveDisplayState(out = {}, now = Date.now()) {
  const signals = out.overlaySignals || out.overlay || {};
  let blue = Number(signals.momentumBarBluePercent);
  let orange = Number(signals.momentumBarOrangePercent);
  if (!Number.isFinite(blue) || !Number.isFinite(orange)) {
    const adapted = adaptMomentumFlowOutput(out);
    blue = adapted.bluePercent;
    orange = adapted.orangePercent;
  }
  blue = clampPercent(blue);
  orange = clampPercent(orange);

  const spread = Math.abs(blue - orange);
  const confidence = clamp01(Number(out.confidence || 0));
  const dominant = blue >= orange ? 'blue' : 'orange';
  const dominantPercent = Math.max(blue, orange);
  const backendState = out.state || 'NEUTRAL';
  const recentFlip = _lastDisplayDominance
    && _lastDisplayDominance.team !== dominant
    && now - _lastDisplayDominance.at <= DISPLAY_CONTESTED_FLIP_MS;

  if (backendState === 'VOLATILE' || spread <= DISPLAY_CONTESTED_SPREAD_PERCENT || (recentFlip && dominantPercent < DISPLAY_PRESSURE_PERCENT)) {
    _lastDisplayDominance = null;
    return 'VOLATILE';
  }

  if (dominantPercent >= DISPLAY_CONTROL_PERCENT) {
    _lastDisplayDominance = { team: dominant, at: now };
  }

  if (dominantPercent >= DISPLAY_PRESSURE_PERCENT && confidence >= 0.35) {
    return dominant === 'blue' ? 'BLUE_PRESSURE' : 'ORANGE_PRESSURE';
  }

  if (dominantPercent >= DISPLAY_CONTROL_PERCENT && confidence >= 0.22) {
    return dominant === 'blue' ? 'BLUE_CONTROL' : 'ORANGE_CONTROL';
  }

  return 'NEUTRAL';
}

function isHoldableDisplayState(state) {
  return state === 'BLUE_CONTROL'
    || state === 'ORANGE_CONTROL'
    || state === 'BLUE_PRESSURE'
    || state === 'ORANGE_PRESSURE';
}

function adaptMomentumFlowOutput(out = {}) {
  const signals = out.overlaySignals || out.overlay || {};
  let bluePercent = Number(signals.momentumBarBluePercent);
  let orangePercent = Number(signals.momentumBarOrangePercent);
  if (!Number.isFinite(bluePercent) || !Number.isFinite(orangePercent)) {
    const blueShare = Number(out.blue?.pressureShare);
    const orangeShare = Number(out.orange?.pressureShare);
    if (Number.isFinite(blueShare) && Number.isFinite(orangeShare) && blueShare + orangeShare > 0.001) {
      bluePercent = blueShare * 100;
      orangePercent = orangeShare * 100;
    } else {
      bluePercent = 50;
      orangePercent = 50;
    }
  }
  bluePercent = clampPercent(bluePercent);
  orangePercent = clampPercent(orangePercent);
  const total = bluePercent + orangePercent;
  if (total > 0.001) {
    bluePercent = (bluePercent / total) * 100;
    orangePercent = 100 - bluePercent;
  } else {
    bluePercent = 50;
    orangePercent = 50;
  }

  return {
    state: out.state || 'NEUTRAL',
    confidence: clamp01(Number(out.confidence || 0)),
    volatility: clamp01(Number(out.volatility || 0)),
    bluePercent,
    orangePercent,
    pulse: signals.pulse || '',
    pulseTeam: signals.pulseTeam || '',
    pulseKey: pulseIdentity(out, signals),
  };
}

function adaptMomentumControlWheelSignal(out = {}) {
  if (out && (out.bluePercent !== undefined || out.blueControlShare !== undefined)) {
    return normalizeMomentumControlWheelSignal(out);
  }

  const adapted = adaptMomentumFlowOutput(out);
  const signals = out.overlaySignals || out.overlay || {};
  let bluePercent = Number(signals.momentumBarBluePercent);
  let orangePercent = Number(signals.momentumBarOrangePercent);

  if (!Number.isFinite(bluePercent) || !Number.isFinite(orangePercent)) {
    const blueControl = Number(out.blue?.controlShare);
    const orangeControl = Number(out.orange?.controlShare);
    if (Number.isFinite(blueControl) && Number.isFinite(orangeControl) && blueControl + orangeControl > 0.001) {
      bluePercent = blueControl * 100;
      orangePercent = orangeControl * 100;
    } else {
      bluePercent = adapted.bluePercent;
      orangePercent = adapted.orangePercent;
    }
  }

  return normalizeMomentumControlWheelSignal({
    time: formatWheelClock(out),
    bluePercent,
    orangePercent,
    state: wheelStateFromEngine(adapted.state, bluePercent, orangePercent, adapted.confidence),
    confidence: confidenceValueLabel(adapted.confidence),
    volatility: adapted.volatility,
    bluePressure: Number(out.blue?.pressureShare ?? bluePercent / 100),
    orangePressure: Number(out.orange?.pressureShare ?? orangePercent / 100),
    ...momentumWheelRecentEvent(out),
    showOOFBadge: true,
    reducedMotion: false,
    performanceMode: false,
  });
}

function normalizeMomentumControlWheelSignal(signal = {}) {
  const values = normalizeMomentumWheelDisplayValues(
    signal.bluePercent ?? signal.blueControlShare,
    signal.orangePercent ?? signal.orangeControlShare
  );
  const confidence = normalizeWheelConfidence(signal.confidence);
  const state = normalizeWheelState(signal.state, values.bluePercent, values.orangePercent, confidence);
  return {
    ...signal,
    time: signal.time || signal.matchClock || '--:--',
    bluePercent: values.bluePercent,
    orangePercent: values.orangePercent,
    blueControlShare: values.bluePercent / 100,
    orangeControlShare: values.orangePercent / 100,
    state,
    confidence,
    volatility: momentumWheelClamp(Number(signal.volatility ?? 0), 0, 1),
    bluePressure: momentumWheelClamp(Number(signal.bluePressure ?? values.bluePercent / 100), 0, 1),
    orangePressure: momentumWheelClamp(Number(signal.orangePressure ?? values.orangePercent / 100), 0, 1),
    recentEventEnergy: momentumWheelClamp(Number(signal.recentEventEnergy ?? 0), 0, 1),
    recentEventTeam: normalizeWheelTeam(signal.recentEventTeam),
    recentEventType: normalizeWheelEventType(signal.recentEventType),
    showOOFBadge: signal.showOOFBadge !== false,
    reducedMotion: Boolean(signal.reducedMotion),
    performanceMode: Boolean(signal.performanceMode),
  };
}

function normalizeMomentumWheelDisplayValues(blueValue, orangeValue) {
  let bluePercent = Number(blueValue);
  let orangePercent = Number(orangeValue);

  if (Number.isFinite(bluePercent) && Math.abs(bluePercent) <= 1 && (!Number.isFinite(orangePercent) || Math.abs(orangePercent) <= 1)) {
    bluePercent *= 100;
    if (Number.isFinite(orangePercent)) orangePercent *= 100;
  }
  if (!Number.isFinite(bluePercent) && Number.isFinite(orangePercent)) {
    bluePercent = 100 - orangePercent;
  }
  if (Number.isFinite(bluePercent) && !Number.isFinite(orangePercent)) {
    orangePercent = 100 - bluePercent;
  }
  if (!Number.isFinite(bluePercent) || !Number.isFinite(orangePercent)) {
    bluePercent = 50;
    orangePercent = 50;
  }

  bluePercent = momentumWheelClamp(bluePercent, 0, 100);
  orangePercent = momentumWheelClamp(orangePercent, 0, 100);
  const total = bluePercent + orangePercent;
  if (total <= 0.001) {
    return { bluePercent: 50, orangePercent: 50 };
  }
  if (Math.abs(total - 100) > 0.5) {
    bluePercent = (bluePercent / total) * 100;
    orangePercent = 100 - bluePercent;
  }
  return {
    bluePercent: momentumWheelClamp(bluePercent, 0, 100),
    orangePercent: momentumWheelClamp(orangePercent, 0, 100),
  };
}

function momentumWheelClamp(value, min, max) {
  const n = Number(value);
  if (!Number.isFinite(n)) return min;
  return Math.min(max, Math.max(min, n));
}

function polar(cx, cy, radius, wheelAngleDegrees) {
  const svgAngle = (Number(wheelAngleDegrees) - 90) * Math.PI / 180;
  return {
    x: cx + radius * Math.cos(svgAngle),
    y: cy + radius * Math.sin(svgAngle),
  };
}

function angularDistanceClockwise(fromAngle, toAngle) {
  return (((Number(toAngle) - Number(fromAngle)) % 360) + 360) % 360;
}

function circularDistance(a, b) {
  const diff = Math.abs((((Number(a) - Number(b)) % 360) + 360) % 360);
  return Math.min(diff, 360 - diff);
}

function calculateSeamAngle(bluePercent, orangePercent) {
  const values = normalizeMomentumWheelDisplayValues(bluePercent, orangePercent);
  return (180 + values.bluePercent * 3.6) % 360;
}

function calculateSegmentOwnership(segmentAngle, bluePercent, orangePercent) {
  const values = normalizeMomentumWheelDisplayValues(bluePercent, orangePercent);
  if (values.bluePercent >= 99.999) return 'blue';
  if (values.orangePercent >= 99.999) return 'orange';
  const angle = (((Number(segmentAngle) % 360) + 360) % 360);
  const blueSpanDegrees = values.bluePercent * 3.6;
  const distanceFromOrigin = angularDistanceClockwise(180, angle);
  return distanceFromOrigin < blueSpanDegrees ? 'blue' : 'orange';
}

function momentumControlWheelSeamAngle(blueControlShare) {
  const bluePercent = Number(blueControlShare) * 100;
  return calculateSeamAngle(bluePercent, 100 - bluePercent);
}

function momentumControlWheelSegmentOwnership(theta, signalOrBlueShare = 0.5) {
  const signal = typeof signalOrBlueShare === 'number'
    ? normalizeMomentumControlWheelSignal({ blueControlShare: signalOrBlueShare })
    : normalizeMomentumControlWheelSignal(signalOrBlueShare);
  return calculateSegmentOwnership(theta, signal.bluePercent, signal.orangePercent);
}

function angleDistance(a, b) {
  return circularDistance(a, b);
}

function svgEl(tag, attrs = {}, children = []) {
  const el = document.createElementNS('http://www.w3.org/2000/svg', tag);
  setAttrs(el, attrs);
  for (const child of children) {
    if (typeof child === 'string') {
      el.appendChild(document.createTextNode(child));
    } else if (child) {
      el.appendChild(child);
    }
  }
  return el;
}

function setAttrs(el, attrs = {}) {
  if (!el) return;
  for (const [key, value] of Object.entries(attrs)) {
    if (value === undefined || value === null) continue;
    const attrName = key === 'className' ? 'class' : key;
    if (overlayPerfEnabled() && el.getAttribute(attrName) !== String(value)) {
      overlayPerfCount('dom.attrMutation');
    }
    if (key === 'className') {
      el.setAttribute('class', String(value));
    } else {
      el.setAttribute(key, String(value));
    }
  }
}

function removeSvgAttr(el, attrName) {
  if (!el) return;
  if (typeof el.removeAttribute === 'function') {
    el.removeAttribute(attrName);
  } else if (el.attrs) {
    delete el.attrs[attrName];
  }
}

function linearGradient(id, stops) {
  const gradient = svgEl('linearGradient', { id, x1: '0%', y1: '0%', x2: '0%', y2: '100%' });
  for (const [offset, color] of stops) {
    gradient.appendChild(svgEl('stop', { offset, 'stop-color': color }));
  }
  return gradient;
}

function radialGradient(id, stops) {
  const gradient = svgEl('radialGradient', { id, cx: '50%', cy: '50%', r: '50%' });
  for (const [offset, color] of stops) {
    gradient.appendChild(svgEl('stop', { offset, 'stop-color': color }));
  }
  return gradient;
}

function segmentAttrs(angle) {
  return {
    x: '505',
    y: '108',
    width: '14',
    height: '108',
    rx: '999',
    ry: '999',
    transform: `rotate(${n(angle)} 512 512)`,
  };
}

function radialLine(innerRadius, outerRadius, angle, color, opacity, strokeWidth = 2) {
  const inner = polar(512, 512, innerRadius, angle);
  const outer = polar(512, 512, outerRadius, angle);
  return svgEl('line', {
    x1: n(inner.x),
    y1: n(inner.y),
    x2: n(outer.x),
    y2: n(outer.y),
    stroke: color,
    'stroke-width': n(strokeWidth),
    'stroke-linecap': 'round',
    opacity: String(opacity),
  });
}

function appendSparkFixtures(group, fixtures, tone) {
  fixtures.forEach((fixture, index) => {
    const el = createSparkParticle(fixture, tone, index);
    group.appendChild(el);
  });
}

function createSparkParticle(fixture, tone, index) {
  const localAngle = Number.isFinite(Number(fixture.localAngle)) ? Number(fixture.localAngle) : 0;
  const point = Number.isFinite(Number(fixture.localAngle))
    ? polar(512, 512, Number(fixture.r || 430), localAngle)
    : { x: 512 + Number(fixture.x || 0), y: 512 - Number(fixture.r || 430) };
  const x = point.x;
  const y = point.y;
  const color = sparkColor(tone);
  const zone = fixture.zone || 'seam';
  const roles = Array.isArray(fixture.roles) ? fixture.roles : [];
  const roleClasses = roles.map((role) => `mcw-spark-role-${role}`).join(' ');
  const cls = `mcw-spark mcw-spark--${fixture.type || 'circle'} mcw-spark--${tone} mcw-spark-zone--${zone} ${roleClasses}`.trim();
  let el;
  switch (fixture.type) {
    case 'line': {
      const half = Number(fixture.length || 18) / 2;
      el = svgEl('line', {
        class: cls,
        x1: n(x - half),
        y1: n(y),
        x2: n(x + half),
        y2: n(y),
        stroke: color,
        'stroke-width': n(fixture.width || 3),
        'stroke-linecap': 'round',
        opacity: n(fixture.opacity || 0.72),
        transform: `rotate(${n(fixture.angle || 0)} ${n(x)} ${n(y)})`,
      });
      break;
    }
    case 'diamond': {
      const s = Number(fixture.size || 6);
      el = svgEl('path', {
        class: cls,
        d: `M ${n(x)} ${n(y - s)} L ${n(x + s)} ${n(y)} L ${n(x)} ${n(y + s)} L ${n(x - s)} ${n(y)} Z`,
        fill: color,
        opacity: n(fixture.opacity || 0.68),
        transform: `rotate(${n(fixture.angle || 0)} ${n(x)} ${n(y)})`,
      });
      break;
    }
    case 'jagged': {
      const width = Number(fixture.width || 3);
      el = svgEl('path', {
        class: cls,
        d: `M ${n(x - 14)} ${n(y + 6)} L ${n(x - 4)} ${n(y - 6)} L ${n(x + 4)} ${n(y + 4)} L ${n(x + 16)} ${n(y - 8)}`,
        fill: 'none',
        stroke: color,
        'stroke-width': n(width),
        'stroke-linecap': 'round',
        'stroke-linejoin': 'round',
        opacity: n(fixture.opacity || 0.78),
        transform: `rotate(${n(fixture.angle || 0)} ${n(x)} ${n(y)})`,
      });
      break;
    }
    default:
      el = svgEl('circle', {
        class: cls,
        cx: n(x),
        cy: n(y),
        r: n(fixture.size || 3),
        fill: color,
        opacity: n(fixture.opacity || 0.65),
      });
      break;
  }
  const travel = Math.abs(Number(fixture.travel || 36));
  const drift = Number(fixture.drift || 0);
  const rad = (localAngle * Math.PI) / 180;
  const dx = Math.sin(rad) * travel + Math.cos(rad) * drift;
  const dy = -Math.cos(rad) * travel + Math.sin(rad) * drift;
  el.style.setProperty('--mcw-spark-dx', `${n(dx)}px`);
  el.style.setProperty('--mcw-spark-dy', `${n(dy)}px`);
  el.style.setProperty('--mcw-spark-delay', `${Math.max(0, Number(fixture.delay || 0))}ms`);
  el.style.setProperty('--mcw-spark-duration', `${Math.max(350, Number(fixture.duration || 850))}ms`);
  el.style.setProperty('--mcw-spark-index', String(index));
  return el;
}

function sparkColor(tone) {
  switch (tone) {
    case 'blue': return '#38B2F6';
    case 'orange': return '#F97316';
    case 'purple': return '#F0B8FF';
    case 'white': return '#F5F7FB';
    default: return '#F5F7FB';
  }
}

function n(value) {
  return Number(value).toFixed(2).replace(/\.00$/, '');
}

function momentumWheelStateConfig(signal) {
  const state = signal.state || 'neutral';
  const confidence = signal.confidence || 'low';
  const recentEnergy = momentumWheelClamp(signal.recentEventEnergy, 0, 1);
  const base = {
    stateText: wheelDisplayLabel(state),
    blueWash: String(0.035 + recentEnergy * 0.065),
    orangeWash: String(0.035 + recentEnergy * 0.065),
    purpleWash: '0',
    centerOpacity: String(0.08 + recentEnergy * 0.06),
    auraOpacity: confidence === 'max' ? '0.5' : '0.28',
    blueAuraBase: String(0.05 + recentEnergy * 0.08),
    blueAuraPeak: String(0.07 + recentEnergy * 0.12),
    orangeAuraBase: String(0.05 + recentEnergy * 0.08),
    orangeAuraPeak: String(0.07 + recentEnergy * 0.12),
    contestAuraBase: String(0.04 + recentEnergy * 0.08 + signal.volatility * 0.08),
    contestAuraPeak: String(0.06 + recentEnergy * 0.12 + signal.volatility * 0.12),
    auraPulseMs: 2200,
    contestOpacity: String(0.12 + recentEnergy * 0.18 + signal.volatility * 0.18),
  };
  switch (state) {
    case 'blue-pressure':
      return { ...base, blueWash: '0.16', orangeWash: '0.055', centerOpacity: '0.16', blueAuraBase: '0.18', blueAuraPeak: '0.30', orangeAuraBase: '0.045', orangeAuraPeak: '0.06', contestAuraBase: '0.12', contestAuraPeak: '0.20', auraPulseMs: 1750, contestOpacity: '0.38' };
    case 'orange-pressure':
      return { ...base, blueWash: '0.055', orangeWash: '0.16', centerOpacity: '0.16', blueAuraBase: '0.045', blueAuraPeak: '0.06', orangeAuraBase: '0.18', orangeAuraPeak: '0.30', contestAuraBase: '0.12', contestAuraPeak: '0.20', auraPulseMs: 1750, contestOpacity: '0.38' };
    case 'blue-control':
      return { ...base, blueWash: '0.23', orangeWash: '0.045', centerOpacity: '0.23', blueAuraBase: '0.26', blueAuraPeak: '0.39', orangeAuraBase: '0.04', orangeAuraPeak: '0.055', contestAuraBase: '0.10', contestAuraPeak: '0.16', auraPulseMs: 2250, contestOpacity: '0.48' };
    case 'orange-control':
      return { ...base, blueWash: '0.045', orangeWash: '0.23', centerOpacity: '0.23', blueAuraBase: '0.04', blueAuraPeak: '0.055', orangeAuraBase: '0.26', orangeAuraPeak: '0.39', contestAuraBase: '0.10', contestAuraPeak: '0.16', auraPulseMs: 2250, contestOpacity: '0.48' };
    case 'volatile':
      return { ...base, blueWash: '0.12', orangeWash: '0.12', purpleWash: '0.22', centerOpacity: '0.18', blueAuraBase: '0.09', blueAuraPeak: '0.13', orangeAuraBase: '0.09', orangeAuraPeak: '0.13', contestAuraBase: '0.36', contestAuraPeak: '0.72', auraPulseMs: 900, contestOpacity: '0.86' };
    case 'dominant-blue':
      return { ...base, blueWash: '0.40', orangeWash: '0.02', centerOpacity: '0.40', auraOpacity: '0.50', blueAuraBase: '0.34', blueAuraPeak: '0.58', orangeAuraBase: '0.025', orangeAuraPeak: '0.04', contestAuraBase: '0.06', contestAuraPeak: '0.10', auraPulseMs: 2850, contestOpacity: '0.24', stateText: 'BLUE CONTROL' };
    case 'dominant-orange':
      return { ...base, blueWash: '0.02', orangeWash: '0.40', centerOpacity: '0.40', auraOpacity: '0.50', blueAuraBase: '0.025', blueAuraPeak: '0.04', orangeAuraBase: '0.34', orangeAuraPeak: '0.58', contestAuraBase: '0.06', contestAuraPeak: '0.10', auraPulseMs: 2850, contestOpacity: '0.24', stateText: 'ORANGE CONTROL' };
    default:
      return base;
  }
}

function applyMomentumWheelDisplayResponse(signal = {}, responseConfig = {}) {
  const response = sanitizeWheelResponseConfig(responseConfig);
  const pressureScale = 0.7 + response.pressureSensitivity * 0.3;
  const volatilityScale = 0.55 + response.volatilitySensitivity * 0.45;
  const eventScale = 0.35 + response.eventReactiveness * 0.65;
  const rawRecentEnergy = momentumWheelClamp(Number(signal.recentEventEnergy || 0) * eventScale, 0, 1);
  const recentExponent = response.timing.decaySpeed / Math.max(0.25, response.timing.holdDuration);
  return {
    ...signal,
    bluePressure: momentumWheelClamp(Number(signal.bluePressure || 0) * pressureScale, 0, 1),
    orangePressure: momentumWheelClamp(Number(signal.orangePressure || 0) * pressureScale, 0, 1),
    volatility: momentumWheelClamp(Number(signal.volatility || 0) * volatilityScale, 0, 1),
    recentEventEnergy: Math.pow(rawRecentEnergy, recentExponent),
  };
}

function momentumWheelRecentEvent(out = {}, now = Date.now()) {
  const signals = out.overlaySignals || out.overlay || {};
  const pulse = normalizeWheelEventType(pulseEventType(signals.pulse));
  const pulseTeam = normalizeWheelTeam(signals.pulseTeam);
  const ev = out.debug?.lastStrongEvent;
  const eventType = normalizeWheelEventType(ev?.type || pulse);
  const eventTeam = normalizeWheelTeam(pulseTeam || ev?.team);

  if (!eventType) {
    return {
      recentEventEnergy: pulse ? 0.82 : 0,
      recentEventTeam: eventTeam,
      recentEventType: pulse,
    };
  }

  const eventTime = Number(ev?.time || 0);
  if (!Number.isFinite(eventTime) || eventTime <= 0) {
    return {
      recentEventEnergy: pulse ? 0.82 : 0,
      recentEventTeam: eventTeam,
      recentEventType: eventType,
    };
  }

  const age = Math.max(0, Number(now) - eventTime);
  const totalMs = eventType === 'goal' ? WHEEL_GOAL_GLOW_DECAY_MS : WHEEL_EVENT_GLOW_DECAY_MS;
  if (age >= totalMs) {
    return {
      recentEventEnergy: 0,
      recentEventTeam: eventTeam,
      recentEventType: eventType,
    };
  }

  const decayStart = Math.min(WHEEL_EVENT_GLOW_FULL_MS, totalMs * 0.35);
  const raw = age <= decayStart ? 1 : 1 - ((age - decayStart) / Math.max(1, totalMs - decayStart));
  return {
    recentEventEnergy: Math.pow(momentumWheelClamp(raw, 0, 1), 1.35),
    recentEventTeam: eventTeam,
    recentEventType: eventType,
  };
}

function pulseEventType(pulse) {
  switch (String(pulse || '').toUpperCase()) {
    case 'SHOT': return 'shot';
    case 'SAVE_FORCED': return 'save';
    case 'DEMO_PRESSURE': return 'demo';
    case 'GOAL_BURST': return 'goal';
    default: return '';
  }
}

function normalizeWheelEventType(type) {
  const raw = String(type || '').trim().toLowerCase();
  return raw === 'shot' || raw === 'save' || raw === 'demo' || raw === 'goal' || raw === 'assist' ? raw : '';
}

function normalizeWheelTeam(team) {
  const raw = String(team || '').trim().toLowerCase();
  return raw === 'blue' || raw === 'orange' ? raw : '';
}

function wheelStateFromEngine(state, bluePercent, orangePercent, confidence) {
  const normalizedState = normalizeWheelState(state, bluePercent, orangePercent, normalizeWheelConfidence(confidence));
  const values = normalizeMomentumWheelDisplayValues(bluePercent, orangePercent);
  const dominant = values.bluePercent >= values.orangePercent ? 'blue' : 'orange';
  const dominantPercent = Math.max(values.bluePercent, values.orangePercent);
  const confidenceLevel = normalizeWheelConfidence(confidence);
  if ((normalizedState === 'blue-pressure' || normalizedState === 'blue-control') && dominant === 'blue' && dominantPercent >= 84 && confidenceLevel === 'max') {
    return 'dominant-blue';
  }
  if ((normalizedState === 'orange-pressure' || normalizedState === 'orange-control') && dominant === 'orange' && dominantPercent >= 84 && confidenceLevel === 'max') {
    return 'dominant-orange';
  }
  return normalizedState;
}

function normalizeWheelState(state, bluePercent, orangePercent, confidence) {
  const raw = String(state || '').trim().toLowerCase().replace(/_/g, '-').replace(/\s+/g, '-');
  const allowed = new Set([
    'neutral',
    'blue-pressure',
    'orange-pressure',
    'blue-control',
    'orange-control',
    'volatile',
    'dominant-blue',
    'dominant-orange',
  ]);
  if (allowed.has(raw)) return raw;
  const values = normalizeMomentumWheelDisplayValues(bluePercent, orangePercent);
  const spread = Math.abs(values.bluePercent - values.orangePercent);
  const dominant = values.bluePercent >= values.orangePercent ? 'blue' : 'orange';
  const confidenceRank = ['low', 'medium', 'high', 'max'].indexOf(confidence);
  if (spread <= 8) return 'neutral';
  if (Math.max(values.bluePercent, values.orangePercent) >= 88 && confidenceRank >= 2) {
    return dominant === 'blue' ? 'dominant-blue' : 'dominant-orange';
  }
  if (Math.max(values.bluePercent, values.orangePercent) >= 66 && confidenceRank >= 1) {
    return dominant === 'blue' ? 'blue-pressure' : 'orange-pressure';
  }
  return dominant === 'blue' ? 'blue-control' : 'orange-control';
}

function normalizeWheelConfidence(confidence) {
  if (typeof confidence === 'string') {
    const raw = confidence.trim().toLowerCase();
    if (raw === 'low' || raw === 'medium' || raw === 'high' || raw === 'max') return raw;
  }
  return confidenceValueLabel(confidence);
}

function confidenceValueLabel(confidence) {
  const c = clamp01(Number(confidence || 0));
  if (c >= 0.82) return 'max';
  if (c >= 0.62) return 'high';
  if (c >= 0.36) return 'medium';
  return 'low';
}

function sanitizeWheelConfig(input = {}) {
  const defaults = DEFAULT_MOMENTUM_WHEEL_CONFIG;
  const colorOverrides = sanitizeColorOverrides(input.colorOverrides || defaults.colorOverrides);
  const reducedMotion = Boolean(input.reducedMotion);
  const performanceMode = Boolean(input.performanceMode);
  const wheelVisual = sanitizeWheelVisualConfig(
    input.momentumControlWheel?.visual || input.wheelVisual || defaults.momentumControlWheel.visual,
    { reducedMotion, performanceMode },
  );
  const wheelResponse = sanitizeWheelResponseConfig(
    input.momentumControlWheel?.response || input.wheelResponse || defaults.momentumControlWheel.response,
    { reducedMotion, performanceMode },
  );
  return {
    enabled: input.enabled !== false,
    visual: input.visual === 'wheel' ? 'wheel' : input.visual === 'bar' ? 'bar' : 'bar',
    variant: sanitizeVariant(input.variant),
    scale: clampNumber(input.scale, 0.5, 1.75, defaults.scale),
    opacity: clampNumber(input.opacity, 0.2, 1, defaults.opacity),
    position: sanitizePosition(input.position),
    showOOFBadge: input.showOOFBadge !== false,
    showConfidence: input.showConfidence !== false,
    showStateLabel: input.showStateLabel !== false && input.showLabels !== false,
    showTelemetryWaveform: input.showTelemetryWaveform !== false,
    reducedMotion,
    performanceMode,
    glowIntensity: reducedMotion ? 0 : clampNumber(input.glowIntensity, 0, 1, defaults.glowIntensity),
    segmentBrightness: clampNumber(input.segmentBrightness, 0, 1, defaults.segmentBrightness),
    inactiveSegmentVisibility: clampNumber(input.inactiveSegmentVisibility, 0, 1, defaults.inactiveSegmentVisibility),
    seamIntensity: clampNumber(input.seamIntensity, 0, 1, defaults.seamIntensity),
    staticAuraIntensity: clampNumber(input.staticAuraIntensity, 0, 1, defaults.staticAuraIntensity),
    volatileEffects: reducedMotion || performanceMode ? 0 : clampNumber(input.volatileEffects, 0, 1, defaults.volatileEffects),
    dominantPulse: reducedMotion ? 0 : clampNumber(input.dominantPulse, 0, 1, defaults.dominantPulse),
    timerScale: clampNumber(input.timerScale, 0.82, 1.22, defaults.timerScale),
    labelScale: clampNumber(input.labelScale, 0.82, 1.18, defaults.labelScale),
    forceHighContrastText: input.forceHighContrastText !== false,
    theme: sanitizeTheme(input.theme),
    colorOverrides,
    debug: sanitizeDebugConfig(input.debug || input.debugWheel),
    momentumControlWheel: {
      visual: wheelVisual,
      response: wheelResponse,
    },
  };
}

function sanitizeWheelResponseConfig(input = {}, options = {}) {
  const source = input && typeof input === 'object' ? input : {};
  const defaults = DEFAULT_WHEEL_RESPONSE_CONFIG;
  const sourceTiming = source.timing && typeof source.timing === 'object' ? source.timing : {};
  const reducedMotion = Boolean(options.reducedMotion);
  const performanceMode = Boolean(options.performanceMode);
  const response = {
    pressureSensitivity: clampNumber(source.pressureSensitivity, 0, 2, defaults.pressureSensitivity),
    volatilitySensitivity: clampNumber(source.volatilitySensitivity, 0, 2, defaults.volatilitySensitivity),
    confidenceInfluence: clampNumber(source.confidenceInfluence, 0, 2, defaults.confidenceInfluence),
    smoothing: clampNumber(source.smoothing, 0, 1, defaults.smoothing),
    transitionSharpness: clampNumber(source.transitionSharpness, 0, 1, defaults.transitionSharpness),
    eventReactiveness: clampNumber(source.eventReactiveness, 0, 2, defaults.eventReactiveness),
    timing: {
      reactionSpeed: clampNumber(sourceTiming.reactionSpeed, 0.5, 2, defaults.timing.reactionSpeed),
      holdDuration: clampNumber(sourceTiming.holdDuration, 0.25, 3, defaults.timing.holdDuration),
      decaySpeed: clampNumber(sourceTiming.decaySpeed, 0.5, 2, defaults.timing.decaySpeed),
      pulseDuration: clampNumber(sourceTiming.pulseDuration, 0.25, 3, defaults.timing.pulseDuration),
      afterglowDuration: clampNumber(sourceTiming.afterglowDuration, 0, 5, defaults.timing.afterglowDuration),
      eventBurstDuration: clampNumber(sourceTiming.eventBurstDuration, 0.25, 3, defaults.timing.eventBurstDuration),
    },
  };
  if (reducedMotion) {
    response.volatilitySensitivity = Math.min(response.volatilitySensitivity, 0.35);
    response.eventReactiveness = 0;
    response.transitionSharpness = Math.min(response.transitionSharpness, 0.45);
    response.timing.pulseDuration = 0.25;
    response.timing.afterglowDuration = 0;
    response.timing.eventBurstDuration = 0.25;
  }
  if (performanceMode) {
    response.volatilitySensitivity = Math.min(response.volatilitySensitivity, 0.45);
    response.eventReactiveness = 0;
    response.transitionSharpness = Math.min(response.transitionSharpness, 0.55);
    response.timing.pulseDuration = Math.min(response.timing.pulseDuration, 0.35);
    response.timing.afterglowDuration = Math.min(response.timing.afterglowDuration, 0.5);
    response.timing.eventBurstDuration = Math.min(response.timing.eventBurstDuration, 0.35);
  }
  return response;
}

function sanitizeWheelVisualConfig(input = {}, options = {}) {
  const source = input && typeof input === 'object' ? input : {};
  const defaults = DEFAULT_WHEEL_VISUAL_CONFIG;
  const reducedMotion = Boolean(options.reducedMotion);
  const performanceMode = Boolean(options.performanceMode);
  const sharedSegments = source.segments && typeof source.segments === 'object' ? source.segments : {};
  const blueSegmentSource = source.blueSegments && typeof source.blueSegments === 'object' ? source.blueSegments : sharedSegments;
  const orangeSegmentSource = source.orangeSegments && typeof source.orangeSegments === 'object' ? source.orangeSegments : sharedSegments;
  const visual = {
    segments: {
      brightness: clampNumber(source.segments?.brightness, 0, 1, defaults.segments.brightness),
      saturation: clampNumber(source.segments?.saturation, 0, 1.6, defaults.segments.saturation),
      glow: clampNumber(source.segments?.glow, 0, 1, defaults.segments.glow),
    },
    blueSegments: {
      brightness: clampNumber(blueSegmentSource.brightness, 0.5, 1.5, defaults.blueSegments.brightness),
      saturation: clampNumber(blueSegmentSource.saturation, 0, 1.5, defaults.blueSegments.saturation),
      glow: clampNumber(blueSegmentSource.glow, 0, 1, defaults.blueSegments.glow),
      opacity: clampNumber(blueSegmentSource.opacity, 0, 1, defaults.blueSegments.opacity),
    },
    orangeSegments: {
      brightness: clampNumber(orangeSegmentSource.brightness, 0.5, 1.5, defaults.orangeSegments.brightness),
      saturation: clampNumber(orangeSegmentSource.saturation, 0, 1.5, defaults.orangeSegments.saturation),
      glow: clampNumber(orangeSegmentSource.glow, 0, 1, defaults.orangeSegments.glow),
      opacity: clampNumber(orangeSegmentSource.opacity, 0, 1, defaults.orangeSegments.opacity),
    },
    inactiveSegments: {
      opacity: clampNumber(source.inactiveSegments?.opacity, 0, 1, defaults.inactiveSegments.opacity),
      brightness: clampNumber(source.inactiveSegments?.brightness, 0.5, 1.5, defaults.inactiveSegments.brightness),
    },
    seam: {
      intensity: clampNumber(source.seam?.intensity, 0, 1, defaults.seam.intensity),
      flare: clampNumber(source.seam?.flare, 0, 1, defaults.seam.flare),
      flicker: clampNumber(source.seam?.flicker, 0, 1, defaults.seam.flicker),
    },
    frontLine: {
      intensity: clampNumber(source.frontLine?.intensity, 0, 1, defaults.frontLine.intensity),
      coreSize: clampNumber(source.frontLine?.coreSize, 0.45, 1.8, defaults.frontLine.coreSize),
      glowSize: clampNumber(source.frontLine?.glowSize, 0.35, 2, defaults.frontLine.glowSize),
      opacity: clampNumber(source.frontLine?.opacity, 0, 1, defaults.frontLine.opacity),
      trailStrength: clampNumber(source.frontLine?.trailStrength, 0, 1, defaults.frontLine.trailStrength),
      trailDuration: clampNumber(source.frontLine?.trailDuration, 0.25, 3, defaults.frontLine.trailDuration),
      layerPriority: clampNumber(source.frontLine?.layerPriority, 0, 1, defaults.frontLine.layerPriority),
    },
    aura: {
      intensity: clampNumber(source.aura?.intensity, 0, 1, defaults.aura.intensity),
      pulse: clampNumber(source.aura?.pulse, 0, 1, defaults.aura.pulse),
      pulseSpeed: clampNumber(source.aura?.pulseSpeed, 0.5, 2, defaults.aura.pulseSpeed),
      reactiveness: clampNumber(source.aura?.reactiveness, 0, 1, defaults.aura.reactiveness),
      blueStrength: clampNumber(source.aura?.blueStrength, 0, 1, defaults.aura.blueStrength),
      orangeStrength: clampNumber(source.aura?.orangeStrength, 0, 1, defaults.aura.orangeStrength),
      volatilePurpleStrength: clampNumber(source.aura?.volatilePurpleStrength, 0, 1.5, defaults.aura.volatilePurpleStrength),
    },
    volatileAura: {
      intensity: clampNumber(source.volatileAura?.intensity, 0, 1, defaults.volatileAura.intensity),
      saturation: clampNumber(source.volatileAura?.saturation, 0, 1.6, defaults.volatileAura.saturation),
    },
    sparks: {
      intensity: clampNumber(source.sparks?.intensity, 0, 1, defaults.sparks.intensity),
      saturation: clampNumber(source.sparks?.saturation, 0, 1.6, defaults.sparks.saturation),
      opacity: clampNumber(source.sparks?.opacity, 0, 1, defaults.sparks.opacity),
      reactiveness: clampNumber(source.sparks?.reactiveness, 0, 1, defaults.sparks.reactiveness),
    },
    seamSparks: {
      intensity: clampNumber(source.seamSparks?.intensity, 0, 1, defaults.seamSparks.intensity),
      opacity: clampNumber(source.seamSparks?.opacity, 0, 1, defaults.seamSparks.opacity),
      travelDistance: clampNumber(source.seamSparks?.travelDistance, 0.5, 2, defaults.seamSparks.travelDistance),
      duration: clampNumber(source.seamSparks?.duration, 0.5, 2, defaults.seamSparks.duration),
      density: clampNumber(source.seamSparks?.density, 0, 2, defaults.seamSparks.density),
      volatileMultiplier: clampNumber(source.seamSparks?.volatileMultiplier, 0, 2, defaults.seamSparks.volatileMultiplier),
    },
    outerSparks: {
      intensity: clampNumber(source.outerSparks?.intensity, 0, 1, defaults.outerSparks.intensity),
      opacity: clampNumber(source.outerSparks?.opacity, 0, 1, defaults.outerSparks.opacity),
      speed: clampNumber(source.outerSparks?.speed, 0.5, 2, defaults.outerSparks.speed),
      pressureMultiplier: clampNumber(source.outerSparks?.pressureMultiplier, 0, 2, defaults.outerSparks.pressureMultiplier),
      controlMultiplier: clampNumber(source.outerSparks?.controlMultiplier, 0, 2, defaults.outerSparks.controlMultiplier),
      dominantMultiplier: clampNumber(source.outerSparks?.dominantMultiplier, 0, 2, defaults.outerSparks.dominantMultiplier),
    },
    eventReactions: {
      shotPulseStrength: clampNumber(source.eventReactions?.shotPulseStrength, 0, 2, defaults.eventReactions.shotPulseStrength),
      saveFlashStrength: clampNumber(source.eventReactions?.saveFlashStrength, 0, 2, defaults.eventReactions.saveFlashStrength),
      epicSaveAfterglow: clampNumber(source.eventReactions?.epicSaveAfterglow, 0, 2, defaults.eventReactions.epicSaveAfterglow),
      goalFullRingPulseStrength: clampNumber(source.eventReactions?.goalFullRingPulseStrength, 0, 2, defaults.eventReactions.goalFullRingPulseStrength),
      demoJaggedSparkStrength: clampNumber(source.eventReactions?.demoJaggedSparkStrength, 0, 2, defaults.eventReactions.demoJaggedSparkStrength),
    },
    stateMultipliers: {
      neutral: clampNumber(source.stateMultipliers?.neutral, 0, 2, defaults.stateMultipliers.neutral),
      pressure: clampNumber(source.stateMultipliers?.pressure, 0, 2, defaults.stateMultipliers.pressure),
      control: clampNumber(source.stateMultipliers?.control, 0, 2, defaults.stateMultipliers.control),
      volatile: clampNumber(source.stateMultipliers?.volatile, 0, 2, defaults.stateMultipliers.volatile),
      dominant: clampNumber(source.stateMultipliers?.dominant, 0, 2, defaults.stateMultipliers.dominant),
    },
    centerWash: {
      intensity: clampNumber(source.centerWash?.intensity, 0, 1, defaults.centerWash.intensity),
      saturation: clampNumber(source.centerWash?.saturation, 0, 1.6, defaults.centerWash.saturation),
      blueStrength: clampNumber(source.centerWash?.blueStrength, 0, 1, defaults.centerWash.blueStrength),
      orangeStrength: clampNumber(source.centerWash?.orangeStrength, 0, 1, defaults.centerWash.orangeStrength),
      purpleStrength: clampNumber(source.centerWash?.purpleStrength, 0, 1, defaults.centerWash.purpleStrength),
    },
    centerText: {
      brightness: clampNumber(source.centerText?.brightness, 0.72, 1.35, defaults.centerText.brightness),
      confidenceBrightness: clampNumber(source.centerText?.confidenceBrightness, 0.72, 1.35, defaults.centerText.confidenceBrightness),
      scale: clampNumber(source.centerText?.scale, 0.82, 1.22, defaults.centerText.scale),
    },
    innerTicks: {
      opacity: clampNumber(source.innerTicks?.opacity, 0.08, 0.62, defaults.innerTicks.opacity),
      brightness: clampNumber(source.innerTicks?.brightness, 0.5, 1.5, defaults.innerTicks.brightness),
      saturation: clampNumber(source.innerTicks?.saturation, 0, 1.6, defaults.innerTicks.saturation),
    },
    frame: {
      brightness: clampNumber(source.frame?.brightness, 0.5, 1.5, defaults.frame.brightness),
      saturation: clampNumber(source.frame?.saturation, 0, 1.6, defaults.frame.saturation),
      opacity: clampNumber(source.frame?.opacity, 0, 1, defaults.frame.opacity),
    },
    badge: {
      opacity: clampNumber(source.badge?.opacity, 0, 1, defaults.badge.opacity),
    },
  };
  if (reducedMotion) {
    visual.aura.pulse = 0;
    visual.aura.pulseSpeed = 1;
    visual.aura.reactiveness = 0;
    visual.frontLine.trailStrength = 0;
    visual.frontLine.trailDuration = 0.25;
    visual.sparks.intensity = 0;
    visual.sparks.reactiveness = 0;
    visual.seamSparks.intensity = 0;
    visual.outerSparks.intensity = 0;
    visual.outerSparks.speed = 1;
    visual.eventReactions.shotPulseStrength = 0;
    visual.eventReactions.saveFlashStrength = 0;
    visual.eventReactions.epicSaveAfterglow = 0;
    visual.eventReactions.goalFullRingPulseStrength = 0;
    visual.eventReactions.demoJaggedSparkStrength = 0;
    visual.seam.flicker = 0;
    visual.volatileAura.intensity = Math.min(visual.volatileAura.intensity, 0.24);
  }
  if (performanceMode) {
    visual.blueSegments.glow = Math.min(visual.blueSegments.glow, 0.12);
    visual.orangeSegments.glow = Math.min(visual.orangeSegments.glow, 0.12);
    visual.segments.glow = Math.min(visual.segments.glow, 0.12);
    visual.seam.flare = Math.min(visual.seam.flare, 0.18);
    visual.seam.flicker = 0;
    visual.frontLine.glowSize = Math.min(visual.frontLine.glowSize, 0.7);
    visual.frontLine.trailStrength = Math.min(visual.frontLine.trailStrength, 0.08);
    visual.frontLine.trailDuration = Math.min(visual.frontLine.trailDuration, 0.35);
    visual.aura.intensity = Math.min(visual.aura.intensity, 0.12);
    visual.aura.pulse = 0;
    visual.aura.reactiveness = 0;
    visual.volatileAura.intensity = Math.min(visual.volatileAura.intensity, 0.08);
    visual.sparks.intensity = 0;
    visual.sparks.reactiveness = 0;
    visual.sparks.opacity = 0;
    visual.seamSparks.intensity = 0;
    visual.seamSparks.opacity = 0;
    visual.outerSparks.intensity = 0;
    visual.outerSparks.opacity = 0;
    visual.eventReactions.shotPulseStrength = Math.min(visual.eventReactions.shotPulseStrength, 0.2);
    visual.eventReactions.saveFlashStrength = Math.min(visual.eventReactions.saveFlashStrength, 0.2);
    visual.eventReactions.goalFullRingPulseStrength = Math.min(visual.eventReactions.goalFullRingPulseStrength, 0.2);
    visual.eventReactions.epicSaveAfterglow = Math.min(visual.eventReactions.epicSaveAfterglow, 0.15);
    visual.eventReactions.demoJaggedSparkStrength = 0;
    visual.centerWash.intensity = Math.min(visual.centerWash.intensity, 0.25);
  }
  return visual;
}

function clampNumber(value, min, max, fallback) {
  const n = Number(value);
  if (!Number.isFinite(n)) return fallback;
  return Math.min(max, Math.max(min, n));
}

function isValidHexColor(value) {
  return typeof value === 'string' && /^#[0-9a-fA-F]{6}$/.test(value.trim());
}

function sanitizeVariant(value) {
  return ['full', 'compact', 'minimal', 'debug'].includes(value) ? value : 'compact';
}

function sanitizeTheme(value) {
  return Object.prototype.hasOwnProperty.call(MOMENTUM_WHEEL_THEMES, value) ? value : 'oof-default';
}

function sanitizePosition(value) {
  const source = value && typeof value === 'object' ? value : DEFAULT_MOMENTUM_WHEEL_CONFIG.position;
  const preset = ['top-center', 'top-left', 'top-right', 'bottom-center', 'lower-center', 'side-left'].includes(source.preset)
    ? source.preset
    : DEFAULT_MOMENTUM_WHEEL_CONFIG.position.preset;
  const anchor = ['center', 'top-left', 'top-right', 'bottom-center', 'left'].includes(source.anchor)
    ? source.anchor
    : DEFAULT_MOMENTUM_WHEEL_CONFIG.position.anchor;
  return {
    preset,
    x: clampNumber(source.x, -400, 400, 0),
    y: clampNumber(source.y, -400, 400, 0),
    anchor,
  };
}

function sanitizeColorOverrides(value = {}) {
  const enabled = Boolean(value.enabled);
  const safe = {
    enabled,
    blue: null,
    orange: null,
    neutral: null,
    frame: null,
    text: null,
  };
  for (const key of ['blue', 'orange', 'neutral', 'frame', 'text']) {
    if (isValidHexColor(value[key])) safe[key] = value[key].trim();
  }
  return safe;
}

function sanitizeDebugConfig(value = {}) {
  return {
    enabled: Boolean(value.enabled),
    showSegmentIndexes: Boolean(value.showSegmentIndexes),
    showSeamAngle: Boolean(value.showSeamAngle),
    showOwnershipBoundaries: Boolean(value.showOwnershipBoundaries),
    previewState: typeof value.previewState === 'string' ? value.previewState : null,
  };
}

function resolveWheelTheme(config = {}) {
  const theme = { ...MOMENTUM_WHEEL_THEMES[sanitizeTheme(config.theme)] };
  const overrides = config.colorOverrides || {};
  if (overrides.enabled) {
    if (isValidHexColor(overrides.blue)) {
      theme.blueCore = overrides.blue;
      theme.blueGlow = overrides.blue;
    }
    if (isValidHexColor(overrides.orange)) {
      theme.orangeCore = overrides.orange;
      theme.orangeGlow = overrides.orange;
    }
    if (isValidHexColor(overrides.neutral)) {
      theme.textSecondary = overrides.neutral;
      theme.textMuted = overrides.neutral;
    }
    if (isValidHexColor(overrides.frame)) {
      theme.frame = overrides.frame;
    }
    if (isValidHexColor(overrides.text)) {
      theme.textPrimary = overrides.text;
    }
  }
  return theme;
}

function dominantWheelTeam(state, blueShare, orangeShare) {
  if (state === 'BLUE_PRESSURE' || state === 'BLUE_CONTROL') return 'blue';
  if (state === 'ORANGE_PRESSURE' || state === 'ORANGE_CONTROL') return 'orange';
  if (blueShare > orangeShare + 0.04) return 'blue';
  if (orangeShare > blueShare + 0.04) return 'orange';
  return 'neutral';
}

function dominantWheelMode(state) {
  if (state === 'BLUE_PRESSURE' || state === 'ORANGE_PRESSURE') return 'pressure';
  if (state === 'BLUE_CONTROL' || state === 'ORANGE_CONTROL') return 'control';
  if (state === 'VOLATILE') return 'volatile';
  return 'neutral';
}

function wheelDisplayLabel(state = 'NEUTRAL') {
  switch (String(state || 'neutral').toLowerCase().replace(/_/g, '-')) {
    case 'blue-pressure':
    case 'BLUE_PRESSURE': return 'BLUE PRESSURE';
    case 'orange-pressure':
    case 'ORANGE_PRESSURE': return 'ORANGE PRESSURE';
    case 'blue-control':
    case 'BLUE_CONTROL': return 'BLUE CONTROL';
    case 'orange-control':
    case 'ORANGE_CONTROL': return 'ORANGE CONTROL';
    case 'volatile':
    case 'VOLATILE': return 'CONTESTED';
    case 'dominant-blue': return 'BLUE CONTROL';
    case 'dominant-orange': return 'ORANGE CONTROL';
    default: return 'NEUTRAL';
  }
}

function momentumShareDisplayLabel(signal = {}) {
  const values = normalizeMomentumWheelDisplayValues(signal.bluePercent, signal.orangePercent);
  const blue = Math.round(values.bluePercent);
  const orange = Math.round(values.orangePercent);
  if (Math.abs(blue - orange) <= 1) return 'EVEN 50%';
  if (blue > orange) return `BLUE ${blue}%`;
  return `ORANGE ${orange}%`;
}

function confidenceLabel(confidence) {
  const c = clamp01(confidence);
  if (c >= 0.82) return 'Confidence: Max';
  if (c >= 0.62) return 'Confidence: High';
  if (c >= 0.36) return 'Confidence: Medium';
  return 'Confidence: Low';
}

function formatWheelClock(out = {}) {
  const clock = Number(out.display?.matchClock);
  if (Number.isFinite(clock)) return formatClockSeconds(clock);
  const events = Array.isArray(out.debug?.lastEvents) ? out.debug.lastEvents : [];
  for (let i = events.length - 1; i >= 0; i--) {
    const matchClock = Number(events[i]?.matchClock);
    if (Number.isFinite(matchClock)) return formatClockSeconds(matchClock);
  }
  return out.matchClock || '--:--';
}

function formatClockSeconds(seconds) {
  const n = Math.max(0, Math.floor(Number(seconds)));
  const minutes = Math.floor(n / 60);
  const remainder = n % 60;
  return `${minutes}:${String(remainder).padStart(2, '0')}`;
}

function stateClass(state) {
  switch (state) {
    case 'BLUE_PRESSURE': return 'is-blue-pressure';
    case 'ORANGE_PRESSURE': return 'is-orange-pressure';
    case 'BLUE_CONTROL': return 'is-blue-control';
    case 'ORANGE_CONTROL': return 'is-orange-control';
    case 'VOLATILE': return 'is-volatile';
    default: return 'is-neutral';
  }
}

function leftStateLabel(state) {
  switch (state) {
    case 'BLUE_PRESSURE': return 'BLUE PRESSURE';
    case 'BLUE_CONTROL': return 'BLUE CONTROL';
    case 'VOLATILE': return 'CONTESTED';
    case 'NEUTRAL': return 'NEUTRAL';
    default: return '';
  }
}

function rightStateLabel(state) {
  switch (state) {
    case 'ORANGE_PRESSURE': return 'ORANGE PRESSURE';
    case 'ORANGE_CONTROL': return 'ORANGE CONTROL';
    case 'VOLATILE': return 'CONTESTED';
    case 'NEUTRAL': return 'NEUTRAL';
    default: return '';
  }
}

function displayState(state) {
  switch (state) {
    case 'BLUE_PRESSURE':
      return { label: 'Blue Pressure', kind: 'blue' };
    case 'ORANGE_PRESSURE':
      return { label: 'Orange Pressure', kind: 'orange' };
    case 'VOLATILE':
      return { label: 'Contested', kind: 'contested' };
    case 'BLUE_CONTROL':
      return { label: 'Blue Momentum', kind: 'blue' };
    case 'ORANGE_CONTROL':
      return { label: 'Orange Momentum', kind: 'orange' };
    default:
      return { label: 'Neutral', kind: 'neutral' };
  }
}

function displayEvent(type) {
  switch (type) {
    case 'goal': return 'Goal';
    case 'shot': return 'Shot';
    case 'save': return 'Save';
    case 'demo': return 'Demo';
    case 'assist': return 'Assist';
    default: return 'Event';
  }
}

function shortEventLabel(type) {
  switch (type) {
    case 'goal': return 'G';
    case 'shot': return 'SH';
    case 'save': return 'SV';
    case 'demo': return 'DM';
    case 'assist': return 'A';
    default: return 'E';
  }
}

function displayTeam(team) {
  if (team === 'blue') return 'Blue';
  if (team === 'orange') return 'Orange';
  return '';
}

function demoOutput(kind) {
  switch (kind) {
    case 'blue':
      return { state: 'BLUE_PRESSURE', confidence: 0.74, volatility: 0.22, matchClock: '2:14', blue: { pressureShare: 0.62 }, orange: { pressureShare: 0.38 }, overlay: { momentumBarBluePercent: 62, momentumBarOrangePercent: 38, pulse: 'SHOT', pulseTeam: 'blue' } };
    case 'orange':
      return { state: 'ORANGE_PRESSURE', confidence: 0.81, volatility: 0.26, blue: { pressureShare: 0.28 }, orange: { pressureShare: 0.72 }, overlay: { momentumBarBluePercent: 28, momentumBarOrangePercent: 72, pulse: 'SAVE_FORCED', pulseTeam: 'orange' } };
    case 'volatile':
      return { state: 'VOLATILE', confidence: 0.58, volatility: 0.82, blue: { pressureShare: 0.55 }, orange: { pressureShare: 0.45 }, overlay: { momentumBarBluePercent: 55, momentumBarOrangePercent: 45, pulse: 'VOLATILE_CONTEST' } };
    default:
      return { state: 'NEUTRAL', confidence: 0.42, volatility: 0.18, blue: { pressureShare: 0.5 }, orange: { pressureShare: 0.5 }, overlay: { momentumBarBluePercent: 50, momentumBarOrangePercent: 50 } };
  }
}

function weightSummary(w) {
  const deltas = [];
  if (w.controlDelta !== undefined) deltas.push(`control ${signed(w.controlDelta)}`);
  if (w.pressureDelta !== undefined) deltas.push(`pressure ${signed(w.pressureDelta)}`);
  if (w.volatilityDelta !== undefined) deltas.push(`volatility ${signed(w.volatilityDelta)}`);
  const team = w.team ? ` ${w.team}` : '';
  return `${w.eventType || 'event'}${team}: ${deltas.join(', ') || 'no delta'} - ${w.reason || 'no reason'}`;
}

function strongEventSummary(ev) {
  if (!ev) return 'None';
  const who = ev.playerName || ev.playerId || 'unknown';
  return `${ev.type || 'event'} / ${ev.team || 'team?'} / ${who}`;
}

function countSummary(counts) {
  if (!counts || typeof counts !== 'object') return '';
  return Object.keys(counts)
    .sort()
    .map(key => `${key}: ${counts[key]}`)
    .join('  |  ');
}

function pulseIdentity(out, signals) {
  if (!signals.pulse) return '';
  const ev = out.debug?.lastStrongEvent;
  if (ev && ev.time) {
    return `${signals.pulse}:${ev.type || ''}:${ev.team || ''}:${ev.time}`;
  }
  return `${signals.pulse}:${out.state || ''}:${signals.pulseTeam || ''}`;
}

function signed(value) {
  const n = Number(value || 0);
  return `${n >= 0 ? '+' : ''}${n.toFixed(2)}`;
}

function setText(id, value) {
  const el = document.getElementById(id);
  if (el) el.textContent = value;
}

function pct(value) {
  const n = Number(value || 0);
  return `${Math.round(n * 100)}%`;
}

function num(value) {
  const n = Number(value || 0);
  return n.toFixed(2);
}

function clampPercent(value) {
  const n = Number(value);
  if (!Number.isFinite(n)) return 50;
  return Math.max(0, Math.min(100, n));
}

function clamp01(value) {
  const n = Number(value);
  if (!Number.isFinite(n)) return 0;
  return Math.max(0, Math.min(1, n));
}

function clampRange(value, min, max, fallback) {
  const n = Number(value);
  if (!Number.isFinite(n)) return fallback;
  return Math.max(min, Math.min(max, n));
}

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#039;');
}
