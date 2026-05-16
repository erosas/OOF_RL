package momentum

import (
	"sync"
	"testing"

	"OOF_RL/internal/oofevents"
)

func TestServiceHandleGameActionUpdatesSnapshot(t *testing.T) {
	service := NewService(Config{Decay: 1})

	state := service.HandleGameAction(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"))
	snapshot := service.Snapshot()

	if state.Sequence != 1 || snapshot.Sequence != 1 {
		t.Fatalf("sequence = state %d snapshot %d, want 1", state.Sequence, snapshot.Sequence)
	}
	if snapshot.Teams[oofevents.TeamBlue].MomentumInfluence <= 0 {
		t.Fatalf("blue signal not updated: %+v", snapshot.Teams[oofevents.TeamBlue])
	}
}

func TestServiceSnapshotReturnsCopy(t *testing.T) {
	service := NewService(Config{Decay: 1})
	snapshot := service.HandleGameAction(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"))
	snapshot.Teams[oofevents.TeamBlue] = TeamSignal{}

	next := service.Snapshot()
	if next.Teams[oofevents.TeamBlue].MomentumInfluence == 0 {
		t.Fatal("mutating returned snapshot should not mutate service state")
	}
}

func TestServiceResetClearsStateAndReactivates(t *testing.T) {
	service := NewService(Config{Decay: 1})
	service.HandleGameAction(oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"))
	service.MarkMatchEnded()

	state := service.Reset("manual")
	state = service.HandleGameAction(oofevents.NewGameAction("match-2", oofevents.ActionBallHit, oofevents.TeamOrange, "pid-b", "Bob"))

	status := service.Status()
	if !status.Active || status.Reason != "manual" {
		t.Fatalf("status = %+v, want active manual reset", status)
	}
	if state.MatchGUID != "match-2" {
		t.Fatalf("MatchGUID = %q, want match-2", state.MatchGUID)
	}
	if state.Sequence != 1 {
		t.Fatalf("Sequence = %d, want 1 after reset and one event", state.Sequence)
	}
	if state.Teams[oofevents.TeamBlue].MomentumInfluence != 0 {
		t.Fatalf("blue signal carried across reset: %+v", state.Teams[oofevents.TeamBlue])
	}
}

func TestServiceLifecycleResetBoundaries(t *testing.T) {
	service := NewService(Config{Decay: 1})
	service.HandleGameAction(oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"))

	state := service.HandleMatchStarted(oofevents.NewMatchStarted("match-2"))
	if state.Sequence != 0 || state.MatchGUID != "" {
		t.Fatalf("match.started reset state = %+v, want empty", state)
	}

	service.HandleGameAction(oofevents.NewGameAction("match-2", oofevents.ActionGoal, oofevents.TeamOrange, "pid-b", "Bob"))
	state = service.HandleMatchRestarted(oofevents.NewMatchRestarted("match-3", "match-2"))
	if state.Sequence != 0 || state.MatchGUID != "" {
		t.Fatalf("match.restarted reset state = %+v, want empty", state)
	}

	service.HandleGameAction(oofevents.NewGameAction("match-3", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"))
	state = service.HandleMatchDestroyed(oofevents.NewMatchDestroyed())
	if state.Sequence != 0 || state.MatchGUID != "" {
		t.Fatalf("match.destroyed reset state = %+v, want empty", state)
	}
}

func TestServiceMarkMatchEndedFreezesSnapshotUntilReset(t *testing.T) {
	service := NewService(Config{Decay: 1})
	service.HandleGameAction(oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"))

	ended := service.HandleMatchEnded(oofevents.NewMatchEnded("match-1", 0))
	ignored := service.HandleGameAction(oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamOrange, "pid-b", "Bob"))

	status := service.Status()
	if status.Active || status.Reason != "match.ended:match-1" {
		t.Fatalf("status = %+v, want inactive match.ended:match-1", status)
	}
	if ignored.Sequence != ended.Sequence {
		t.Fatalf("inactive service sequence = %d, want frozen %d", ignored.Sequence, ended.Sequence)
	}
	if ignored.Teams[oofevents.TeamOrange].MomentumInfluence != 0 {
		t.Fatalf("inactive service applied new event: %+v", ignored.Teams[oofevents.TeamOrange])
	}
}

func TestServiceConcurrentAccess(t *testing.T) {
	service := NewService(Config{Decay: 1})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			service.HandleGameAction(oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamBlue, "pid-a", "Alice"))
		}()
		go func() {
			defer wg.Done()
			_ = service.Snapshot()
		}()
	}
	wg.Wait()

	if service.Snapshot().Sequence == 0 {
		t.Fatal("expected at least one applied action")
	}
}
