package server

// ProvidePublicMethods exposes the unexported providePublicMethods function to
// external (_test) packages so tests can assert which RPCs are on the allowlist
// without relying on a full fx graph.
func ProvidePublicMethods() []string {
	return providePublicMethods()
}
