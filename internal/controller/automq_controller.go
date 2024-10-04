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

package controller

import (
	"context"
	"errors"
	"fmt"
	infrav1beta1 "github.com/cuisongliu/automq-operator/api/v1beta1"
	"github.com/cuisongliu/automq-operator/defaults"
	"github.com/cuisongliu/automq-operator/internal/pkg/storage"
	"github.com/labring/operator-sdk/controller"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerlib "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

// AutoMQReconciler reconciles a AutoMQ object
type AutoMQReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	Finalizer string
	MountTZ   bool
}

// finalizeSetting will perform the required operations before delete the CR.
func (r *AutoMQReconciler) doFinalizerOperationsForSetting(ctx context.Context, automq *infrav1beta1.AutoMQ) error {
	return r.cleanup(ctx, automq)
}

func (r *AutoMQReconciler) cleanup(ctx context.Context, automq *infrav1beta1.AutoMQ) error {
	return nil
}

//+kubebuilder:rbac:groups=infra.cuisongliu.github.com,resources=automqs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.cuisongliu.github.com,resources=automqs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infra.cuisongliu.github.com,resources=automqs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AutoMQ object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *AutoMQReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	autoMQFinalizer := r.Finalizer
	// Fetch the Setting instance
	// The purpose is check if the Custom Resource for the Kind Setting
	// is applied on the cluster if not we return nil to stop the reconciliation
	autoMQ := &infrav1beta1.AutoMQ{}

	err := r.Get(ctx, req.NamespacedName, autoMQ)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if autoMQ.GetDeletionTimestamp() != nil && !autoMQ.GetDeletionTimestamp().IsZero() {
		if err = r.doFinalizerOperationsForSetting(ctx, autoMQ); err != nil {
			return ctrl.Result{}, err
		}
		if controllerutil.ContainsFinalizer(autoMQ, autoMQFinalizer) {
			controllerutil.RemoveFinalizer(autoMQ, autoMQFinalizer)
		}
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.Update(ctx, autoMQ)
		}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err = r.statusReconcile(ctx, autoMQ); err != nil {
					lg := log.FromContext(ctx)
					lg.Error(err, "Failed to update automq status")
				}
			}
		}
	}()

	if autoMQ.GetDeletionTimestamp().IsZero() || autoMQ.GetDeletionTimestamp() == nil {
		controllerutil.AddFinalizer(autoMQ, autoMQFinalizer)
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.Update(ctx, autoMQ)
		}); err != nil {
			return ctrl.Result{}, err
		}
		return r.reconcile(ctx, autoMQ)
	}

	return ctrl.Result{}, errors.New("reconcile error from Finalizer")
}

func (r *AutoMQReconciler) reconcile(ctx context.Context, obj client.Object) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.V(1).Info("update reconcile controller automq", "request", client.ObjectKeyFromObject(obj))
	automq, ok := obj.(*infrav1beta1.AutoMQ)
	var err error
	if !ok {
		return ctrl.Result{}, errors.New("obj convert automq is error")
	}
	automq.Status.ControllerAddresses = r.controllerVoters(automq)
	pipelines := []func(ctx context.Context, mq *infrav1beta1.AutoMQ) context.Context{
		r.s3Service,
		r.scriptConfigmap,
		r.syncControllersScale,
		r.syncControllers,
	}

	for _, fn := range pipelines {
		ctx = fn(ctx, automq)
	}
	automq.Status.ControllerReplicas = automq.Spec.Controller.Replicas
	automq.Status.BrokerReplicas = automq.Spec.Broker.Replicas
	err = r.syncStatus(ctx, automq)
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *AutoMQReconciler) SetupWithManager(mgr ctrl.Manager, opts controller.RateLimiterOptions) error {
	if r.Client == nil {
		r.Client = mgr.GetClient()
	}
	r.Scheme = mgr.GetScheme()
	if r.Recorder == nil {
		r.Recorder = mgr.GetEventRecorderFor("automq-controller")
	}
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controllerlib.Options{
			MaxConcurrentReconciles: controller.GetConcurrent(opts),
			RateLimiter:             controller.GetRateLimiter(opts),
		}).
		For(&infrav1beta1.AutoMQ{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

func (r *AutoMQReconciler) syncStatus(ctx context.Context, automq *infrav1beta1.AutoMQ) error {
	log := log.FromContext(ctx)
	// Let's just set the status as Unknown when no status are available
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		original := &infrav1beta1.AutoMQ{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(automq), original); err != nil {
			return err
		}
		original.Status = *automq.Status.DeepCopy()
		return r.Client.Status().Update(ctx, original)
	}); err != nil {
		log.Error(err, "Failed to update automq status")
		return err
	}
	return nil
}

