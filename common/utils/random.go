package utils

import (
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Random struct {
	ctx      *pulumi.Context
	namer    namer.Namer
	provider *random.Provider
}

func WithProvider(provider *random.Provider) func(*Random) {
	return func(r *Random) {
		r.provider = provider
	}
}

func NewRandom(ctx *pulumi.Context, options ...func(*Random)) (*Random, error) {
	var err error

	rand := &Random{
		ctx:   ctx,
		namer: namer.NewNamer(ctx, "random"),
	}
	for _, opt := range options {
		opt(rand)
	}

	if rand.provider == nil {
		rand.provider, err = random.NewProvider(ctx, rand.namer.ResourceName("provider"), &random.ProviderArgs{})
		if err != nil {
			return nil, err
		}
	}

	return rand, nil
}

func (r *Random) RandomString(name string, length int, special bool) (*random.RandomString, error) {
	return random.NewRandomString(r.ctx, r.namer.ResourceName("random-string", name), &random.RandomStringArgs{
		Length:  pulumi.Int(length),
		Special: pulumi.Bool(special),
	})
}
