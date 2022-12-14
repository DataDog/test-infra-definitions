package vm

import (
	"github.com/DataDog/test-infra-definitions/aws/ec2/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Run(ctx *pulumi.Context) error {
	_, err := ec2.NewVM(ctx)
	if err != nil {
		return err
	}

	return nil
}
