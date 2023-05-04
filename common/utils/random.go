package utils

import (
	"fmt"

	"github.com/DataDog/test-infra-definitions/common/config"
	"github.com/DataDog/test-infra-definitions/common/namer"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type RandomGenerator struct {
	e     config.CommonEnvironment
	namer namer.Namer
}

func NewRandomGenerator(e config.CommonEnvironment, name string, options ...func(*RandomGenerator)) *RandomGenerator {
	rand := &RandomGenerator{
		e:     e,
		namer: namer.NewNamer(e.Ctx, "random-"+name),
	}
	for _, opt := range options {
		opt(rand)
	}

	return rand
}

func (r *RandomGenerator) RandomString(name string, length int, special bool) (*random.RandomString, error) {
	provider, err := r.e.RandomProvider()
	if err != nil {
		panic(fmt.Sprintf("failed to get random provider %s", err))
	}
	return random.NewRandomString(r.e.Ctx, r.namer.ResourceName("random-string", name), &random.RandomStringArgs{
		Length:  pulumi.Int(length),
		Special: pulumi.Bool(special),
	}, pulumi.Provider(provider))
}
