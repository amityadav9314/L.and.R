package search

// Registry holds all registered search providers
type Registry struct {
	providers []SearchProvider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: []SearchProvider{},
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(provider SearchProvider) {
	r.providers = append(r.providers, provider)
}

// GetAll returns all registered providers
func (r *Registry) GetAll() []SearchProvider {
	return r.providers
}

// Count returns the number of registered providers
func (r *Registry) Count() int {
	return len(r.providers)
}
