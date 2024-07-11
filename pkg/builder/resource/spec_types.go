package resource

// SpecResource defines required methods for implementing a resource with spec.
type SpecResource interface {
	// CopyTo copies the content of the spec to a parent resource.
	CopyTo(parent ObjectWithSpec)
}
