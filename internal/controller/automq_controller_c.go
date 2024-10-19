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

package controller

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	infrav1beta1 "github.com/cuisongliu/automq-operator/api/v1beta1"
	"github.com/cuisongliu/automq-operator/defaults"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	controllerRole = "controller"
)

func (r *AutoMQReconciler) cleanController(ctx context.Context, obj *infrav1beta1.AutoMQ) error {
	for i := 0; i < int(obj.Status.ControllerReplicas); i++ {
		svcc := &v1.Service{}
		svcc.Namespace = obj.Namespace
		index := int32(i)
		svcc.Name = getAutoMQName(controllerRole, &index)
		_ = r.Client.Delete(ctx, svcc)

		deploy := &appsv1.Deployment{}
		deploy.Namespace = obj.Namespace
		deploy.Name = getAutoMQName(controllerRole, &index)
		_ = r.Client.Delete(ctx, deploy)

		pvc := &v1.PersistentVolumeClaim{}
		pvc.Namespace = obj.Namespace
		pvc.Name = getAutoMQName(controllerRole, &index)
		_ = r.Client.Delete(ctx, pvc)
	}
	return nil
}

func (r *AutoMQReconciler) syncControllersScale(ctx context.Context, obj *infrav1beta1.AutoMQ) bool {
	conditionType := "SyncControllerScale"
	currentReplicas := obj.Status.ControllerReplicas
	if currentReplicas > obj.Spec.Controller.Replicas {
		for i := obj.Spec.Controller.Replicas; i < currentReplicas; i++ {
			deploy := &appsv1.Deployment{}
			deploy.Namespace = obj.Namespace
			deploy.Name = getAutoMQName(controllerRole, &i)
			_ = r.Client.Delete(ctx, deploy)
			svc := &v1.Service{}
			svc.Namespace = obj.Namespace
			svc.Name = getAutoMQName(controllerRole, &i)
			_ = r.Client.Delete(ctx, svc)
			pvc := &v1.PersistentVolumeClaim{}
			pvc.Namespace = obj.Namespace
			pvc.Name = getAutoMQName(controllerRole, &i)
			_ = r.Client.Delete(ctx, pvc)
		}
	}
	meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: obj.Generation,
		Reason:             "ControllerScaleReconciling",
		Message:            fmt.Sprintf("Controller scale for the custom resource (%s) has been reconciled", obj.Name),
	})
	return true
}

func (r *AutoMQReconciler) syncControllers(ctx context.Context, obj *infrav1beta1.AutoMQ) bool {
	conditionType := "SyncControllerReady"
	log := log.FromContext(ctx)
	// 1. sync pvc
	// 2. sync deploy
	// 3. sync svc
	// 3. sync monitor

	for i := 0; i < int(obj.Spec.Controller.Replicas); i++ {
		if err := r.syncControllerPVC(ctx, obj, int32(i)); err != nil {
			meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
				Type:               conditionType,
				Status:             metav1.ConditionFalse,
				ObservedGeneration: obj.Generation,
				Reason:             "ControllerPVCReconciling",
				Message:            fmt.Sprintf("Failed to create pvc for the custom resource (%s): (%s)", obj.Name, err),
			})
			log.Error(err, "Failed to create pvc for the custom resource (%s)", obj.Name, "role", controllerRole)
			return true
		}
		if err := r.syncControllerDeploy(ctx, obj, int32(i)); err != nil {
			meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
				Type:               conditionType,
				Status:             metav1.ConditionFalse,
				ObservedGeneration: obj.Generation,
				Reason:             "ControllerSTSReconciling",
				Message:            fmt.Sprintf("Failed to create deploy for the custom resource (%s): (%s)", obj.Name, err),
			})
			log.Error(err, "Failed to create deploy for the custom resource (%s)", obj.Name, "role", controllerRole)
			return true
		}
		if err := r.syncControllerService(ctx, obj, int32(i)); err != nil {
			meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
				Type:               conditionType,
				Status:             metav1.ConditionFalse,
				ObservedGeneration: obj.Generation,
				Reason:             "ControllerServiceReconciling",
				Message:            fmt.Sprintf("Failed to create service for the custom resource (%s): (%s)", obj.Name, err),
			})
			log.Error(err, "Failed to create service for the custom resource (%s)", obj.Name, "role", controllerRole)
			return true
		}
	}
	meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: obj.Generation,
		Reason:             "ControllerReconciling",
		Message:            fmt.Sprintf("Controller resource for the custom resource (%s) has been created or update", obj.Name),
	})
	return true
}

