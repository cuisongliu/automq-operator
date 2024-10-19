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
	"os"
	"time"

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
				automq.Spec.S3.Endpoint = "http://minio.minio.svc.cluster.local:9000"
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
