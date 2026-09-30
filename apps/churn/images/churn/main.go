package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
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

	"github.com/gobuffalo/flect"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// retryInterval is how long to wait before retrying the creation of an object whose previous
// instance is still being deleted, or whose namespace does not exist yet.
const retryInterval = 1 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dir := flag.String("manifests-dir", ".", "directory containing YAML manifests")
	flag.Parse()

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
	client := dynamic.NewForConfigOrDie(config)

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

			wg.Go(func() {
				if err := handleOneObject(runCtx, client, obj); err != nil {
					cancel(err)
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

func handleOneObject(ctx context.Context, client dynamic.Interface, obj *unstructured.Unstructured) error {
	gvr := obj.GroupVersionKind().GroupVersion().WithResource(strings.ToLower(flect.Pluralize(obj.GetKind())))

	nbInstances := 1
	if nb, ok := obj.GetAnnotations()["churn.datadoghq.com/instances"]; ok {
		var err error
		nbInstances, err = strconv.Atoi(nb)
		if err != nil {
			return err
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
			if err := deleteObject(ctx, client, gvr, instance(i), metav1.DeleteOptions{GracePeriodSeconds: new(int64(1))}); err != nil {
				log.Print(err)
			}
		}
	}()

	// An object left over by a previous churn instance that could not clean up (e.g. it was
	// killed) is created again from the template, as everything churn manages is transient.
	for i := range nbInstances {
		if err := recreateObject(ctx, client, gvr, instance(i)); err != nil {
			return ignoreCancellation(ctx, err)
		}
	}

	if lifetime == 0 {
		<-ctx.Done()
		return nil
	}

	ticker := time.NewTicker(time.Duration(int(lifetime) / nbInstances))
	defer ticker.Stop()
	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := recreateObject(ctx, client, gvr, instance(i%nbInstances)); err != nil {
				return ignoreCancellation(ctx, err)
			}
		}
	}
}

// recreateObject deletes o if it exists and creates it again.
func recreateObject(ctx context.Context, client dynamic.Interface, gvr schema.GroupVersionResource, o *unstructured.Unstructured) error {
	if err := deleteObject(ctx, client, gvr, o, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return createObject(ctx, client, gvr, o)
}

// createObject creates o. Creation is retried while a deleted object with the same name is not
// gone yet, and while the namespace of o does not exist yet or is being deleted, since
// namespaces are created and deleted by other handlers running concurrently.
func createObject(ctx context.Context, client dynamic.Interface, gvr schema.GroupVersionResource, o *unstructured.Unstructured) error {
	resource := client.Resource(gvr).Namespace(o.GetNamespace())
	logged := false
	for {
		_, err := resource.Create(ctx, o, metav1.CreateOptions{})
		switch {
		case err == nil:
			log.Printf("Created %s %s/%s", o.GetKind(), o.GetNamespace(), o.GetName())
			return nil
		case apierrors.IsAlreadyExists(err), apierrors.IsNotFound(err), apierrors.HasStatusCause(err, "NamespaceTerminating"):
			if !logged {
				log.Printf("Waiting to create %s %s/%s: %v", o.GetKind(), o.GetNamespace(), o.GetName(), err)
				logged = true
			}
		default:
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
		}
	}
}

// deleteObject deletes o. An object that does not exist is not an error.
func deleteObject(ctx context.Context, client dynamic.Interface, gvr schema.GroupVersionResource, o *unstructured.Unstructured, opts metav1.DeleteOptions) error {
	if err := client.Resource(gvr).Namespace(o.GetNamespace()).Delete(ctx, o.GetName(), opts); err != nil {
		if apierrors.IsNotFound(err) {
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
