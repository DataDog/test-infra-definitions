package fakeintake

import "github.com/DataDog/test-infra-definitions/common"

type Params struct {
	ImageURL        string
	DDDevForwarding bool
}

type Option = func(*Params) error

// NewParams returns a new instance of Fakeintake Params
func NewParams(options ...Option) (*Params, error) {
	params := &Params{
		ImageURL: "public.ecr.aws/datadog/fakeintake:latest",
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

// WithDDDevForwarding sets the flag to enable DD Dev forwarding
func WithDDDevForwarding() Option {
	return func(p *Params) error {
		p.DDDevForwarding = true
		return nil
	}
}
