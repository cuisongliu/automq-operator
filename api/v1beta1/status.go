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

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionType string

type StatusCondition struct {
	// Type is the type of the condition.
	Type ConditionType `json:"type"`
	// Status is the status of the condition. One of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// LastTransitionTime is the last time the condition changed from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Reason is a (brief) reason for the condition's last status change.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Message is a human-readable message indicating details about the last status change.
	// +optional
	Message string `json:"message,omitempty"`
}

// SetCondition sets a condition on the status object. If the condition already
// exists, it will be replaced. SetCondition does not update the resource in
// the cluster.
func SetCondition(conditions []StatusCondition, condition StatusCondition) []StatusCondition {
	now := metav1.Now()
	for i := range conditions {
		if conditions[i].Type == condition.Type {
			if conditions[i].Status != condition.Status {
				condition.LastTransitionTime = now
			} else {
				condition.LastTransitionTime = conditions[i].LastTransitionTime
			}
			conditions[i] = condition
			return conditions
		}
	}

	// If the condition does not exist,
	// initialize the lastTransitionTime
	condition.LastTransitionTime = now
	conditions = append(conditions, condition)
	return conditions
}

// RemoveCondition removes the condition with the passed condition type from
// the status object. If the condition is not already present, the returned
// status object is returned unchanged. RemoveCondition does not update the
// resource in the cluster.
func RemoveCondition(conditions []StatusCondition, conditionType ConditionType) []StatusCondition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			conditions = append(conditions[:i], conditions[i+1:]...)
			return conditions
		}
	}
	return conditions
}
