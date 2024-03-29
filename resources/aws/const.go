package aws

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

var (
	EnabledString  = pulumi.String("ENABLED")
	DisabledString = pulumi.String("DISABLED")
	AgentQAECR     = "669783387624.dkr.ecr.us-east-1.amazonaws.com"
)
