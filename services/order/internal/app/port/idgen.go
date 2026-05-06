package port

// IDGenerator produces unique string ids. The use cases depend on this port
// rather than reaching into a uuid library directly, so tests can substitute
// a deterministic generator.
type IDGenerator interface {
	New() (string, error)
}
