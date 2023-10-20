package fakeintakeparams

import "github.com/DataDog/test-infra-definitions/common"

type Params struct {
	LoadBalancerEnabled bool
}

type Option = func(*Params) error

// NewParams returns a new instance of Fakeintake Params
func NewParams(options ...Option) (*Params, error) {
	params := &Params{
		LoadBalancerEnabled: true,
	}
	return common.ApplyOption(params, options)
}

func WithoutLoadBalancer() Option {
	return func(p *Params) error {
		p.LoadBalancerEnabled = false
		return nil
	}
}
