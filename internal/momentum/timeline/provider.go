package timeline

// SnapshotProvider is the read-only Timeline access contract for internal
// consumers. It intentionally excludes lifecycle and event ingestion methods.
type SnapshotProvider interface {
	Snapshot() TimelineSnapshot
}
