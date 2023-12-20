package fakeintake

import "github.com/DataDog/test-infra-definitions/common"

type Params struct {
	LoadBalancerEnabled bool
	ImageURL            string
}

type Option = func(*Params) error

// NewParams returns a new instance of Fakeintake Params
func NewParams(options ...Option) (*Params, error) {
	params := &Params{
		LoadBalancerEnabled: false,
		ImageURL:            "public.ecr.aws/datadog/fakeintake:latest",
	}
	return common.ApplyOption(params, options)
}

// WithLoadBalancer enable load balancer in front of the fakeintake
func WithLoadBalancer() Option {
	return func(p *Params) error {
		p.LoadBalancerEnabled = true
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
