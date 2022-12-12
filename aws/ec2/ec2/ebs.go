package ec2

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/mitchellh/mapstructure"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type BlockDeviceArgs struct {
	// Whether the volume should be destroyed on instance termination. Defaults to `true`.
	DeleteOnTermination pulumi.BoolPtrInput `mapstructure:"delete_on_termination"`
	// Name of the device to mount.
	DeviceName pulumi.StringInput `mapstructure:"device_name"`
	// Enables [EBS encryption](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/EBSEncryption.html) on the volume. Defaults to `false`. Cannot be used with `snapshotId`. Must be configured to perform drift detection.
	Encrypted pulumi.BoolPtrInput `mapstructure:"encrypted"`
	// Amount of provisioned [IOPS](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-io-characteristics.html). Only valid for volumeType of `io1`, `io2` or `gp3`.
	Iops pulumi.IntPtrInput `mapstructure:"iops"`
	// Amazon Resource Name (ARN) of the KMS Key to use when encrypting the volume. Must be configured to perform drift detection.
	KmsKeyId pulumi.StringPtrInput `mapstructure:"kmsKeyId"`
	// Snapshot ID to mount.
	SnapshotId pulumi.StringPtrInput `mapstructure:"snapshot_id"`
	// Map of tags to assign to the device.
	Tags pulumi.StringMapInput `mapstructure:"tags"`
	// Throughput to provision for a volume in mebibytes per second (MiB/s). This is only valid for `volumeType` of `gp3`.
	Throughput pulumi.IntPtrInput `mapstructure:"throughput"`
	// ID of the volume. For example, the ID can be accessed like this, `aws_instance.web.root_block_device.0.volume_id`.
	VolumeId pulumi.StringPtrInput `mapstructure:"volume_id"`
	// Size of the volume in gibibytes (GiB).
	VolumeSize pulumi.IntPtrInput `mapstructure:"volume_size"`
	// Type of volume. Valid values include `standard`, `gp2`, `gp3`, `io1`, `io2`, `sc1`, or `st1`. Defaults to `gp2`.
	VolumeType pulumi.StringPtrInput `mapstructure:"volume_type"`
}

func decodeHook(f reflect.Kind, t reflect.Kind, data interface{}) (interface{}, error) {
	if f == reflect.Map {
		return data, nil
	}
	if f == reflect.Bool {
		return pulumi.Bool(data.(bool)), nil
	}
	if f == reflect.Int {
		return pulumi.Int(data.(int)), nil
	}
	if f == reflect.String {
		dataStr := data.(string)
		intFromStr, err := strconv.Atoi(dataStr)
		if err == nil {
			return pulumi.Int(intFromStr), nil
		}
		boolFromStr, err := strconv.ParseBool(dataStr)
		if err == nil {
			return pulumi.Bool(boolFromStr), nil
		}
		return pulumi.String(data.(string)), nil
	}

	return nil, fmt.Errorf("unhandled type: %v", f)
}

func createNewRootDeviceMapping(root ec2.GetAmiBlockDeviceMapping, newSize int) (ec2.InstanceEbsBlockDeviceArgs, error) {
	var arg BlockDeviceArgs

	dc := &mapstructure.DecoderConfig{Result: &arg, DecodeHook: decodeHook}
	ms, err := mapstructure.NewDecoder(dc)
	if err != nil {
		return ec2.InstanceEbsBlockDeviceArgs{}, err
	}
	if err := ms.Decode(root.Ebs); err != nil {
		return ec2.InstanceEbsBlockDeviceArgs{}, err
	}

	result := ec2.InstanceEbsBlockDeviceArgs(arg)
	result.DeviceName = pulumi.String(root.DeviceName)
	result.VolumeSize = pulumi.Int(newSize)

	return result, nil
}
