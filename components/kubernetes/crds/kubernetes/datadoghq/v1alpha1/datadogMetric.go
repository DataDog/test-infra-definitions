// Code generated by crd2pulumi DO NOT EDIT.
// *** WARNING: Do not edit by hand unless you're certain you know what you are doing! ***

package v1alpha1

import (
	"context"
	"reflect"

	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/utilities"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// DatadogMetric allows autoscaling on arbitrary Datadog query
type DatadogMetric struct {
	pulumi.CustomResourceState

	ApiVersion pulumi.StringPtrOutput     `pulumi:"apiVersion"`
	Kind       pulumi.StringPtrOutput     `pulumi:"kind"`
	Metadata   metav1.ObjectMetaPtrOutput `pulumi:"metadata"`
	// DatadogMetricSpec defines the desired state of DatadogMetric
	Spec DatadogMetricSpecPtrOutput `pulumi:"spec"`
	// DatadogMetricStatus defines the observed state of DatadogMetric
	Status DatadogMetricStatusPtrOutput `pulumi:"status"`
}

// NewDatadogMetric registers a new resource with the given unique name, arguments, and options.
func NewDatadogMetric(ctx *pulumi.Context,
	name string, args *DatadogMetricArgs, opts ...pulumi.ResourceOption) (*DatadogMetric, error) {
	if args == nil {
		args = &DatadogMetricArgs{}
	}

	args.ApiVersion = pulumi.StringPtr("datadoghq.com/v1alpha1")
	args.Kind = pulumi.StringPtr("DatadogMetric")
	opts = utilities.PkgResourceDefaultOpts(opts)
	var resource DatadogMetric
	err := ctx.RegisterResource("kubernetes:datadoghq.com/v1alpha1:DatadogMetric", name, args, &resource, opts...)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

// GetDatadogMetric gets an existing DatadogMetric resource's state with the given name, ID, and optional
// state properties that are used to uniquely qualify the lookup (nil if not required).
func GetDatadogMetric(ctx *pulumi.Context,
	name string, id pulumi.IDInput, state *DatadogMetricState, opts ...pulumi.ResourceOption) (*DatadogMetric, error) {
	var resource DatadogMetric
	err := ctx.ReadResource("kubernetes:datadoghq.com/v1alpha1:DatadogMetric", name, id, state, &resource, opts...)
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

// Input properties used for looking up and filtering DatadogMetric resources.
type datadogMetricState struct {
}

type DatadogMetricState struct {
}

func (DatadogMetricState) ElementType() reflect.Type {
	return reflect.TypeOf((*datadogMetricState)(nil)).Elem()
}

type datadogMetricArgs struct {
	ApiVersion *string            `pulumi:"apiVersion"`
	Kind       *string            `pulumi:"kind"`
	Metadata   *metav1.ObjectMeta `pulumi:"metadata"`
	// DatadogMetricSpec defines the desired state of DatadogMetric
	Spec *DatadogMetricSpec `pulumi:"spec"`
	// DatadogMetricStatus defines the observed state of DatadogMetric
	Status *DatadogMetricStatus `pulumi:"status"`
}

// The set of arguments for constructing a DatadogMetric resource.
type DatadogMetricArgs struct {
	ApiVersion pulumi.StringPtrInput
	Kind       pulumi.StringPtrInput
	Metadata   metav1.ObjectMetaPtrInput
	// DatadogMetricSpec defines the desired state of DatadogMetric
	Spec DatadogMetricSpecPtrInput
	// DatadogMetricStatus defines the observed state of DatadogMetric
	Status DatadogMetricStatusPtrInput
}

func (DatadogMetricArgs) ElementType() reflect.Type {
	return reflect.TypeOf((*datadogMetricArgs)(nil)).Elem()
}

type DatadogMetricInput interface {
	pulumi.Input

	ToDatadogMetricOutput() DatadogMetricOutput
	ToDatadogMetricOutputWithContext(ctx context.Context) DatadogMetricOutput
}

func (*DatadogMetric) ElementType() reflect.Type {
	return reflect.TypeOf((**DatadogMetric)(nil)).Elem()
}

func (i *DatadogMetric) ToDatadogMetricOutput() DatadogMetricOutput {
	return i.ToDatadogMetricOutputWithContext(context.Background())
}

func (i *DatadogMetric) ToDatadogMetricOutputWithContext(ctx context.Context) DatadogMetricOutput {
	return pulumi.ToOutputWithContext(ctx, i).(DatadogMetricOutput)
}

type DatadogMetricOutput struct{ *pulumi.OutputState }

func (DatadogMetricOutput) ElementType() reflect.Type {
	return reflect.TypeOf((**DatadogMetric)(nil)).Elem()
}

func (o DatadogMetricOutput) ToDatadogMetricOutput() DatadogMetricOutput {
	return o
}

func (o DatadogMetricOutput) ToDatadogMetricOutputWithContext(ctx context.Context) DatadogMetricOutput {
	return o
}

func (o DatadogMetricOutput) ApiVersion() pulumi.StringPtrOutput {
	return o.ApplyT(func(v *DatadogMetric) pulumi.StringPtrOutput { return v.ApiVersion }).(pulumi.StringPtrOutput)
}

func (o DatadogMetricOutput) Kind() pulumi.StringPtrOutput {
	return o.ApplyT(func(v *DatadogMetric) pulumi.StringPtrOutput { return v.Kind }).(pulumi.StringPtrOutput)
}

func (o DatadogMetricOutput) Metadata() metav1.ObjectMetaPtrOutput {
	return o.ApplyT(func(v *DatadogMetric) metav1.ObjectMetaPtrOutput { return v.Metadata }).(metav1.ObjectMetaPtrOutput)
}

// DatadogMetricSpec defines the desired state of DatadogMetric
func (o DatadogMetricOutput) Spec() DatadogMetricSpecPtrOutput {
	return o.ApplyT(func(v *DatadogMetric) DatadogMetricSpecPtrOutput { return v.Spec }).(DatadogMetricSpecPtrOutput)
}

// DatadogMetricStatus defines the observed state of DatadogMetric
func (o DatadogMetricOutput) Status() DatadogMetricStatusPtrOutput {
	return o.ApplyT(func(v *DatadogMetric) DatadogMetricStatusPtrOutput { return v.Status }).(DatadogMetricStatusPtrOutput)
}

func init() {
	pulumi.RegisterInputType(reflect.TypeOf((*DatadogMetricInput)(nil)).Elem(), &DatadogMetric{})
	pulumi.RegisterOutputType(DatadogMetricOutput{})
}
