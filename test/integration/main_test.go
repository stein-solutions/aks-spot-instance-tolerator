//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	v1core "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/kind"
)

var (
	testenv env.Environment
)

func installHelmChart() error {
	pwd, _ := os.Getwd()
	chartName := filepath.Join(pwd, "../../helm", "aks-spot-instance-tolerator")
	releaseName := "aks-spot-instance-tolerator"

	cmd := exec.Command("helm", "upgrade", "--install", releaseName, chartName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Fehler beim Installieren des Helm-Charts: %v, %s", err, string(output))
	}

	return nil
}

func TestKubernetes(t *testing.T) {
	f1 := features.New("apply new pod").
		WithLabel("type", "pod-count").
		Assess("should not fail", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {

			var testPod = v1core.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Spec: v1core.PodSpec{
					Containers: []v1core.Container{
						{
							Name:  "test-container",
							Image: "nginx",
						},
					},
				},
			}

			if err := cfg.Client().Resources().WithNamespace("default").Create(context.TODO(), &testPod); err != nil {
				t.Fatal(err)
			}

			if err := wait.For(
				conditions.New(cfg.Client().Resources()).ResourceMatch(&testPod, func(object k8s.Object) bool {
					pod := object.(*v1core.Pod)
					for _, toleration := range pod.Spec.Tolerations {
						if toleration.Key == "kubernetes.azure.com/scalesetpriority" && toleration.Value == "spot" {
							return true
						}
					}
					return false
				}),
				wait.WithTimeout(5*time.Minute),
				wait.WithInterval(10*time.Second),
			); err != nil {
				t.Fatal(err)
			}

			return ctx
		}).WithTeardown("teardown", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		cfg.Client().Resources().WithNamespace("default").Delete(context.TODO(), &v1core.Pod{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
		})
		return ctx
	}).Feature()

	f2 := features.New("count namespaces").
		WithLabel("type", "ns-count").
		Assess("namespace exist", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			var nspaces v1core.NamespaceList
			err := cfg.Client().Resources().List(context.TODO(), &nspaces)
			if err != nil {
				t.Fatal(err)
			}
			if len(nspaces.Items) == 1 {
				t.Fatal("no other namespace")
			}
			return ctx
		}).Feature()

	f3 := features.New("check health probes").
		WithLabel("type", "health-probes").
		Assess("healthz", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			resp, err := http.Get("http://localhost:8080/healthz")
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected status code 200, got %d", resp.StatusCode)
			}
			return ctx
		}).Assess("Readyz", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		resp, err := http.Get("http://localhost:8080/readyz")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status code 200, got %d", resp.StatusCode)
		}
		return ctx
	}).Feature()

	// test feature
	testenv.Test(t, f1, f2, f3)
}

func TestMain(m *testing.M) {
	testenv = env.New()
	kindClusterName := "aks-spot-instance-tolerator-webhook-cluster"
	namespace := envconf.RandomName("myns", 16)
	kindCluster := kind.NewCluster(kindClusterName)

	// Use pre-defined environment funcs to create a kind cluster prior to test run
	testenv.Setup(
		envfuncs.CreateCluster(kindCluster, kindClusterName),
		func(ctx context.Context, c *envconf.Config) (context.Context, error) {

			err := installHelmChart()
			if err != nil {
				return nil, err
			}
			client := c.Client()

			if err := wait.For(
				conditions.New(client.Resources().WithNamespace("default")).
					DeploymentAvailable("aks-spot-instance-tolerator", "default"),
				wait.WithTimeout(5*time.Minute),
				wait.WithInterval(10*time.Second),
			); err != nil {
				return nil, err
			}
			return ctx, nil
		},
	)

	// Use pre-defined environment funcs to teardown kind cluster after tests
	testenv.Finish(
		envfuncs.DestroyCluster(kindClusterName),
		envfuncs.DeleteNamespace(namespace),
	)

	// launch package tests
	os.Exit(testenv.Run(m))

}
