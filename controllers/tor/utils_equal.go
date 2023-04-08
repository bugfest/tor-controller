/*
Copyright 2021.

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

package tor

import (
	"reflect"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// deploymentEqual compares two deployments and returns true if they are equal.
func deploymentEqual(dep1, dep2 *appsv1.Deployment) bool {
	// Compare metadata
	if !reflect.DeepEqual(dep1.ObjectMeta, dep2.ObjectMeta) {
		return false
	}

	// Compare spec
	if !reflect.DeepEqual(dep1.Spec, dep2.Spec) {
		return false
	}

	// Compare status
	if !reflect.DeepEqual(dep1.Status, dep2.Status) {
		return false
	}

	return true
}

func serviceEqual(svc1, svc2 *corev1.Service) bool {
	// Compare metadata
	if !reflect.DeepEqual(svc1.ObjectMeta, svc2.ObjectMeta) {
		return false
	}

	// Compare spec
	if !reflect.DeepEqual(svc1.Spec, svc2.Spec) {
		return false
	}

	// Compare status
	if !reflect.DeepEqual(svc1.Status, svc2.Status) {
		return false
	}

	return true
}

func monitorServiceEqual(sm1, sm2 *monitoringv1.ServiceMonitor) bool {
	// Compare metadata
	if !reflect.DeepEqual(sm1.ObjectMeta, sm2.ObjectMeta) {
		return false
	}

	// Compare spec
	if !reflect.DeepEqual(sm1.Spec, sm2.Spec) {
		return false
	}

	return true
}

func serviceAccountEqual(sa1, sa2 *corev1.ServiceAccount) bool {
	// Compare metadata
	if !reflect.DeepEqual(sa1.ObjectMeta, sa2.ObjectMeta) {
		return false
	}

	// Compare secrets
	if !reflect.DeepEqual(sa1.Secrets, sa2.Secrets) {
		return false
	}

	// Compare image pull secrets
	if !reflect.DeepEqual(sa1.ImagePullSecrets, sa2.ImagePullSecrets) {
		return false
	}

	// Compare automount service account token
	if !reflect.DeepEqual(sa1.AutomountServiceAccountToken, sa2.AutomountServiceAccountToken) {
		return false
	}

	return true
}

func rolebindingEqual(rb1, rb2 *rbacv1.RoleBinding) bool {
	// Compare metadata
	if !reflect.DeepEqual(rb1.ObjectMeta, rb2.ObjectMeta) {
		return false
	}

	// Compare subjects
	if !reflect.DeepEqual(rb1.Subjects, rb2.Subjects) {
		return false
	}

	// Compare roleRef
	if !reflect.DeepEqual(rb1.RoleRef, rb2.RoleRef) {
		return false
	}

	return true
}

func roleEqual(role1, role2 *rbacv1.Role) bool {
	// Compare metadata
	if !reflect.DeepEqual(role1.ObjectMeta, role2.ObjectMeta) {
		return false
	}

	// Compare rules
	if !reflect.DeepEqual(role1.Rules, role2.Rules) {
		return false
	}

	return true
}
