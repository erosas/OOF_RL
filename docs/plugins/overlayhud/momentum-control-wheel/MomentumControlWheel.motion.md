# MomentumControlWheel Motion Manifest

Scope: UI only. Motion must visualize already-computed Momentum Engine output and must not introduce match-state logic.

## Always-On Motion

| Element | Motion |
| --- | --- |
| Outer aura | Slow opacity breathing |
| Center rim | Very subtle glow pulse |
| Segment highlights | Gentle shimmer on active side |
| Contested flare | Small pulse every 1-2 seconds |

## Event-Based Motion

| Event | Motion |
| --- | --- |
| Momentum shift | Active segments interpolate to new values |
| Team gains pressure | Dominant side glow increases |
| Volatility spike | Purple flicker near seam |
| Contest collision | Purple/blue/orange radial shockwave from seam |
| Goal / major event | Brief full-ring pulse |
| Confidence increase | Center rim brightens |
| Confidence decrease | Center glow softens |

## Animation Timing

```json
{
  "segmentTransition": "180ms ease-out",
  "centerWashTransition": "240ms ease-out",
  "outerGlowTransition": "300ms ease-out",
  "contestPulse": "600ms ease-in-out",
  "volatileFlicker": "80-140ms randomized",
  "shockwave": "700-1000ms cubic-bezier(0.2, 0.85, 0.25, 1)"
}
```

## State Motion Rules

### Neutral

- Keep seam centered at top.
- Use balanced blue/orange glow.
- Keep contest pulse low and slow.

### Blue Pressure

- Increase blue aura and blue segment brightness.
- Move seam toward orange territory based on blue percentage.
- Center wash should lean blue at pressure opacity.

### Orange Pressure

- Increase orange aura and orange segment brightness.
- Move seam toward blue territory based on orange percentage.
- Center wash should lean orange at pressure opacity.

### Blue Control

- Use stronger blue aura and outside blue streaks.
- Keep orange readable as a compressed opposing territory.
- Center wash should use control opacity.

### Orange Control

- Use stronger orange aura and outside orange streaks.
- Keep blue readable as a compressed opposing territory.
- Center wash should use control opacity.

### Volatile

- Increase purple seam glow.
- Enable seam flicker.
- Enable alternating blue/orange sparks near seam.
- Enable radial contest shockwave when the wheel is close to 50/50.
- Do not make the whole wheel constantly shake.

### Dominant Blue

- Almost full blue ring.
- Opposing orange sliver remains visible unless blue is exactly 100%.
- Strong blue center bloom.
- Full-ring pulse may breathe at max confidence.

### Dominant Orange

- Almost full orange ring.
- Opposing blue sliver remains visible unless orange is exactly 100%.
- Strong orange center bloom.
- Full-ring pulse may breathe at max confidence.

## Accessibility / Performance

- Respect reduced-motion settings.
- Keep aura, streaks, sparks, and shockwaves independently toggleable.
- If performance mode is enabled, disable sparks first, then shockwave, then shimmer.
- Text must remain readable during all animations.
