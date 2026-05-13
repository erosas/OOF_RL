package oofevents

import "time"

// Certainty describes how confident we are in an event's truth.
type Certainty int

const (
	// Authoritative: directly observed — no inference required.
	Authoritative Certainty = iota
	// Inferred: deterministically derived from authoritative data,
	// or corroborated by multiple independent sources.
	Inferred
	// Signal: a single heuristic data point. May be wrong.
	// Accumulate; don't act until corroborated.
	Signal
)

func (c Certainty) String() string {
	switch c {
	case Authoritative:
		return "authoritative"
	case Inferred:
		return "inferred"
	case Signal:
		return "signal"
	default:
		return "unknown"
	}
}

// Source identifies who emitted an event.
// PluginID == "" means the event came from the RL API translator.
type Source struct {
	PluginID string
	Label    string
}

// OOFEvent is the interface every event in the system satisfies.
type OOFEvent interface {
	Type()       string
	OccurredAt() time.Time
	Certainty()  Certainty
	Source()     Source
	MatchGUID()  string
}

// Base can be embedded in concrete event types to satisfy OOFEvent.
// Callers should not set Src directly — the bus stamps it on publish.
type Base struct {
	EventType string
	At        time.Time
	Cert      Certainty
	Src       Source
	GUID      string
}

func (b Base) Type()       string    { return b.EventType }
func (b Base) OccurredAt() time.Time { return b.At }
func (b Base) Certainty()  Certainty { return b.Cert }
func (b Base) Source()     Source    { return b.Src }
func (b Base) MatchGUID()  string    { return b.GUID }


// EventDeclaration describes an event type a plugin may emit.
type EventDeclaration struct {
	Type        string
	Certainty   Certainty
	Description string
}

// Subscription is returned by Subscribe calls. Cancel must be called on shutdown.
type Subscription interface {
	Cancel()
}