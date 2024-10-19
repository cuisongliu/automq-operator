/*
Copyright 2024 cuisongliu.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cuisongliu/automq-operator/internal/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	infrav1beta1 "github.com/cuisongliu/automq-operator/api/v1beta1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var kubeconfigPath string

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = Describe("automq_controller", func() {
	Context("automq_controller apis tests", func() {
		ctx := context.Background()
		It("check api", func() {
			Eventually(func() error {
				nodes := &v1.NodeList{}
				err := k8sClient.List(ctx, nodes)
				if err != nil {
					return fmt.Errorf("list node error %s", err.Error())
				}
				if len(nodes.Items) == 0 {
					return fmt.Errorf("expected 1 node, found %d", len(nodes.Items))
				}
				nodeName := nodes.Items[0].Name
				nodeIp := nodes.Items[0].Status.Addresses[0].Address
				if nodeIp == "" {
					return fmt.Errorf("node ip not found")
				}
				if nodeName == "" {
					return fmt.Errorf("node name not found")
				}
				ip := os.Getenv("OPERATOR_APIS_IP")
				if ip == "" {
					return fmt.Errorf("OPERATOR_APIS_IP is empty")
				}
				apiAddr := fmt.Sprintf("http://%s:9090/api/v1/nodes/%s", ip, nodeName)
				out := RestHttpApi(ctx, apiAddr, "GET", nil, 0)
				if out.Code != 200 {
					return fmt.Errorf("api response code %d", out.Code)
				}
				if string(out.Data) != nodeIp {
					return fmt.Errorf("api response %s", string(out.Data))
				}
				return nil
			}, "60s", "1s").Should(Succeed())
		})
	})
	Context("automq_controller component tests", func() {
		ctx := context.Background()
		It("check minio status", func() {
			Eventually(func() error {
				podList := &v1.PodList{}
				labelSelector := labels.Set(map[string]string{"release": "minio"}).AsSelector()
				err := k8sClient.List(ctx, podList, &client.ListOptions{Namespace: "minio", LabelSelector: labelSelector})
				if err != nil {
					return err
				}
				if len(podList.Items) == 0 {
					return fmt.Errorf("expected 1 pod, found %d", len(podList.Items))
				}
				if podList.Items[0].Status.Phase != v1.PodRunning {
					return fmt.Errorf("expected pod phase to be 'Running', got '%s'", podList.Items[0].Status.Phase)
				}
				return nil
			}, "60s", "1s").Should(Succeed())
		})
		It("check cert-manager status", func() {
			Eventually(func() error {
				podList := &v1.PodList{}
				labelSelector := labels.Set(map[string]string{"app.kubernetes.io/instance": "cert-manager"}).AsSelector()
				err := k8sClient.List(ctx, podList, &client.ListOptions{Namespace: "cert-manager", LabelSelector: labelSelector})
				if err != nil {
					return err
				}
				if len(podList.Items) == 0 {
					return fmt.Errorf("expected 1 pod, found %d", len(podList.Items))
				}
				if podList.Items[0].Status.Phase != v1.PodRunning {
					return fmt.Errorf("expected pod phase to be 'Running', got '%s'", podList.Items[0].Status.Phase)
				}
				return nil
			}, "60s", "1s").Should(Succeed())
		})
		It("check prometheus status", func() {
			Eventually(func() error {
				podList := &v1.PodList{}
				labelSelector := labels.Set(map[string]string{"release": "prometheus"}).AsSelector()
				err := k8sClient.List(ctx, podList, &client.ListOptions{Namespace: "monitoring", LabelSelector: labelSelector})
				if err != nil {
					return err
				}
				if len(podList.Items) == 0 {
					return fmt.Errorf("expected 1 pod, found %d", len(podList.Items))
				}
				if podList.Items[0].Status.Phase != v1.PodRunning {
					return fmt.Errorf("expected pod phase to be 'Running', got '%s'", podList.Items[0].Status.Phase)
				}
				return nil
			}, "60s", "1s").Should(Succeed())
		})
		It("check kafka-ui status", func() {
			Eventually(func() error {
				podList := &v1.PodList{}
				labelSelector := labels.Set(map[string]string{"release": "minio"}).AsSelector()
				err := k8sClient.List(ctx, podList, &client.ListOptions{Namespace: "minio", LabelSelector: labelSelector})
				if err != nil {
					return err
				}
				if len(podList.Items) == 0 {
					return fmt.Errorf("expected 1 pod, found %d", len(podList.Items))
				}
				if podList.Items[0].Status.Phase != v1.PodRunning {
					return fmt.Errorf("expected pod phase to be 'Running', got '%s'", podList.Items[0].Status.Phase)
				}
				return nil
			}, "60s", "1s").Should(Succeed())
		})
	})

})

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	usingCluster := true
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		UseExistingCluster:    &usingCluster,
		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "bin", "k8s",
			fmt.Sprintf("1.28.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	// cfg is defined in this file globally.
	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = infrav1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = clientgoscheme.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = promv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	err = os.Setenv("ENABLE_WEBHOOKS", "false")
	Expect(err).To(Not(HaveOccurred()))

	err = os.Setenv("OPERATOR_APIS_IP", GetLocalIpv4())
	Expect(err).To(Not(HaveOccurred()))

	go func() {
		controller.APIRegistry(context.Background(), k8sClient)
	}()
	err = os.Setenv("NAMESPACE_NAME", "default")
	Expect(err).To(Not(HaveOccurred()))
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
