package main

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakediscovery "k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

var (
	crdGVR = schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	crdGVK = crdGVR.GroupVersion().WithKind("CustomResourceDefinition")
	// widgetGVR is the resource of a custom resource whose definition chose a plural that can't
	// be guessed from its kind.
	widgetGVR = schema.GroupVersionResource{Group: "churn.datadoghq.com", Version: "v1", Resource: "widgetz"}
	widgetGVK = widgetGVR.GroupVersion().WithKind("Widget")

	crdResources = &metav1.APIResourceList{
		GroupVersion: crdGVR.GroupVersion().String(),
		APIResources: []metav1.APIResource{{Name: crdGVR.Resource, Kind: crdGVK.Kind}},
	}
	widgetResources = &metav1.APIResourceList{
		GroupVersion: widgetGVR.GroupVersion().String(),
		APIResources: []metav1.APIResource{{Name: widgetGVR.Resource, Kind: widgetGVK.Kind, Namespaced: true}},
	}
)

// newTestResources returns the resources of a fake API server serving the resources of served,
// through the same discovery cache as churn.
func newTestResources(served ...*metav1.APIResourceList) (resources, *dynamicfake.FakeDynamicClient, *fakediscovery.FakeDiscovery) {
	client := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	discoveryClient := &fakediscovery.FakeDiscovery{Fake: &k8stesting.Fake{Resources: served}}
	return newResources(client, discoveryClient), client, discoveryClient
}

// testContext returns a context that expires long after createObject should have returned, so
// that a test fails rather than hangs when createObject keeps retrying. Within a synctest
// bubble, whose clock jumps ahead whenever all its goroutines are blocked, that takes no time.
func testContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)
	return ctx
}

func newObject(gvk schema.GroupVersionKind, namespace, name string) *unstructured.Unstructured {
	o := &unstructured.Unstructured{}
	o.SetGroupVersionKind(gvk)
	o.SetNamespace(namespace)
	o.SetName(name)
	return o
}

func TestCreateObjectWaitsForItsDefinition(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		res, client, discoveryClient := newTestResources(crdResources)
		// The definition gets established after churn first looked for its kind. The reactor
		// runs in the goroutine querying the discovery, before it reads the served resources.
		established := time.Now().Add(retryInterval / 2)
		discoveryClient.PrependReactor("get", "*", func(k8stesting.Action) (bool, runtime.Object, error) {
			if time.Now().After(established) && !slices.Contains(discoveryClient.Resources, widgetResources) {
				discoveryClient.Resources = append(discoveryClient.Resources, widgetResources)
			}
			return false, nil, nil
		})

		if err := createObject(testContext(t), res, newObject(widgetGVK, "default", "widget-0")); err != nil {
			t.Fatalf("createObject: %v", err)
		}
		if _, err := client.Resource(widgetGVR).Namespace("default").Get(t.Context(), "widget-0", metav1.GetOptions{}); err != nil {
			t.Errorf("the custom resource wasn't created with the resource of its definition: %v", err)
		}
		// The whole API server is only discovered again once the definition is established.
		discoveries := 0
		for _, action := range discoveryClient.Actions() {
			if action.GetResource().Resource == "group" {
				discoveries++
			}
		}
		if discoveries != 2 {
			t.Errorf("the API server was discovered %d times, want 2", discoveries)
		}
	})
}

func TestCreateObjectFollowsAReplacedDefinition(t *testing.T) {
	for name, old := range map[string]metav1.APIResource{
		// A leftover definition of the same name, hence the same plural, may have another scope.
		"scope": {Name: widgetGVR.Resource, Kind: widgetGVK.Kind},
		// A definition replaced by someone else may have another plural.
		"plural": {Name: "widgets", Kind: widgetGVK.Kind, Namespaced: true},
	} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				oldResources := &metav1.APIResourceList{GroupVersion: widgetResources.GroupVersion, APIResources: []metav1.APIResource{old}}
				res, client, discoveryClient := newTestResources(crdResources, oldResources)
				widget := newObject(widgetGVK, "default", "widget-0")
				if _, err := res.of(widget); err != nil {
					t.Fatalf("of: %v", err)
				}

				// It was replaced since the discovery was cached, and its old resource is gone.
				discoveryClient.Resources = []*metav1.APIResourceList{crdResources, widgetResources}
				client.PrependReactor("create", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
					if action.GetResource() != widgetGVR || action.GetNamespace() != widget.GetNamespace() {
						return true, nil, apierrors.NewNotFound(action.GetResource().GroupResource(), "")
					}
					return false, nil, nil
				})

				if err := createObject(testContext(t), res, widget); err != nil {
					t.Fatalf("createObject: %v", err)
				}
				if _, err := client.Resource(widgetGVR).Namespace("default").Get(t.Context(), "widget-0", metav1.GetOptions{}); err != nil {
					t.Errorf("the custom resource wasn't created with the resource of its new definition: %v", err)
				}
			})
		})
	}
}

