package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
)

const (
	// retryInterval is how long createObject waits between two attempts.
	retryInterval = 1 * time.Second
	// discoveryTimeout bounds each discovery query, which takes no context, below the 32s
	// default of client-go: a discovery takes two queries, which must fit well within the
	// 1-minute cleanup of a handler.
	discoveryTimeout = 10 * time.Second
)

// resources gives access to the API resources of objects, found from their kinds.
type resources struct {
	client    dynamic.Interface
	discovery discovery.DiscoveryInterface
	// mapper maps kinds to resources from a cached discovery of the API server.
	mapper meta.ResettableRESTMapper
}

func newResources(client dynamic.Interface, discoveryClient discovery.DiscoveryInterface) resources {
	return resources{
		client:    client,
		discovery: discoveryClient,
		mapper:    restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient)),
	}
}

// of returns the client of the resource of o: cluster-wide for a cluster-scoped kind, in the
// namespace of o otherwise. The error satisfies meta.IsNoMatchError when the kind of o isn't
// in the discovery cache.
func (r resources) of(o *unstructured.Unstructured) (dynamic.ResourceInterface, error) {
	gvk := o.GroupVersionKind()
	mapping, err := r.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}
	if mapping.Scope.Name() == meta.RESTScopeNameRoot {
		return r.client.Resource(mapping.Resource), nil
	}
	return r.client.Resource(mapping.Resource).Namespace(o.GetNamespace()), nil
}

// refresh resets the cached discovery when it doesn't map the kind of o to the resource and
// scope the API server serves now, as when the custom resource definition of o gets established
// or is replaced. It first queries the group version of o alone: one small query that doesn't
// hold up the mappings of the other handlers.
func (r resources) refresh(o *unstructured.Unstructured) {
	gvk := o.GroupVersionKind()
	list, err := r.discovery.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return
	}
	i := slices.IndexFunc(list.APIResources, func(resource metav1.APIResource) bool {
		return resource.Kind == gvk.Kind && !strings.Contains(resource.Name, "/") // not a subresource
	})
	if i < 0 {
		return
	}
	served := list.APIResources[i]
	mapping, err := r.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil || mapping.Resource.Resource != served.Name || (mapping.Scope.Name() == meta.RESTScopeNameNamespace) != served.Namespaced {
		r.mapper.Reset()
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dir := flag.String("manifests-dir", ".", "directory containing YAML manifests")
	qps := flag.Float64("qps", float64(rest.DefaultQPS), "maximum queries per second to the Kubernetes API server")
	burst := flag.Int("burst", rest.DefaultBurst, "maximum burst of queries to the Kubernetes API server")
	flag.Parse()
	// Unlike rest.Config.QPS and Burst, the rate limiter set below has no default for zero, and
	// refuses every query beyond its burst when its rate isn't positive.
	if *qps <= 0 || *burst <= 0 {
		log.Fatal("-qps and -burst must be positive")
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig, ok := os.LookupEnv("KUBECONFIG")
		if !ok {
			kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		log.Fatal(err)
	}
	// The dynamic and discovery clients share one rate limiter, so that -qps and -burst bound
	// all the queries of churn.
	config.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(float32(*qps), *burst)
	discoveryConfig := rest.CopyConfig(config)
	discoveryConfig.Timeout = discoveryTimeout
	res := newResources(dynamic.NewForConfigOrDie(config), discovery.NewDiscoveryClientForConfigOrDie(discoveryConfig))

	// runCtx is cancelled on a termination signal, or as soon as one object handler fails so
	// that all the others stop too. Either way, every handler then deletes the objects it
	// created, and main waits for them before exiting.
	runCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	var wg sync.WaitGroup

	if err := filepath.WalkDir(*dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !slices.Contains([]string{".yaml", ".yml"}, filepath.Ext(path)) {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		dec := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
		for {
			obj := &unstructured.Unstructured{}
			if err := dec.Decode(obj); err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
			// An empty document, or one with only comments, holds no object.
			if len(obj.Object) == 0 {
				continue
			}

			wg.Go(func() {
				if err := handleOneObject(runCtx, res, obj); err != nil {
					cancel(fmt.Errorf("%s %s/%s: %w", obj.GetKind(), obj.GetNamespace(), obj.GetName(), err))
				}
			})
		}
	}); err != nil {
		cancel(err)
	}

	<-runCtx.Done()
	wg.Wait()

	if err := context.Cause(runCtx); !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}

func handleOneObject(ctx context.Context, res resources, obj *unstructured.Unstructured) error {
	nbInstances := 1
	if nb, ok := obj.GetAnnotations()["churn.datadoghq.com/instances"]; ok {
		var err error
		nbInstances, err = strconv.Atoi(nb)
		if err != nil {
			return err
		}
		if nbInstances < 1 {
			return fmt.Errorf("churn.datadoghq.com/instances must be positive, got %d", nbInstances)
		}
	}

	var lifetime time.Duration
	if t, ok := obj.GetAnnotations()["churn.datadoghq.com/lifetime"]; ok {
		var err error
		lifetime, err = time.ParseDuration(t)
		if err != nil {
			return err
		}
	}
	// One instance is recreated every interval, which a ticker requires to be positive.
	interval := lifetime / time.Duration(nbInstances)
	if lifetime != 0 && interval <= 0 {
		return fmt.Errorf("churn.datadoghq.com/lifetime must be 0 or at least %s, got %s", time.Duration(nbInstances), lifetime)
	}

	instance := func(i int) *unstructured.Unstructured {
		return transformObject(obj, func(s string) string {
			return strings.ReplaceAll(s, "{{i}}", strconv.Itoa(i))
		})
	}

	// Delete all the instances when done, including after a partial creation or an error.
	// ctx is already cancelled by then, so the deletion uses its own context.
	defer func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 1*time.Minute)
		defer cancel()

		for i := range nbInstances {
			if err := deleteObject(ctx, res, instance(i), metav1.DeleteOptions{GracePeriodSeconds: new(int64(1))}); err != nil {
				// Past the timeout, the remaining deletions would fail the same way, each
				// maybe after a discovery query.
				if ctx.Err() != nil {
					log.Printf("Not deleting the %d remaining instances of %s %s/%s: %v", nbInstances-i, obj.GetKind(), obj.GetNamespace(), obj.GetName(), err)
					break
				}
				log.Print(err)
			}
		}
	}()

	// An object left over by a previous churn instance that could not clean up (e.g. it was
	// killed) is created again from the template, as everything churn manages is transient.
	for i := range nbInstances {
		if err := recreateObject(ctx, res, instance(i)); err != nil {
			return ignoreCancellation(ctx, err)
		}
	}

	if lifetime == 0 {
		<-ctx.Done()
		return nil
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := recreateObject(ctx, res, instance(i%nbInstances)); err != nil {
				return ignoreCancellation(ctx, err)
			}
		}
	}
}

