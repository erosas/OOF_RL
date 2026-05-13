package oofevents

// PluginBus is the bus surface given to plugins and the RL translator in Init.
// Each caller receives an instance pre-configured with its plugin ID; the bus
// stamps Source.PluginID automatically on every Publish call.
type PluginBus interface {
	// PublishAuthoritative emits a direct, unambiguous fact.
	// Panics if e.Certainty() != Authoritative.
	PublishAuthoritative(e OOFEvent)

	// PublishInferred emits a high-confidence derived event.
	// Panics if e.Certainty() != Inferred.
	PublishInferred(e OOFEvent)

	// PublishSignal emits a heuristic data point.
	// Panics if e.Certainty() != Signal.
	PublishSignal(e OOFEvent)

	// Subscribe registers fn for events of the given type.
	// The returned Subscription must be cancelled on shutdown.
	Subscribe(eventType string, fn func(OOFEvent)) Subscription

	// SubscribeAll registers fn for every event on the bus.
	SubscribeAll(fn func(OOFEvent)) Subscription

	// SubscribeMinCertainty registers fn for events of eventType at or above min.
	SubscribeMinCertainty(eventType string, min Certainty, fn func(OOFEvent)) Subscription
}

// Bus is the full internal bus used by application bootstrap.
// Plugins receive PluginBus; only the server holds Bus.
type Bus interface {
	PluginBus
	Start() error
	Stop()
	// ResetMatch clears corroboration state for the given match GUID.
	// Called automatically when a match.ended event is published.
	ResetMatch(guid string)
	// ForPlugin returns a PluginBus scoped to the given plugin ID.
	// The returned bus stamps that ID as Source.PluginID on every publish.
	ForPlugin(pluginID string) PluginBus
}