func TestCreateObjectRetriesWhileItsDefinitionIsDeleted(t *testing.T) {
	const reason = "create not allowed while custom resource definition is terminating"
	widgets := widgetGVR.GroupResource()
	methodNotSupported := apierrors.NewMethodNotSupported(widgets, "create")
	methodNotSupported.ErrStatus.Message = reason

	for name, terminating := range map[string]error{
		"Kubernetes 1.34 and later": apierrors.NewForbidden(widgets, "widget-0", errors.New(reason)),
		"Kubernetes 1.33 and older": methodNotSupported,
	} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				res, client, _ := newTestResources(widgetResources)
				attempts := 0
				client.PrependReactor("create", widgetGVR.Resource, func(k8stesting.Action) (bool, runtime.Object, error) {
					attempts++
					if attempts == 1 {
						return true, nil, terminating
					}
					return false, nil, nil
				})

				if err := createObject(testContext(t), res, newObject(widgetGVK, "default", "widget-0")); err != nil {
					t.Fatalf("createObject: %v", err)
				}
				if attempts != 2 {
					t.Errorf("createObject attempted %d creations, want 2", attempts)
				}
			})
		})
	}
}

func TestCreateObjectFailsWhenCreationIsNotSupported(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		res, client, _ := newTestResources(widgetResources)
		client.PrependReactor("create", widgetGVR.Resource, func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewMethodNotSupported(widgetGVR.GroupResource(), "create")
		})

		if err := createObject(testContext(t), res, newObject(widgetGVK, "default", "widget-0")); !apierrors.IsMethodNotSupported(err) {
			t.Errorf("createObject returned %v, want a MethodNotSupported error", err)
		}
	})
}

func TestDeleteObjectOfUnservedKind(t *testing.T) {
	res, client, _ := newTestResources(crdResources)

	if err := deleteObject(t.Context(), res, newObject(widgetGVK, "default", "widget-0"), metav1.DeleteOptions{}); err != nil {
		t.Errorf("deleteObject: %v", err)
	}
	if actions := client.Actions(); len(actions) != 0 {
		t.Errorf("deleteObject called the API server: %v", actions)
	}
}

func TestHandleOneObjectRejectsInvalidAnnotations(t *testing.T) {
	for name, annotations := range map[string]map[string]string{
		"no instances":       {"churn.datadoghq.com/instances": "0", "churn.datadoghq.com/lifetime": "1m"},
		"negative instances": {"churn.datadoghq.com/instances": "-1", "churn.datadoghq.com/lifetime": "1m"},
		"negative lifetime":  {"churn.datadoghq.com/lifetime": "-1m"},
		"lifetime too short": {"churn.datadoghq.com/instances": "5", "churn.datadoghq.com/lifetime": "4ns"},
	} {
		t.Run(name, func(t *testing.T) {
			res, client, _ := newTestResources(widgetResources)
			o := newObject(widgetGVK, "default", "widget-0")
			o.SetAnnotations(annotations)

			if err := handleOneObject(t.Context(), res, o); err == nil {
				t.Error("handleOneObject accepted invalid annotations")
			}
			if actions := client.Actions(); len(actions) != 0 {
				t.Errorf("handleOneObject called the API server: %v", actions)
			}
		})
	}
}

func TestClusterScopedObjectsIgnoreTheirNamespace(t *testing.T) {
	res, client, _ := newTestResources(crdResources)
	crd := newObject(crdGVK, "", "widgetz.churn.datadoghq.com")
	if _, err := client.Resource(crdGVR).Create(t.Context(), crd, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	// Like the API server, churn ignores the namespace of a cluster-scoped object.
	resource, err := res.of(newObject(crdGVK, "default", crd.GetName()))
	if err != nil {
		t.Fatalf("of: %v", err)
	}
	if _, err := resource.Get(t.Context(), crd.GetName(), metav1.GetOptions{}); err != nil {
		t.Errorf("the resource isn't cluster-wide: %v", err)
	}
}
