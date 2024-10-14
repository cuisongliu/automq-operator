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
	"fmt"

	"github.com/cuisongliu/automq-operator/defaults"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var automqlog = logf.Log.WithName("automq-resource")

func (r *AutoMQ) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-infra-cuisongliu-github-com-v1beta1-automq,mutating=true,failurePolicy=fail,sideEffects=None,groups=infra.cuisongliu.github.com,resources=automqs,verbs=create;update,versions=v1beta1,name=mautomq.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &AutoMQ{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *AutoMQ) Default() {
	automqlog.Info("default", "name", r.Name)
	if r.Spec.Image == "" {
		r.Spec.Image = defaults.DefaultImageName
	}
	if r.Spec.S3.Region == "" {
		r.Spec.S3.Region = "us-east-1"
	}
	if r.Spec.ClusterID == "" {
		r.Spec.ClusterID = "rZdE0DjZSrqy96PXrMUZVw"
	}
	if r.Spec.S3.Bucket == "" {
		r.Spec.S3.Bucket = "ko3"
	}
	if r.Spec.Controller.JVMOptions == nil {
		r.Spec.Controller.JVMOptions = []string{"-Xms1g", "-Xmx1g", "-XX:MetaspaceSize=96m"}
	}
	if r.Spec.Controller.Replicas == 0 {
		r.Spec.Controller.Replicas = 1
	}
	if r.Spec.Broker.JVMOptions == nil {
		r.Spec.Broker.JVMOptions = []string{"-Xms1g", "-Xmx1g", "-XX:MetaspaceSize=96m", "-XX:MaxDirectMemorySize=1G"}
	}
	if r.Spec.Broker.Replicas == 0 {
		r.Spec.Broker.Replicas = 1
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-infra-cuisongliu-github-com-v1beta1-automq,mutating=false,failurePolicy=fail,sideEffects=None,groups=infra.cuisongliu.github.com,resources=automqs,verbs=create;update,versions=v1beta1,name=vautomq.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &AutoMQ{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AutoMQ) ValidateCreate() (admission.Warnings, error) {
	automqlog.Info("validate create", "name", r.Name)
	if err := validate(r); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AutoMQ) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	automqlog.Info("validate update", "name", r.Name)
	mqOld := old.(*AutoMQ)

	if r.Spec.S3.Endpoint != mqOld.Spec.S3.Endpoint {
		return nil, fmt.Errorf("field s3.Endpoint is immutable")
	}
	if r.Spec.S3.Region != mqOld.Spec.S3.Region {
		return nil, fmt.Errorf("field s3.Region is immutable")
	}
	if r.Spec.S3.Bucket != mqOld.Spec.S3.Bucket {
		return nil, fmt.Errorf("field s3.Bucket is immutable")
	}
	if r.Spec.ClusterID != mqOld.Spec.ClusterID {
		return nil, fmt.Errorf("field clusterID is immutable")
	}
	if r.Spec.Controller.Replicas != mqOld.Spec.Controller.Replicas {
		return nil, fmt.Errorf("field controller.replicas is immutable")
	}
	if err := validate(r); err != nil {
		return nil, err
	}
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AutoMQ) ValidateDelete() (admission.Warnings, error) {
	automqlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}

func validate(r *AutoMQ) error {
	if r.Spec.S3.Endpoint == "" {
		return fmt.Errorf("field s3.Endpoint is required")
	}
	if r.Spec.S3.Region == "" {
		return fmt.Errorf("field s3.Region is required")
	}
	if r.Spec.S3.Bucket == "" {
		return fmt.Errorf("field s3.Bucket is required")
	}
	if r.Spec.ClusterID == "" {
		return fmt.Errorf("field clusterID is required")
	}
	if r.Spec.Image == "" {
		return fmt.Errorf("field image is required")
	}
	if len(r.Spec.Controller.JVMOptions) == 0 {
		return fmt.Errorf("field controller.jvmOptions is required")
	}
	if len(r.Spec.Broker.JVMOptions) == 0 {
		return fmt.Errorf("field broker.jvmOptions is required")
	}
	if r.Spec.Controller.Affinity != nil {
		if r.Spec.Controller.Affinity.PodAntiAffinity != nil {
			if r.Spec.Controller.Affinity.PodAntiAffinity.Type == "" {
				return fmt.Errorf("field controller.affinity.podAntiAffinity.type is required")
			}
		}
		if r.Spec.Controller.Affinity.PodAffinity != nil {
			if r.Spec.Controller.Affinity.PodAffinity.Type == "" {
				return fmt.Errorf("field controller.affinity.podAffinity.type is required")
			}
		}
		if r.Spec.Controller.Affinity.NodeAffinity != nil {
			if r.Spec.Controller.Affinity.NodeAffinity.Type == "" {
				return fmt.Errorf("field controller.affinity.nodeAffinity.type is required")
			}
		}
	}
	return nil
}
