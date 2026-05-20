package overlayhud

import (
	"testing"
	"time"

	"OOF_RL/internal/momentum"
	"OOF_RL/internal/oofevents"
)

func TestSignalDisplayPipelineReachesControlFromBallHitChain(t *testing.T) {
	start := time.Unix(100, 0)
	engine := momentum.NewEngine(momentum.Config{Decay: 1})
	tracker := NewDisplayStateTracker()

	state := engine.ApplyGameAction(atPipeline(oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamBlue, "pid-a", "Alice"), start))
	state = engine.ApplyGameAction(atPipeline(oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamBlue, "pid-a", "Alice"), start.Add(time.Second)))
	state = engine.ApplyGameAction(atPipeline(oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamBlue, "pid-a", "Alice"), start.Add(2*time.Second)))

	vm := mapMomentumViewModel(state, momentum.ServiceStatus{Active: true}, start.Add(2*time.Second))
	stabilized := tracker.Apply(vm, start.Add(2*time.Second))
	model := buildRenderModel(stabilized)

	if vm.DisplayState != displayStateBlueControl {
		t.Fatalf("raw ViewModel state = %q, want blue control; vm=%+v state=%+v", vm.DisplayState, vm, state.Teams[oofevents.TeamBlue])
	}
	if model.DisplayState != displayStateBlueControl {
		t.Fatalf("RenderModel state = %q, want blue control", model.DisplayState)
	}
}

func TestSignalDisplayPipelineReachesContestedFromAlternatingTouches(t *testing.T) {
	start := time.Unix(100, 0)
	engine := momentum.NewEngine(momentum.Config{Decay: 1})
	tracker := NewDisplayStateTracker()

	state := engine.ApplyGameAction(atPipeline(oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamBlue, "pid-a", "Alice"), start))
	state = engine.ApplyGameAction(atPipeline(oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamOrange, "pid-b", "Bob"), start.Add(time.Second)))

	vm := mapMomentumViewModel(state, momentum.ServiceStatus{Active: true}, start.Add(time.Second))
	stabilized := tracker.Apply(vm, start.Add(time.Second))
	model := buildRenderModel(stabilized)

	if vm.DisplayState != displayStateVolatile {
		t.Fatalf("raw ViewModel state = %q label = %q, want volatile/CONTESTED; vm=%+v blue=%+v orange=%+v", vm.DisplayState, vm.StateLabel, vm, state.Teams[oofevents.TeamBlue], state.Teams[oofevents.TeamOrange])
	}
	if model.DisplayState != displayStateVolatile || model.Center.StateLabel != "CONTESTED" {
		t.Fatalf("RenderModel state = %q label = %q, want volatile/CONTESTED", model.DisplayState, model.Center.StateLabel)
	}
}

func TestSignalDisplayPipelineContestedBypassesHeldPressure(t *testing.T) {
	start := time.Unix(100, 0)
	engine := momentum.NewEngine(momentum.Config{Decay: 1})
	tracker := NewDisplayStateTracker()

	shotState := engine.ApplyGameAction(atPipeline(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"), start))
	tracker.Apply(mapMomentumViewModel(shotState, momentum.ServiceStatus{Active: true}, start), start)

	state := engine.ApplyGameAction(atPipeline(oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamBlue, "pid-a", "Alice"), start.Add(time.Second)))
	state = engine.ApplyGameAction(atPipeline(oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamOrange, "pid-b", "Bob"), start.Add(2*time.Second)))
	vm := mapMomentumViewModel(state, momentum.ServiceStatus{Active: true}, start.Add(2*time.Second))
	stabilized := tracker.Apply(vm, start.Add(2*time.Second))
	model := buildRenderModel(stabilized)

	if vm.DisplayState != displayStateVolatile {
		t.Fatalf("raw ViewModel state = %q, want volatile/CONTESTED; vm=%+v", vm.DisplayState, vm)
	}
	if model.DisplayState != displayStateVolatile || model.Center.StateLabel != "CONTESTED" {
		t.Fatalf("held RenderModel state = %q label = %q, want volatile/CONTESTED", model.DisplayState, model.Center.StateLabel)
	}
}

