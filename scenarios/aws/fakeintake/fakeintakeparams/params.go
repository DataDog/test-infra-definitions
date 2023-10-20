package fakeintakeparams

type Params struct {
	LoadBalancerEnabled bool
}

type Option = func(*Params) error

// NewParams returns a new instance of Fakeintake Params
func NewParams(options ...Option) *Params {
	return &Params{
		LoadBalancerEnabled: true,
	}
}

func WithoutLoadBalancer() Option {
	return func(p *Params) error {
		p.LoadBalancerEnabled = false
		return nil
	}
}