func (r *AutoMQReconciler) syncControllerPVC(ctx context.Context, obj *infrav1beta1.AutoMQ, index int32) error {
	storage, _ := resource.ParseQuantity("100Gi")
	pvc := &v1.PersistentVolumeClaim{}
	pvc.Namespace = obj.Namespace
	pvc.Name = getAutoMQName(controllerRole, &index)
	labelMap := getAutoMQLabelMap(obj.GetName(), controllerRole)
	labelMap[autoMQIndexKey] = fmt.Sprintf("%d", index)
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, pvc, func() error {
			pvc.Labels = labelMap
			pvc.Spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
			pvc.Spec.Resources = v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: storage,
				},
			}
			if obj.Spec.Controller.StorageClass != "" {
				pvc.Spec.StorageClassName = &obj.Spec.Controller.StorageClass
			}
			return nil
		})
		return err
	}); err != nil {
		return err
	}
	return nil
}

func (r *AutoMQReconciler) controllerVoters(obj *infrav1beta1.AutoMQ) []string {
	var voters []string
	for i := 0; i < int(obj.Spec.Controller.Replicas); i++ {
		index := int32(i)
		voters = append(voters, fmt.Sprintf("%d@%s.%s.svc:%d", i, getAutoMQName(controllerRole, &index), obj.Namespace, 9093))
	}
	return voters
}

