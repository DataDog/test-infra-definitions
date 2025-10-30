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
	"syscall"
	"time"

	"github.com/gobuffalo/flect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

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

	errs := make(chan error)

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

			go func() {
				errs <- handleOneObject(ctx, client, obj)
			}()
		}
	}); err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case err := <-errs:
			if err != nil {
				log.Fatal(err)
			}
		case <-ctx.Done():
			break
		}
	}
}

func handleOneObject(ctx context.Context, client *dynamic.DynamicClient, obj *unstructured.Unstructured) error {
	gvr := obj.GroupVersionKind().GroupVersion().WithResource(strings.ToLower(flect.Pluralize(obj.GetKind())))

	nbInstances := 1
	if nb, ok := obj.GetAnnotations()["churn.datadoghq.com/instances"]; ok {
		var err error
		nbInstances, err = strconv.Atoi(nb)
		if err != nil {
			return err
		}
	}

	for i := range nbInstances {
		o := patchObject(obj, func(s string) string {
			return strings.ReplaceAll(s, "{{i}}", strconv.Itoa(i))
		})
		if _, err := client.Resource(gvr).Namespace(o.GetNamespace()).Create(ctx, o, metav1.CreateOptions{}); err != nil {
			return err
		}
		log.Printf("Created %s %s/%s", o.GetKind(), o.GetNamespace(), o.GetName())
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		for i := range nbInstances {
			o := patchObject(obj, func(s string) string {
				return strings.ReplaceAll(s, "{{i}}", strconv.Itoa(i))
			})
			un := int64(1)
			if err := client.Resource(gvr).Namespace(o.GetNamespace()).Delete(ctx, o.GetName(), metav1.DeleteOptions{
				GracePeriodSeconds: &un,
			}); err != nil {
				log.Print(err)
			}
			log.Printf("Deleted %s %s/%s", o.GetKind(), o.GetNamespace(), o.GetName())
		}
	}()

	var lifetime time.Duration
	if t, ok := obj.GetAnnotations()["churn.datadoghq.com/lifetime"]; ok {
		var err error
		lifetime, err = time.ParseDuration(t)
		if err != nil {
			return err
		}
	}

	if lifetime != 0 {
		i := 0
		ticker := time.NewTicker(time.Duration(int(lifetime) / nbInstances))
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				o := patchObject(obj, func(s string) string {
					return strings.ReplaceAll(s, "{{i}}", strconv.Itoa(i%nbInstances))
				})
				i++

				if err := client.Resource(gvr).Namespace(o.GetNamespace()).Delete(ctx, o.GetName(), metav1.DeleteOptions{}); err != nil {
					return err
				}
				log.Printf("Deleted %s %s/%s", o.GetKind(), o.GetNamespace(), o.GetName())

				if _, err := client.Resource(gvr).Namespace(o.GetNamespace()).Create(ctx, o, metav1.CreateOptions{}); err != nil {
					return err
				}
				log.Printf("Created %s %s/%s", o.GetKind(), o.GetNamespace(), o.GetName())

			}
		}
	}

	<-ctx.Done()
	return nil
}

func patchObject(u *unstructured.Unstructured, f func(in string) string) *unstructured.Unstructured {
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

	case nil, int64, bool:
		return x

	default:
		log.Fatalf("Unknown type: %T", x)
		return x
	}
}
