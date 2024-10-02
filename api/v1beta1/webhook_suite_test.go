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

package v1beta1

import (
	"context"
	"crypto/tls"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"net"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	v1 "k8s.io/api/admissionregistration/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	//+kubebuilder:scaffold:imports
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Webhook Suite")
}

func initAutoMQ() *AutoMQ {
	return &AutoMQ{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: AutoMQSpec{
			S3: S3Spec{
				Endpoint:        "http://localhost:9000",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
			},
		},
	}
}

var _ = Describe("Default", func() {
	Context("Default Webhook", func() {
		BeforeEach(func() {
			aq := initAutoMQ()
			_ = k8sClient.Delete(context.Background(), aq)
		})
		It("Default ImageName", func() {
			aq := initAutoMQ()
			err := k8sClient.Create(context.Background(), aq)
			Expect(err).To(BeNil())
			err = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(aq), aq)
			Expect(err).To(BeNil())
			Expect(aq.Spec.Image).To(Equal(DefaultImageName))
		})
		It("Default Region", func() {
			aq := initAutoMQ()
			err := k8sClient.Create(context.Background(), aq)
			Expect(err).To(BeNil())
			err = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(aq), aq)
			Expect(err).To(BeNil())
			Expect(aq.Spec.S3.Region).To(Equal("us-east-1"))
		})
		It("Default ClusterID", func() {
			aq := initAutoMQ()
			err := k8sClient.Create(context.Background(), aq)
			Expect(err).To(BeNil())
			err = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(aq), aq)
			Expect(err).To(BeNil())
			Expect(aq.Spec.ClusterID).To(Equal("rZdE0DjZSrqy96PXrMUZVw"))
			err = k8sClient.Delete(context.Background(), aq)
			Expect(err).To(BeNil())
		})
		It("Default Bucket", func() {
			aq := initAutoMQ()
			err := k8sClient.Create(context.Background(), aq)
			Expect(err).To(BeNil())
			err = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(aq), aq)
			Expect(err).To(BeNil())
			Expect(aq.Spec.S3.Bucket).To(Equal("ko3"))
		})
		It("Default Replicas", func() {
			aq := initAutoMQ()
			err := k8sClient.Create(context.Background(), aq)
			Expect(err).To(BeNil())
			err = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(aq), aq)
			Expect(err).To(BeNil())
			Expect(aq.Spec.Controller.Replicas).To(Equal(int32(1)))
			Expect(aq.Spec.Broker.Replicas).To(Equal(int32(1)))
		})
		It("Default JVM", func() {
			aq := initAutoMQ()
			err := k8sClient.Create(context.Background(), aq)
			Expect(err).To(BeNil())
			err = k8sClient.Get(context.Background(), client.ObjectKeyFromObject(aq), aq)
			Expect(err).To(BeNil())
			Expect(aq.Spec.Controller.JVMOptions).To(Equal([]string{"-Xms1g", "-Xmx1g", "-XX:MetaspaceSize=96m"}))
			Expect(aq.Spec.Broker.JVMOptions).To(Equal([]string{"-Xms1g", "-Xmx1g", "-XX:MetaspaceSize=96m", "-XX:MaxDirectMemorySize=1G"}))
		})
	})

})
var _ = Describe("Update", func() {
	Context("Update Webhook", func() {
		BeforeEach(func() {
			aq := initAutoMQ()
			_ = k8sClient.Delete(context.Background(), aq)
		})
		It("Update Endpoint", func() {
			aq := initAutoMQ()
			err := k8sClient.Create(context.Background(), aq)
			Expect(err).To(BeNil())
			aq.Spec.S3.Endpoint = "http://localhost:9001"
			err = k8sClient.Update(context.Background(), aq)
			Expect(true).To(Equal(errors.IsForbidden(err)))
			Expect(err.Error()).To(ContainSubstring("s3.Endpoint"))
			Expect(err.Error()).To(ContainSubstring("immutable"))
		})
		It("Update Region", func() {
			aq := initAutoMQ()
			err := k8sClient.Create(context.Background(), aq)
			Expect(err).To(BeNil())
			aq.Spec.S3.Region = "minioadmin1"
			err = k8sClient.Update(context.Background(), aq)
			Expect(true).To(Equal(errors.IsForbidden(err)))
			Expect(err.Error()).To(ContainSubstring("s3.Region"))
			Expect(err.Error()).To(ContainSubstring("immutable"))
		})
		It("Update Bucket", func() {
			aq := initAutoMQ()
			err := k8sClient.Create(context.Background(), aq)
			Expect(err).To(BeNil())
			aq.Spec.S3.Bucket = "minioadmin1"
			err = k8sClient.Update(context.Background(), aq)
			Expect(true).To(Equal(errors.IsForbidden(err)))
			Expect(err.Error()).To(ContainSubstring("s3.Bucket"))
			Expect(err.Error()).To(ContainSubstring("immutable"))
		})
		It("Update ClusterID", func() {
			aq := initAutoMQ()
			err := k8sClient.Create(context.Background(), aq)
			Expect(err).To(BeNil())
			aq.Spec.ClusterID = "minioadmin1"
			err = k8sClient.Update(context.Background(), aq)
			Expect(true).To(Equal(errors.IsForbidden(err)))
			Expect(err.Error()).To(ContainSubstring("clusterID"))
			Expect(err.Error()).To(ContainSubstring("immutable"))
		})
	})

})

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.28.0-%s-%s", runtime.GOOS, runtime.GOARCH)),

		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "config", "webhook")},
		},
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := apimachineryruntime.NewScheme()
	err = AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = admissionv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	err = v1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	err = apiextensionsv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// start webhook server using Manager
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookInstallOptions.LocalServingHost,
			Port:    webhookInstallOptions.LocalServingPort,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
		LeaderElection: false,
		Metrics:        metricsserver.Options{BindAddress: "0"},
	})
	Expect(err).NotTo(HaveOccurred())

	err = (&AutoMQ{}).SetupWebhookWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:webhook

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	// wait for the webhook server to get ready
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}).Should(Succeed())
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
