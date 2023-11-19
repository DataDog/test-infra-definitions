package fakeintakeparams

import "github.com/DataDog/test-infra-definitions/common"

type Params struct {
	LoadBalancerEnabled bool
	Name                string
	ImageURL            string
}

type Option = func(*Params) error

// NewParams returns a new instance of Fakeintake Params
func NewParams(options ...Option) (*Params, error) {
	params := &Params{
		LoadBalancerEnabled: true,
		Name:                "fakeintake",
		ImageURL:            "public.ecr.aws/datadog/fakeintake:latest",
	}
	return common.ApplyOption(params, options)
}

// WithoutLoadBalancer disable load balancer in front of the fakeintake
func WithoutLoadBalancer() Option {
	return func(p *Params) error {
		p.LoadBalancerEnabled = false
		return nil
	}
}

// WithName sets the name of the fakeintake.
// Only useful when using several fakeintakes in the same test.
func WithName(name string) Option {
	return func(p *Params) error {
		p.Name = name
		return nil
	}
}

// WithImageURL sets the URL of the image to use to define the fakeintake
func WithImageURL(imageURL string) Option {
	return func(p *Params) error {
		p.ImageURL = imageURL
		return nil
	}
}
