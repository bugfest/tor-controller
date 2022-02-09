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
	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func onionServiceEqual(a, b *torv1alpha2.OnionService) bool {
	// TODO: actually detect differences
	return false
}

func deploymentEqual(a, b *appsv1.Deployment) bool {
	// TODO: actually detect differences
	return false
}

func serviceEqual(a, b *corev1.Service) bool {
	// TODO: actually detect differences
	return true
}

// func secretEqual(a, b *corev1.Service) bool {
// 	// TODO: actually detect differences
// 	return true
// }

func serviceAccountEqual(a, b *corev1.ServiceAccount) bool {
	// TODO: actually detect differences
	return true
}

func rolebindingEqual(a, b *rbacv1.RoleBinding) bool {
	// TODO: actually detect differences
	return true
}

func roleEqual(a, b *rbacv1.Role) bool {
	// TODO: actually detect differences
	return true
}
