/*
Copyright 2024 cuisongliu@qq.com.

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
	"time"

	"github.com/cuisongliu/automq-operator/internal/controller"
	v2 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1beta1 "github.com/cuisongliu/automq-operator/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("automq_controller", func() {
	Context("automq_controller cr tests", func() {
		ctx := context.Background()
		namespaceName := "automq-cr"
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
		It("create cr namespace", func() {
			By("Creating the Namespace to perform the tests")
			err := k8sClient.Create(ctx, namespace)
			Expect(err).To(Not(HaveOccurred()))
		})
		It("create cr", func() {
			By("get minio ip and port")
			minioService := &v1.Service{}
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: "minio", Name: "minio"}, minioService)
			Expect(err).To(Not(HaveOccurred()))
			ip := minioService.Spec.ClusterIP
			By("creating the custom resource for the automq")
			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(automq), automq)
			if err != nil && errors.IsNotFound(err) {
				// Let's mock our custom resource at the same way that we would
				// apply on the cluster the manifest under config/samples
				automq.Spec.S3.Endpoint = fmt.Sprintf("http://%s:9000", ip)
				automq.Spec.S3.Bucket = "ko3"
				automq.Spec.S3.AccessKeyID = "admin"
				automq.Spec.S3.SecretAccessKey = "minio123"
				automq.Spec.S3.Region = "us-east-1"
				automq.Spec.S3.EnablePathStyle = true
				automq.Spec.Controller.Replicas = 1
				automq.Spec.Broker.Replicas = 3
				automq.Spec.NodePort = 32009
				err = k8sClient.Create(ctx, automq)
				Expect(err).To(Not(HaveOccurred()))
			}
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &controller.AutoMQReconciler{
				Client:    k8sClient,
				Scheme:    k8sClient.Scheme(),
				Finalizer: "apps.cuisongliu.com/automq.finalizer",
				MountTZ:   true,
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: client.ObjectKeyFromObject(automq),
			})
			Expect(err).NotTo(HaveOccurred())
		})
		It("get automq deployment", func() {
			ctx := context.Background()
			Eventually(func() error {
				deployment := &v2.DeploymentList{}
				labelSelector := labels.Set(map[string]string{"app.kubernetes.io/owner-by": "automq", "app.kubernetes.io/instance": automq.Name}).AsSelector()
				err := k8sClient.List(ctx, deployment, &client.ListOptions{Namespace: automq.Namespace, LabelSelector: labelSelector})
				if err != nil {
					return err
				}
				if len(deployment.Items) != 4 {
					return fmt.Errorf("expected 4 deploy, found %d", len(deployment.Items))
				}
				for i, deploy := range deployment.Items {
					if deploy.Status.ReadyReplicas != 1 {
						return fmt.Errorf("expected deploy %d ready replicas to be 1, got '%d'", i, deploy.Status.ReadyReplicas)
					}
				}
				return nil
			}, "60s", "1s").Should(Succeed())
		})
		It("check controller status", func() {
			ctx := context.Background()
			Eventually(func() error {
				podList := &v1.PodList{}
				labelSelector := labels.Set(map[string]string{"app.kubernetes.io/owner-by": "automq", "app.kubernetes.io/instance": automq.Name, "app.kubernetes.io/role": "controller"}).AsSelector()
				err := k8sClient.List(ctx, podList, &client.ListOptions{Namespace: automq.Namespace, LabelSelector: labelSelector})
				if err != nil {
					return err
				}
				if len(podList.Items) != 1 {
					return fmt.Errorf("expected 3 pod, found %d", len(podList.Items))
				}
				for i, pod := range podList.Items {
					if pod.Status.Phase != v1.PodRunning {
						return fmt.Errorf("expected pod %d phase to be 'Running', got '%s'", i, pod.Status.Phase)
					}
				}
				return nil
			}, "60s", "1s").Should(Succeed())
		})

		It("check broker status", func() {
			ctx := context.Background()
			Eventually(func() error {
				podList := &v1.PodList{}
				labelSelector := labels.Set(map[string]string{"app.kubernetes.io/owner-by": "automq", "app.kubernetes.io/instance": automq.Name, "app.kubernetes.io/role": "broker"}).AsSelector()
				err := k8sClient.List(ctx, podList, &client.ListOptions{Namespace: automq.Namespace, LabelSelector: labelSelector})
				if err != nil {
					return err
				}
				if len(podList.Items) != 3 {
					return fmt.Errorf("expected 1 pod, found %d", len(podList.Items))
				}
				for i, pod := range podList.Items {
					if pod.Status.Phase != v1.PodRunning {
						return fmt.Errorf("expected pod %d phase to be 'Running', got '%s'", i, pod.Status.Phase)
					}
				}
				return nil
			}, "60s", "1s").Should(Succeed())
		})
		It("check automq status", func() {
			ctx := context.Background()
			Eventually(func() error {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(automq), automq)
				if err != nil {
					return err
				}
				if automq.Status.Phase != infrav1beta1.AutoMQReady {
					return fmt.Errorf("expected automq phase to be 'Ready', got '%s'", automq.Status.Phase)
				}
				if automq.Status.ControllerReplicas != automq.Spec.Controller.Replicas {
					return fmt.Errorf("expected automq controller replicas to be %d, got '%d'", automq.Spec.Controller.Replicas, automq.Status.ControllerReplicas)
				}
				if automq.Status.BrokerReplicas != automq.Spec.Broker.Replicas {
					return fmt.Errorf("expected automq broker replicas to be %d, got '%d'", automq.Spec.Broker.Replicas, automq.Status.BrokerReplicas)
				}
				showReadyPods := automq.Spec.Controller.Replicas + automq.Spec.Broker.Replicas
				if automq.Status.ReadyPods != showReadyPods {
					return fmt.Errorf("expected automq ready pods to be %d, got '%d'", showReadyPods, automq.Status.ReadyPods)
				}
				if len(automq.Status.ControllerAddresses) != int(automq.Spec.Controller.Replicas) {
					return fmt.Errorf("expected automq controller addresses to have %d elements, got '%d'", automq.Spec.Controller.Replicas, len(automq.Status.ControllerAddresses))
				}
				if automq.Status.BootstrapInternalAddress == "" {
					return fmt.Errorf("expected automq bootstrap internal address to be set")
				}
				bootstrapService := fmt.Sprintf("%s.%s.svc:%d", "automq-"+"broker-bootstrap", automq.Namespace, 9092)
				if automq.Status.BootstrapInternalAddress != bootstrapService {
					return fmt.Errorf("expected automq bootstrap internal address to be '%s', got '%s'", bootstrapService, automq.Status.BootstrapInternalAddress)
				}
				for i, address := range automq.Status.ControllerAddresses {
					controllerService := fmt.Sprintf("%d@%s.%s.svc:%d", i, "automq-controller-"+fmt.Sprintf("%d", i), automq.Namespace, 9093)
					if address != controllerService {
						return fmt.Errorf("expected automq controller address %d to be '%s', got '%s'", i, controllerService, address)
					}
				}
				return nil
			}, "60s", "1s").Should(Succeed())
		})
		It("clean automq", func() {
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
