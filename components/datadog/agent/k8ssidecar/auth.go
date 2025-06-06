package k8ssidecar

import (
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	rbacv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/rbac/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	datadogSecretName                  = "datadog-secret"
	serviceAccountName                 = "datadog"
	sidecarAgentClusterRoleName        = "datadog-sidecar"
	sidecarAgentClusterRoleBindingName = sidecarAgentClusterRoleName
)

// NewServiceAccount creates a ServiceAccount
func NewServiceAccount(ctx *pulumi.Context, namespace string, name string, opts ...pulumi.ResourceOption) (*corev1.ServiceAccount, error) {
	sa, err := corev1.NewServiceAccount(ctx, "datadog", &corev1.ServiceAccountArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(name),
			Namespace: pulumi.String(namespace),
		},
	}, opts...)
	if err != nil {
		return nil, err
	}
	return sa, nil
}

// NewDatadogSecret creates a Secret with two fields
// - api-key
// - token
func NewDatadogSecret(ctx *pulumi.Context, namespace string, name string, apiKey pulumi.StringInput,
	token pulumi.StringInput, opts ...pulumi.ResourceOption) (*corev1.Secret, error) {
	s, err := corev1.NewSecret(ctx,
		name,
		&corev1.SecretArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String(name),
				Namespace: pulumi.String(namespace),
			},
			StringData: pulumi.StringMap{
				"api-key": apiKey,
				"token":   token,
			},
		}, opts...)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// NewAgentClusterRole creates a cluster role for a sidecar agent
func NewAgentClusterRole(ctx *pulumi.Context, name string, opts ...pulumi.ResourceOption) (*rbacv1.ClusterRole, error) {
	cr, err := rbacv1.NewClusterRole(ctx, name, &rbacv1.ClusterRoleArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String(name),
		},
		Rules: rbacv1.PolicyRuleArray{
			rbacv1.PolicyRuleArgs{
				ApiGroups: pulumi.StringArray{
					pulumi.String(""),
				},
				Resources: pulumi.StringArray{
					pulumi.String("nodes"),
					pulumi.String("namespaces"),
					pulumi.String("endpoints"),
				},
				Verbs: pulumi.StringArray{
					pulumi.String("get"),
					pulumi.String("list"),
				},
			},
			rbacv1.PolicyRuleArgs{
				ApiGroups: pulumi.StringArray{
					pulumi.String(""),
				},
				Resources: pulumi.StringArray{
					pulumi.String("nodes/metrics"),
					pulumi.String("nodes/metrics"),
					pulumi.String("nodes/spec"),
					pulumi.String("nodes/stats"),
					pulumi.String("nodes/proxy"),
					pulumi.String("nodes/pods"),
					pulumi.String("nodes/healthz"),
				},
				Verbs: pulumi.StringArray{
					pulumi.String("get"),
				},
			},
		},
	}, opts...)
	if err != nil {
		return nil, err
	}

	return cr, nil
}

// NewClusterRoleBinding creates a cluster role binding
func NewClusterRoleBinding(ctx *pulumi.Context, name string, clusterRole *rbacv1.ClusterRole,
	serviceAccount *corev1.ServiceAccount, opts ...pulumi.ResourceOption) (*rbacv1.ClusterRoleBinding, error) {
	crb, err := rbacv1.NewClusterRoleBinding(ctx, name, &rbacv1.ClusterRoleBindingArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String(name),
		},
		RoleRef: &rbacv1.RoleRefArgs{
			ApiGroup: pulumi.String("rbac.authorization.k8s.io"),
			Kind:     clusterRole.Kind,
			Name:     clusterRole.Metadata.Name().Elem(),
		},
		Subjects: rbacv1.SubjectArray{
			&rbacv1.SubjectArgs{
				Kind:      serviceAccount.Kind,
				Name:      serviceAccount.Metadata.Name().Elem(),
				Namespace: serviceAccount.Metadata.Namespace().Elem(),
			},
		},
	}, opts...)

	if err != nil {
		return nil, err
	}

	return crb, nil
}

// NewServiceAccountWithClusterPermissions creates a cluster role with default permissions, and returns a service account
// with those permissions attached
func NewServiceAccountWithClusterPermissions(ctx *pulumi.Context, namespace string, apiKey pulumi.StringInput,
	clusterAgentToken pulumi.StringInput, opts ...pulumi.ResourceOption) (*corev1.ServiceAccount, error) {
	_, err := NewDatadogSecret(ctx, namespace, datadogSecretName, apiKey, clusterAgentToken)
	if err != nil {
		return nil, err
	}

	serviceAccount, err := NewServiceAccount(ctx, namespace, serviceAccountName)
	if err != nil {
		return nil, err
	}

	clusterRole, err := NewAgentClusterRole(ctx, sidecarAgentClusterRoleName, opts...)
	if err != nil {
		return nil, err
	}

	_, err = NewClusterRoleBinding(ctx, sidecarAgentClusterRoleBindingName, clusterRole, serviceAccount, opts...)
	if err != nil {
		return nil, err
	}

	return serviceAccount, nil
}
