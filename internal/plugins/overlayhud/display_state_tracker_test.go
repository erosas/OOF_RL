package overlayhud

import (
	"testing"
	"time"
)

func TestDisplayStateTrackerHoldsPressureBeforeControl(t *testing.T) {
	now := time.Unix(100, 0)
	tracker := NewDisplayStateTracker()

	pressure := ViewModel{
		MatchActive:  true,
		HasData:      true,
		BlueShare:    0.62,
		OrangeShare:  0.38,
		DisplayState: displayStateBluePressure,
		StateLabel:   "BLUE PRESSURE",
		LastUpdated:  now,
	}
	control := pressure
	control.BlueShare = 0.72
	control.OrangeShare = 0.28
	control.DisplayState = displayStateBlueControl
	control.StateLabel = "BLUE CONTROL"

	first := tracker.Apply(pressure, now)
	held := tracker.Apply(control, now.Add(displayStateMinHold/2))
	released := tracker.Apply(control, now.Add(displayStateMinHold+time.Millisecond))

	if first.DisplayState != displayStateBluePressure {
		t.Fatalf("first state = %q, want pressure", first.DisplayState)
	}
	if held.DisplayState != displayStateBluePressure || held.StateLabel != "BLUE PRESSURE" {
		t.Fatalf("held state = %q label = %q, want blue pressure", held.DisplayState, held.StateLabel)
	}
	if released.DisplayState != displayStateBlueControl || released.StateLabel != "BLUE CONTROL" {
		t.Fatalf("released state = %q label = %q, want blue control", released.DisplayState, released.StateLabel)
	}
}

func TestDisplayStateTrackerSuppressesRapidTeamFlip(t *testing.T) {
	now := time.Unix(100, 0)
	tracker := NewDisplayStateTracker()
	blue := ViewModel{
		MatchActive:  true,
		HasData:      true,
		BlueShare:    0.64,
		OrangeShare:  0.36,
		DisplayState: displayStateBluePressure,
		StateLabel:   "BLUE PRESSURE",
		LastUpdated:  now,
	}
	orange := blue
	orange.BlueShare = 0.36
	orange.OrangeShare = 0.64
	orange.DisplayState = displayStateOrangePressure
	orange.StateLabel = "ORANGE PRESSURE"

	tracker.Apply(blue, now)
	held := tracker.Apply(orange, now.Add(displayContestedFlipSuppress/2))
	released := tracker.Apply(orange, now.Add(displayStateMinHold+time.Millisecond))

	if held.DisplayState != displayStateBluePressure {
		t.Fatalf("rapid flip state = %q, want held blue pressure", held.DisplayState)
	}
	if released.DisplayState != displayStateOrangePressure {
		t.Fatalf("released flip state = %q, want orange pressure", released.DisplayState)
	}
}

func TestDisplayStateTrackerResetsOnNoDataOrStale(t *testing.T) {
	now := time.Unix(100, 0)
	tracker := NewDisplayStateTracker()
	active := ViewModel{
		MatchActive:  true,
		HasData:      true,
		BlueShare:    0.72,
		OrangeShare:  0.28,
		DisplayState: displayStateBlueControl,
		StateLabel:   "BLUE CONTROL",
		LastUpdated:  now,
	}
	noData := ViewModel{
		DisplayState: displayStateNoData,
		StateLabel:   "NO DATA",
		IsStale:      true,
	}

	tracker.Apply(active, now)
	reset := tracker.Apply(noData, now.Add(time.Second))
	next := tracker.Apply(active, now.Add(time.Second+time.Millisecond))

	if reset.DisplayState != displayStateNoData {
		t.Fatalf("reset state = %q, want no-data", reset.DisplayState)
	}
	if next.DisplayState != displayStateBlueControl {
		t.Fatalf("next state = %q, want immediate blue control after reset", next.DisplayState)
	}
}

func TestDisplayStateTrackerKeepsRecentEventPulseDuringWindow(t *testing.T) {
	now := time.Unix(100, 0)
	tracker := NewDisplayStateTracker()
	withPulse := ViewModel{
		MatchActive:       true,
		HasData:           true,
		DisplayState:      displayStateNeutral,
		StateLabel:        "NEUTRAL",
		RecentEventEnergy: 1,
		RecentEventTeam:   "blue",
		RecentEventType:   "goal",
		LastUpdated:       now,
	}
	noPulse := withPulse
	noPulse.RecentEventEnergy = 0
	noPulse.RecentEventTeam = ""
	noPulse.RecentEventType = ""

	tracker.Apply(withPulse, now)
	held := tracker.Apply(noPulse, now.Add(displayRecentEventHold/2))
	expired := tracker.Apply(noPulse, now.Add(displayRecentEventHold+time.Millisecond))

	if held.RecentEventEnergy <= 0 || held.RecentEventTeam != "blue" || held.RecentEventType != "goal" {
		t.Fatalf("held recent event = %f/%q/%q, want blue goal pulse", held.RecentEventEnergy, held.RecentEventTeam, held.RecentEventType)
	}
	if expired.RecentEventEnergy != 0 || expired.RecentEventTeam != "" || expired.RecentEventType != "" {
		t.Fatalf("expired recent event = %f/%q/%q, want cleared", expired.RecentEventEnergy, expired.RecentEventTeam, expired.RecentEventType)
	}
}