func (r *AutoMQReconciler) s3Service(ctx context.Context, obj *infrav1beta1.AutoMQ) context.Context {
	conditionType := "SyncS3ServiceReady"
	sg, err := storage.NewBucket(storage.Config{
		Type:     "s3",
		Key:      obj.Spec.S3.AccessKeyID,
		Secret:   obj.Spec.S3.SecretAccessKey,
		Region:   obj.Spec.S3.Region,
		Endpoint: obj.Spec.S3.Endpoint,
	})
	if err != nil {
		meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
			Type:               conditionType,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: obj.Generation,
			Reason:             "AwsS3ReconcilingInit",
			Message:            fmt.Sprintf("Failed to create S3 Bucket interface for the custom resource (%s): (%s)", obj.Name, err),
		})
		return ctx
	}
	_ = sg.MkBucket(ctx, obj.Spec.S3.Bucket)
	meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: obj.Generation,
		Reason:             "AwsS3Reconciling",
		Message:            fmt.Sprintf("S3 Bucket interface for the custom resource (%s) has been created", obj.Name),
	})
	return ctx
}

func (r *AutoMQReconciler) scriptConfigmap(ctx context.Context, obj *infrav1beta1.AutoMQ) context.Context {
	log := log.FromContext(ctx)
	conditionType := "SyncConfigmapReady"
	data, err := defaults.Asset("defaults/up.sh")
	if err != nil {
		meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
			Type:               conditionType,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: obj.Generation,
			Reason:             "ConfigmapReconcilingInit",
			Message:            fmt.Sprintf("Failed to create script configmap for the custom resource (%s): (%s)", obj.Name, err),
		})
		return ctx
	}
	if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var change controllerutil.OperationResult
		var e error
		cm := &v1.ConfigMap{}
		cm.Name = obj.Name
		cm.Namespace = obj.Namespace
		if change, e = controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
			cm.Data = map[string]string{
				"up.sh": string(data),
			}
			return nil
		}); e != nil {
			return e
		}
		log.V(1).Info("create or update configmap  by AutoMQ", "OperationResult", change)
		return nil
	}); err != nil {
		meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
			Type:               conditionType,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: obj.Generation,
			Reason:             "ConfigmapReconcilingCreate",
			Message:            fmt.Sprintf("Failed to create script configmap for the custom resource (%s): (%s)", obj.Name, err),
		})
		return ctx
	}

	meta.SetStatusCondition(&obj.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: obj.Generation,
		Reason:             "ConfigmapReconciling",
		Message:            fmt.Sprintf("Configmap script for the custom resource (%s) has been created", obj.Name),
	})
	return ctx
}

func getAutoMQLabelMap(name, role string) map[string]string {
	if role == "" {
		return map[string]string{
			"app.kubernetes.io/owner-by":  "automq",
			"app.kubernetes.io/component": "autpmq-operator",
			"app.kubernetes.io/instance":  name,
		}
	}
	return map[string]string{
		"app.kubernetes.io/owner-by":  "automq",
		"app.kubernetes.io/component": "autpmq-operator",
		"app.kubernetes.io/instance":  name,
		"app.kubernetes.io/role":      role,
	}
}

func getAutoMQName(role string, index *int32) string {
	if index != nil {
		return "automq-" + role + fmt.Sprintf("-%d", *index)
	}
	return "automq-" + role
}

const autoMQIndexKey = "app.kubernetes.io/index"