func (r *AutoMQReconciler) syncControllerDeploy(ctx context.Context, obj *infrav1beta1.AutoMQ, index int32) error {
	deploy := &appsv1.Deployment{}
	deploy.Namespace = obj.Namespace
	deploy.Name = getAutoMQName(controllerRole, &index)
	labelMap := getAutoMQLabelMap(obj.GetName(), controllerRole)
	labelMap[autoMQIndexKey] = fmt.Sprintf("%d", index)
	deploy.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: labelMap,
	}
	sysctl := sysctlContainer()
	envs := []v1.EnvVar{
		{
			Name: "NAMESPACE_NAME",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name: "POD_NAME",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: "POD_IP",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
		{
			Name:  "KAFKA_S3_ACCESS_KEY",
			Value: obj.Spec.S3.AccessKeyID,
		},
		{
			Name:  "KAFKA_S3_SECRET_KEY",
			Value: obj.Spec.S3.SecretAccessKey,
		},
		{
			Name:  "KAFKA_HEAP_OPTS",
			Value: strings.Join(obj.Spec.Controller.JVMOptions, " "),
		},
	}
	cmds := []string{
		"/opt/kafka/scripts/mq-start.sh",
		"up",
		"--process.roles",
		"controller",
		"--node.id",
		fmt.Sprintf("%d", index),
		"--cluster.id",
		obj.Spec.ClusterID,
		"--controller.quorum.voters",
		strings.Join(r.controllerVoters(obj), ","),
		"--s3.bucket",
		obj.Spec.S3.Bucket,
		"--s3.endpoint",
		obj.Spec.S3.Endpoint,
		"--s3.region",
		obj.Spec.S3.Region,
		"--s3.path.style",
		fmt.Sprintf("%t", obj.Spec.S3.EnablePathStyle),
	}
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploy, func() error {
			deploy.Labels = getAutoMQLabelMap(obj.GetName(), controllerRole)
			deploy.Spec.Strategy = appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			}
			deploy.Spec.Template.Labels = labelMap
			deploy.Spec.Template.Spec.HostNetwork = false
			deploy.Spec.Template.Spec.TerminationGracePeriodSeconds = aws.Int64(60 * 2)
			deploy.Spec.Template.Spec.InitContainers = []v1.Container{
				sysctl,
			}
			deploy.Spec.Template.Spec.Affinity = obj.Spec.Controller.Affinity.ToK8sAffinity()
			deploy.Spec.Template.Spec.Volumes = []v1.Volume{
				{
					Name: "script",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: obj.Name,
							},
							DefaultMode: aws.Int32(0755),
						},
					},
				},
				{
					Name: deploy.Name,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: deploy.Name,
						},
					},
				},
			}
			deploy.Spec.Template.Spec.Containers = []v1.Container{
				{
					Name:  controllerRole,
					Image: defaults.DefaultImageName,
					Env:   envs,
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      deploy.Name,
							MountPath: "/data/kafka",
						},
						{
							Name:      "script",
							MountPath: "/opt/kafka/scripts/mq-start.sh",
							SubPath:   "up.sh",
							ReadOnly:  false,
						},
					},
					Lifecycle: &v1.Lifecycle{
						PreStop: &v1.LifecycleHandler{
							Exec: &v1.ExecAction{
								Command: []string{
									"bash",
									"-c",
									"/opt/kafka/kafka/bin/kafka-server-stop.sh",
								},
							},
						},
					},
					Command: []string{
						"/bin/bash",
						"-c",
						strings.Join(cmds, " \\\n"),
					},
					LivenessProbe: &v1.Probe{
						ProbeHandler: v1.ProbeHandler{
							TCPSocket: &v1.TCPSocketAction{
								Port: intstr.FromString(controllerRole),
							},
						},
						InitialDelaySeconds:           20,
						TimeoutSeconds:                10,
						PeriodSeconds:                 30,
						SuccessThreshold:              1,
						FailureThreshold:              4,
						TerminationGracePeriodSeconds: nil,
					},
					Ports: []v1.ContainerPort{
						{
							Name:          controllerRole,
							ContainerPort: 9093,
							Protocol:      v1.ProtocolTCP,
						},
					},
					ImagePullPolicy: v1.PullIfNotPresent,
				},
			}
			hash, ok := ctx.Value(ctxKey("hash-configmap")).(string)
			if !ok {
				hash = ""
			}
			deploy.Spec.Template.Annotations = map[string]string{
				"configmap/script-hash": hash,
			}

			if obj.Spec.Metrics.Enable {
				deploy.Spec.Template.Spec.Containers[0].Env = append(deploy.Spec.Template.Spec.Containers[0].Env, v1.EnvVar{
					Name:  "KAFKA_CFG_S3_TELEMETRY_METRICS_EXPORTER_URI",
					Value: "prometheus://?host=0.0.0.0&port=9090",
				})
				deploy.Spec.Template.Annotations["prometheus.io/scrape"] = "true"
				deploy.Spec.Template.Annotations["prometheus.io/port"] = "9090"
				deploy.Spec.Template.Annotations["prometheus.io/path"] = "/metrics"
			}
			if r.MountTZ {
				deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, v1.Volume{
					Name: "k8tz",
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: "/etc/localtime",
						},
					},
				})
				deploy.Spec.Template.Spec.Containers[0].VolumeMounts = append(deploy.Spec.Template.Spec.Containers[0].VolumeMounts, v1.VolumeMount{
					Name:      "k8tz",
					MountPath: "/etc/localtime",
				})
			}
			if obj.Spec.Controller.Resource.Requests != nil {
				deploy.Spec.Template.Spec.Containers[0].Resources.Requests = obj.Spec.Controller.Resource.Requests
			}
			if obj.Spec.Controller.Resource.Limits != nil {
				deploy.Spec.Template.Spec.Containers[0].Resources.Limits = obj.Spec.Controller.Resource.Limits
			}
			if obj.Spec.Controller.Envs != nil && len(obj.Spec.Controller.Envs) > 0 {
				deploy.Spec.Template.Spec.Containers[0].Env = append(deploy.Spec.Template.Spec.Containers[0].Env, obj.Spec.Controller.Envs...)
			}
			return nil
		})
		return err
	}); err != nil {
		return err
	}
	return nil
}
func (r *AutoMQReconciler) syncControllerService(ctx context.Context, obj *infrav1beta1.AutoMQ, index int32) error {
	svc := &v1.Service{}
	svc.Namespace = obj.Namespace
	svc.Name = getAutoMQName(controllerRole, &index)
	labelMap := getAutoMQLabelMap(obj.GetName(), controllerRole)
	labelMap[autoMQIndexKey] = fmt.Sprintf("%d", index)
	svc.Spec.Selector = labelMap
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
			svc.Labels = labelMap
			svc.Spec.Ports = []v1.ServicePort{
				{
					Name:       controllerRole,
					Port:       9093,
					TargetPort: intstr.FromString(controllerRole),
					Protocol:   v1.ProtocolTCP,
				},
			}
			svc.Spec.Type = v1.ServiceTypeClusterIP
			return nil
		})
		return err
	}); err != nil {
		return err
	}
	return nil
}