// recreateObject deletes o if it exists and creates it again.
func recreateObject(ctx context.Context, res resources, o *unstructured.Unstructured) error {
	if err := deleteObject(ctx, res, o, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return createObject(ctx, res, o)
}

// createObject creates o. Creation is retried while a deleted object with the same name is not
// gone yet, while the namespace of o does not exist yet or is being deleted, and while the
// custom resource definition of o does not exist, is not established yet or is being deleted,
// since namespaces and custom resource definitions are created and deleted by other handlers
// running concurrently.
func createObject(ctx context.Context, res resources, o *unstructured.Unstructured) error {
	logged := false
	for {
		resource, err := res.of(o)
		if err == nil {
			_, err = resource.Create(ctx, o, metav1.CreateOptions{})
		}
		switch {
		case err == nil:
			log.Printf("Created %s %s/%s", o.GetKind(), o.GetNamespace(), o.GetName())
			return nil
		case meta.IsNoMatchError(err), apierrors.IsNotFound(err):
			// The kind of o may not be served yet, or no longer as cached.
			res.refresh(o)
		case apierrors.IsAlreadyExists(err), apierrors.HasStatusCause(err, "NamespaceTerminating"), isDefinitionTerminating(err):
		default:
			return err
		}

		if !logged {
			log.Printf("Waiting to create %s %s/%s: %v", o.GetKind(), o.GetNamespace(), o.GetName(), err)
			logged = true
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
		}
	}
}

// isDefinitionTerminating tells whether err is the refusal of the API server to create a custom
// resource while its definition is being deleted: Forbidden since Kubernetes 1.34,
// MethodNotSupported before. It is told apart by its message, as the same errors also mean
// creation is denied or never supported: were the message to change, churn would stop on this
// error rather than wait forever on the others.
func isDefinitionTerminating(err error) bool {
	return (apierrors.IsForbidden(err) || apierrors.IsMethodNotSupported(err)) &&
		strings.Contains(err.Error(), "custom resource definition is terminating")
}

// deleteObject deletes o. An object that does not exist is not an error, nor is an object whose
// kind isn't served: a custom resource whose definition was deleted is deleted with it.
func deleteObject(ctx context.Context, res resources, o *unstructured.Unstructured, opts metav1.DeleteOptions) error {
	resource, err := res.of(o)
	if err == nil {
		err = resource.Delete(ctx, o.GetName(), opts)
	}
	if err != nil {
		if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			return nil
		}
		return err
	}
	log.Printf("Deleted %s %s/%s", o.GetKind(), o.GetNamespace(), o.GetName())
	return nil
}

// ignoreCancellation returns nil for an error caused by the cancellation of ctx, which is how
// handlers are asked to stop.
func ignoreCancellation(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return nil
	}
	return err
}

// transformObject returns a deep copy of u with f applied to every string value, at any depth.
// Map keys are left unchanged. It is used to render an instance from an object template.
func transformObject(u *unstructured.Unstructured, f func(in string) string) *unstructured.Unstructured {
	o := &unstructured.Unstructured{}
	o.SetUnstructuredContent(transformValue(u.UnstructuredContent(), f).(map[string]any))
	return o
}

func transformValue(v any, f func(in string) string) any {
	switch x := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(x))
		for k, v := range x {
			m[k] = transformValue(v, f)
		}
		return m

	case []any:
		s := make([]any, len(x))
		for i := range x {
			s[i] = transformValue(x[i], f)
		}
		return s

	case string:
		return f(x)

	case nil, int64, float64, bool:
		return x

	default:
		log.Fatalf("Unknown type: %T", x)
		return x
	}
}
