package momentum

import (
	"encoding/json"
	"testing"
	"time"

	"OOF_RL/internal/events"
)

func TestNormalizeGoalScoredIgnoresReplayEndPacket(t *testing.T) {
	data, err := json.Marshal(events.GoalScoredData{
		MatchGuid: "match-1",
		Scorer: events.PlayerRef{
			TeamNum: 0,
		},
		BallLastTouch: events.LastTouch{
			Player: events.PlayerRef{
				Name:    "Real scorer",
				TeamNum: 1,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	got := NormalizeEnvelope(events.Envelope{Event: "GoalScored", Data: data}, time.UnixMilli(1000))

	if len(got) != 0 {
		t.Fatalf("replay-end GoalScored packet should be ignored, got %+v", got)
	}
}

func TestNormalizeGoalScoredWithRealScorer(t *testing.T) {
	data, err := json.Marshal(events.GoalScoredData{
		MatchGuid: "match-1",
		GoalTime:  42,
		Scorer: events.PlayerRef{
			Name:      "Mr Mung Beans",
			PrimaryId: "pid",
			TeamNum:   1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	got := NormalizeEnvelope(events.Envelope{Event: "GoalScored", Data: data}, time.UnixMilli(1000))

	if len(got) != 1 || got[0].Type != EventGoal || got[0].Team != TeamOrange {
		t.Fatalf("real GoalScored should normalize as orange goal, got %+v", got)
	}
}
