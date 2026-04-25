package suggest

type Registry struct {
	engines map[string]Engine
	names   []string
}

func NewRegistry(items []Item, rankedCacheK int) *Registry {
	engines := []Engine{
		NewLinear(items),
		NewSorted(items),
		NewRadix(items),
		NewHashicorpRadix(items),
		NewRankedTrie(items, rankedCacheK),
	}

	registry := &Registry{
		engines: make(map[string]Engine, len(engines)),
		names:   make([]string, 0, len(engines)),
	}
	for _, engine := range engines {
		registry.engines[engine.Name()] = engine
		registry.names = append(registry.names, engine.Name())
	}
	return registry
}

func (r *Registry) Get(name string) (Engine, bool) {
	if name == "" {
		name = "ranked-trie"
	}
	engine, ok := r.engines[name]
	return engine, ok
}

func (r *Registry) Names() []string {
	out := make([]string, len(r.names))
	copy(out, r.names)
	return out
}
