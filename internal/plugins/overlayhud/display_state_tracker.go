package overlayhud

import "time"

const (
	displayStateMinHold          = 3600 * time.Millisecond
	displayContestedFlipSuppress = 3200 * time.Millisecond
	displayRecentEventHold       = 1200 * time.Millisecond
)

// DisplayStateTracker owns display-layer temporal smoothing. It consumes an
// already-built ViewModel and returns a stabilized ViewModel for rendering.
type DisplayStateTracker struct {
	currentState     string
	currentLabel     string
	currentTeam      string
	stateChangedAt   time.Time
	lastRecentEnergy float64
	lastRecentTeam   string
	lastRecentType   string
	lastRecentAt     time.Time
}

func NewDisplayStateTracker() *DisplayStateTracker {
	return &DisplayStateTracker{}
}

func (t *DisplayStateTracker) Apply(vm ViewModel, now time.Time) ViewModel {
	if t == nil {
		return vm
	}
	if shouldResetDisplayState(vm) {
		t.reset()
		return vm
	}

	vm = t.applyStateHold(vm, now)
	vm = t.applyRecentEventHold(vm, now)
	return vm
}

func (t *DisplayStateTracker) applyStateHold(vm ViewModel, now time.Time) ViewModel {
	nextState := vm.DisplayState
	nextLabel := vm.StateLabel
	nextTeam := displayStateTeam(nextState)
	if t.currentState == "" {
		t.rememberState(nextState, nextLabel, nextTeam, now)
		return vm
	}
	if nextState == t.currentState {
		t.currentLabel = nextLabel
		return vm
	}
	if shouldBypassStateHold(vm) {
		t.rememberState(nextState, nextLabel, nextTeam, now)
		return vm
	}

	heldFor := now.Sub(t.stateChangedAt)
	if heldFor < displayStateMinHold || isRapidOpposingFlip(t.currentTeam, nextTeam, heldFor) {
		vm.DisplayState = t.currentState
		vm.StateLabel = t.currentLabel
		return vm
	}

	t.rememberState(nextState, nextLabel, nextTeam, now)
	return vm
}

func (t *DisplayStateTracker) applyRecentEventHold(vm ViewModel, now time.Time) ViewModel {
	if vm.RecentEventEnergy > 0 && vm.RecentEventTeam != "" && vm.RecentEventType != "" {
		t.lastRecentEnergy = vm.RecentEventEnergy
		t.lastRecentTeam = vm.RecentEventTeam
		t.lastRecentType = vm.RecentEventType
		t.lastRecentAt = now
		return vm
	}
	if t.lastRecentEnergy <= 0 || t.lastRecentAt.IsZero() || now.Sub(t.lastRecentAt) > displayRecentEventHold {
		return vm
	}
	remaining := 1 - now.Sub(t.lastRecentAt).Seconds()/displayRecentEventHold.Seconds()
	vm.RecentEventEnergy = clamp01(t.lastRecentEnergy * remaining)
	vm.RecentEventTeam = t.lastRecentTeam
	vm.RecentEventType = t.lastRecentType
	return vm
}

func (t *DisplayStateTracker) rememberState(state, label, team string, now time.Time) {
	t.currentState = state
	t.currentLabel = label
	t.currentTeam = team
	t.stateChangedAt = now
}

func (t *DisplayStateTracker) reset() {
	*t = DisplayStateTracker{}
}

func shouldResetDisplayState(vm ViewModel) bool {
	return !vm.HasData || !vm.MatchActive || vm.IsStale ||
		vm.DisplayState == displayStateNoData ||
		vm.DisplayState == displayStateInactive ||
		vm.DisplayState == displayStateStale
}

func shouldBypassStateHold(vm ViewModel) bool {
	if vm.DisplayState == displayStateVolatile {
		return true
	}
	return vm.RecentEventEnergy > 0 && isPressureEvent(vm.RecentEventType) &&
		(vm.DisplayState == displayStateBluePressure || vm.DisplayState == displayStateOrangePressure)
}

func isRapidOpposingFlip(currentTeam, nextTeam string, heldFor time.Duration) bool {
	return currentTeam != "" && nextTeam != "" && currentTeam != nextTeam && heldFor < displayContestedFlipSuppress
}

func displayStateTeam(state string) string {
	switch state {
	case displayStateBluePressure, displayStateBlueControl, displayStateDominantBlue:
		return "blue"
	case displayStateOrangePressure, displayStateOrangeControl, displayStateDominantOrange:
		return "orange"
	default:
		return ""
	}
}