func TestSignalDisplayPipelineGoalWindowForcesPressureOverControl(t *testing.T) {
	start := time.Unix(100, 0)
	engine := momentum.NewEngine(momentum.Config{Decay: 1})
	tracker := NewDisplayStateTracker()

	for i := 0; i < 4; i++ {
		engine.ApplyGameAction(atPipeline(
			oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamBlue, "pid-a", "Alice"),
			start.Add(time.Duration(i)*time.Second),
		))
	}
	state := engine.ApplyGameAction(atPipeline(oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"), start.Add(4*time.Second)))
	vm := mapMomentumViewModel(state, momentum.ServiceStatus{Active: true}, start.Add(4*time.Second))
	model := buildRenderModel(tracker.Apply(vm, start.Add(4*time.Second)))

	if vm.DisplayState != displayStateBluePressure {
		t.Fatalf("goal ViewModel state = %q, want blue pressure; vm=%+v blue=%+v", vm.DisplayState, vm, state.Teams[oofevents.TeamBlue])
	}
	if model.DisplayState != displayStateBluePressure || model.Center.StateLabel != "BLUE PRESSURE" {
		t.Fatalf("goal RenderModel state = %q label = %q, want blue pressure", model.DisplayState, model.Center.StateLabel)
	}
}

func TestSignalDisplayPipelineSaveWindowUsesAttackingPressure(t *testing.T) {
	start := time.Unix(100, 0)
	engine := momentum.NewEngine(momentum.Config{Decay: 1})
	tracker := NewDisplayStateTracker()

	state := engine.ApplyGameAction(atPipeline(oofevents.NewGameAction("match-1", oofevents.ActionSave, oofevents.TeamBlue, "pid-a", "Alice"), start))
	vm := mapMomentumViewModel(state, momentum.ServiceStatus{Active: true}, start)
	model := buildRenderModel(tracker.Apply(vm, start))

	if vm.DisplayState != displayStateOrangePressure {
		t.Fatalf("save ViewModel state = %q, want orange pressure; vm=%+v blue=%+v orange=%+v", vm.DisplayState, vm, state.Teams[oofevents.TeamBlue], state.Teams[oofevents.TeamOrange])
	}
	if model.DisplayState != displayStateOrangePressure || model.Center.StateLabel != "ORANGE PRESSURE" {
		t.Fatalf("save RenderModel state = %q label = %q, want orange pressure", model.DisplayState, model.Center.StateLabel)
	}
}

func TestSignalDisplayPipelineKeepsNearGoalPressureThroughHold(t *testing.T) {
	start := time.Unix(100, 0)
	engine := momentum.NewEngine(momentum.Config{Decay: 1})
	tracker := NewDisplayStateTracker()

	shotState := engine.ApplyGameAction(atPipeline(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"), start))
	shotVM := mapMomentumViewModel(shotState, momentum.ServiceStatus{Active: true}, start)
	held := tracker.Apply(shotVM, start)

	chainState := engine.ApplyGameAction(atPipeline(oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamBlue, "pid-a", "Alice"), start.Add(time.Second)))
	chainState = engine.ApplyGameAction(atPipeline(oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamBlue, "pid-a", "Alice"), start.Add(2*time.Second)))
	chainState = engine.ApplyGameAction(atPipeline(oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamBlue, "pid-a", "Alice"), start.Add(3*time.Second)))
	chainVM := mapMomentumViewModel(chainState, momentum.ServiceStatus{Active: true}, start.Add(3*time.Second))
	stillHeld := tracker.Apply(chainVM, start.Add(3*time.Second))
	released := tracker.Apply(chainVM, start.Add(displayStateMinHold+time.Millisecond))

	if held.DisplayState != displayStateBluePressure {
		t.Fatalf("first state = %q, want pressure", held.DisplayState)
	}
	if chainVM.DisplayState != displayStateBluePressure {
		t.Fatalf("raw chain state = %q, want pressure while shot pressure remains; vm=%+v blue=%+v", chainVM.DisplayState, chainVM, chainState.Teams[oofevents.TeamBlue])
	}
	if stillHeld.DisplayState != displayStateBluePressure {
		t.Fatalf("state before hold release = %q, want held pressure", stillHeld.DisplayState)
	}
	if released.DisplayState != displayStateBluePressure {
		t.Fatalf("state after hold release = %q, want pressure while pressure threshold remains active", released.DisplayState)
	}
}

func TestSignalDisplayPipelineControlTakesOverAfterSustainedTouches(t *testing.T) {
	start := time.Unix(100, 0)
	engine := momentum.NewEngine(momentum.Config{Decay: 1})
	tracker := NewDisplayStateTracker()

	var state momentum.MomentumState
	for i := 0; i <= 3; i++ {
		state = engine.ApplyGameAction(atPipeline(
			oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamBlue, "pid-a", "Alice"),
			start.Add(time.Duration(i)*time.Second),
		))
	}

	now := start.Add(displayStateMinHold + time.Second)
	vm := mapMomentumViewModel(state, momentum.ServiceStatus{Active: true}, now)
	stabilized := tracker.Apply(vm, now)
	model := buildRenderModel(stabilized)

	if vm.DisplayState != displayStateBlueControl {
		t.Fatalf("raw ViewModel state = %q, want blue control after sustained touches; vm=%+v blue=%+v", vm.DisplayState, vm, state.Teams[oofevents.TeamBlue])
	}
	if model.DisplayState != displayStateBlueControl {
		t.Fatalf("RenderModel state = %q, want blue control", model.DisplayState)
	}
	if model.Diagnostics.BlueControlShare < 0.70 {
		t.Fatalf("blue control share = %f, want >= 0.70", model.Diagnostics.BlueControlShare)
	}
}

func atPipeline(event oofevents.GameActionEvent, t time.Time) oofevents.GameActionEvent {
	event.Base.At = t
	return event
}
