package fakeintake

import "github.com/DataDog/test-infra-definitions/common"

type Params struct {
	DDDevForwarding bool
	ImageURL        string
	Memory          int
}

type Option = func(*Params) error

// NewParams returns a new instance of Fakeintake Params
func NewParams(options ...Option) (*Params, error) {
	params := &Params{
		ImageURL:        "gcr.io/datadoghq/fakeintake:latest",
		DDDevForwarding: true,
		Memory:          1024,
	}
	return common.ApplyOption(params, options)
}

// WithImageURL sets the URL of the image to use to define the fakeintake
func WithImageURL(imageURL string) Option {
	return func(p *Params) error {
		p.ImageURL = imageURL
		return nil
	}
}

// WithoutDDDevForwarding sets the flag to disable DD Dev forwarding
func WithoutDDDevForwarding() Option {
	return func(p *Params) error {
		p.DDDevForwarding = false
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
