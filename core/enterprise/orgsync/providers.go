//go:build enterprise

package orgsync

import "fmt"

type Config map[string]string

type ProviderFactory func(Config) (ProviderClient, error)

type ProviderRegistry struct {
	factories map[string]ProviderFactory
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		factories: make(map[string]ProviderFactory),
	}
}

func (r *ProviderRegistry) Register(provider string, factory ProviderFactory) {
	r.factories[provider] = factory
}

func (r *ProviderRegistry) New(provider string, config Config) (ProviderClient, error) {
	factory, ok := r.factories[provider]
	if !ok {
		return nil, fmt.Errorf("provider not registered: %s", provider)
	}

	return factory(config)
}
