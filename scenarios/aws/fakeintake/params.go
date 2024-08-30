package fakeintake

import "github.com/DataDog/test-infra-definitions/common"

type Params struct {
	LoadBalancerEnabled bool
	ImageURL            string
	CPU                 int
	Memory              int
	DDDevForwarding     bool
}

type Option = func(*Params) error

// NewParams returns a new instance of Fakeintake Params
func NewParams(options ...Option) (*Params, error) {
	params := &Params{
		LoadBalancerEnabled: false,
		ImageURL:            "public.ecr.aws/datadog/fakeintake:latest",
		CPU:                 512,
		Memory:              1024,
		DDDevForwarding:     true,
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

// WithCPU sets the number of CPU units to allocate to the fakeintake
func WithCPU(cpu int) Option {
	return func(p *Params) error {
		p.CPU = cpu
		return nil
	}
}

// WithMemory sets the amount (in MiB) of memory to allocate to the fakeintake
func WithMemory(memory int) Option {
	return func(p *Params) error {
		p.Memory = memory
		return nil
	}
}

func WithDDDevForwarding() Option {
	return func(p *Params) error {
		p.DDDevForwarding = true
		return nil
	}
}
