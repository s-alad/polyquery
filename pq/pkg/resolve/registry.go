package resolve

type Registry struct {
	resolvers []Resolver
}

func NewRegistry(rs ...Resolver) *Registry {
	return &Registry{resolvers: rs}
}

func (r *Registry) Match(m Market) (Resolver, bool) {
	for _, res := range r.resolvers {
		if res.CanResolve(m) {
			return res, true
		}
	}
	return nil, false
}
