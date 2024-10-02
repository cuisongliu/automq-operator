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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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
	Context("install", func() {
		It("PrometheusOperator", func() {
			var err error
			err = InstallPrometheusOperator()
			Expect(err).To(Not(HaveOccurred()))
		})

	})
	Context("automq_controller tests", func() {
		ctx := context.Background()
		namespaceName := "automq-operator"
		namespace := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespaceName,
				Namespace: namespaceName,
			},
		}
		automq := &infrav1beta1.AutoMQ{}
		automq.Name = "automq-s1"
		automq.Namespace = namespaceName
		automq.Spec.ClusterID = "rZdE0DjZSrqy96PXrMUZVw"

		BeforeEach(func() {
			By("Creating the Namespace to perform the tests")
			err := k8sClient.Create(ctx, namespace)
			Expect(err).To(Not(HaveOccurred()))

			By("Setting the NAMESPACE_NAME ENV VAR which stores the Operand image")
			err = os.Setenv("NAMESPACE_NAME", namespaceName)
			Expect(err).To(Not(HaveOccurred()))
		})
		It("Update Endpoint", func() {
			By("creating the custom resource for the automq")
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(automq), automq)
			if err != nil && errors.IsNotFound(err) {
				// Let's mock our custom resource at the same way that we would
				// apply on the cluster the manifest under config/samples
				automq.Spec.S3.Endpoint = "http://localhost:9000"
				automq.Spec.S3.Bucket = "ko3"
				automq.Spec.S3.AccessKeyID = "min"
				automq.Spec.S3.SecretAccessKey = "test"
				automq.Spec.S3.Region = "us-east-1"
				err = k8sClient.Create(ctx, automq)
				Expect(err).To(Not(HaveOccurred()))
			}
		})
		AfterEach(func() {
			By("removing the custom resource for the automq")
			found := &infrav1beta1.AutoMQ{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(automq), found)
			Expect(err).To(Not(HaveOccurred()))

			Eventually(func() error {
				return k8sClient.Delete(context.TODO(), found)
			}, 2*time.Minute, time.Second).Should(Succeed())

			// TODO(user): Attention if you improve this code by adding other context test you MUST
			// be aware of the current delete namespace limitations.
			// More info: https://book.kubebuilder.io/reference/envtest.html#testing-considerations
			By("Deleting the Namespace to perform the tests")
			_ = k8sClient.Delete(ctx, namespace)

			By("Removing the Image ENV VAR which stores the Operand image")
			_ = os.Unsetenv("NAMESPACE_NAME")
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

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
