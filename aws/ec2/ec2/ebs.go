package ec2

import (
	"strconv"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func createNewRootDeviceMapping(root ec2.GetAmiBlockDeviceMapping, newSize int) ec2.InstanceEbsBlockDeviceArgs {
	var arg ec2.InstanceEbsBlockDeviceArgs

	if val, ok := root.Ebs["delete_on_termination"]; ok {
		if val == "true" {
			arg.DeleteOnTermination = pulumi.Bool(true)
		}
		arg.DeleteOnTermination = pulumi.Bool(false)
	}

	if val, ok := root.Ebs["encrypted"]; ok {
		if val == "true" {
			arg.Encrypted = pulumi.Bool(true)
		}

		arg.Encrypted = pulumi.Bool(false)
	}

	if val, ok := root.Ebs["iops"]; ok {
		iops, err := strconv.Atoi(val)
		if err == nil {
			arg.Iops = pulumi.Int(iops)
		}
	}

	if val, ok := root.Ebs["snapshot_id"]; ok {
		arg.SnapshotId = pulumi.String(val)
	}

	if val, ok := root.Ebs["throughput"]; ok {
		throughput, err := strconv.Atoi(val)
		if err == nil {
			arg.Throughput = pulumi.Int(throughput)
		}
	}

	if val, ok := root.Ebs["volume_type"]; ok {
		arg.VolumeType = pulumi.String(val)
	}

	if val, ok := root.Ebs["volume_id"]; ok {
		arg.VolumeId = pulumi.String(val)
	}

	arg.DeviceName = pulumi.String(root.DeviceName)
	arg.VolumeSize = pulumi.Int(newSize)
	return arg
}
