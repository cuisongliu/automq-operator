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
	infrav1beta1 "github.com/cuisongliu/automq-operator/api/v1beta1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *AutoMQReconciler) statusReconcile(ctx context.Context, obj client.Object) error {
	log := log.FromContext(ctx)
	log.V(1).Info("update reconcile status controller automq", "request", client.ObjectKeyFromObject(obj))
	automq := &infrav1beta1.AutoMQ{}
	err := r.Get(ctx, client.ObjectKeyFromObject(obj), automq)
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	automq.Status.Phase = infrav1beta1.AutoMQPending
	// Let's just set the status as Unknown when no status are available

	if err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
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
