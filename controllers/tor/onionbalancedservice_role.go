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
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	k8slog "sigs.k8s.io/controller-runtime/pkg/log"

	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
	"github.com/cockroachdb/errors"
)

func (r *OnionBalancedServiceReconciler) reconcileRole(ctx context.Context, onionBalancedService *torv1alpha2.OnionBalancedService) error {
	log := k8slog.FromContext(ctx)

	namespace := onionBalancedService.Namespace

	roleName := onionBalancedService.RoleName()
	if roleName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(errors.New("role name must be specified"))

		return nil
	}

	var role rbacv1.Role
	err := r.Get(ctx, types.NamespacedName{Name: roleName, Namespace: namespace}, &role)

	newRole := onionbalanceRole(onionBalancedService)
	if apierrors.IsNotFound(err) {
		err := r.Create(ctx, newRole)
		if err != nil {
			return errors.Wrapf(err, "failed to create Role %#v", newRole)
		}

		role = *newRole
	} else if err != nil {
		return errors.Wrapf(err, "failed to get Role %s", roleName)
	}

	if !metav1.IsControlledBy(&role.ObjectMeta, onionBalancedService) {
		log.Info(fmt.Sprintf("Role %s already exists and is not controlled by %s", role.Name, onionBalancedService.Name))

		return nil
	}

	// If the role specs don't match, update
	if !roleEqual(&role, newRole) {
		err := r.Update(ctx, newRole)
		if err != nil {
			return errors.Wrapf(err, "failed to update Role %#v", newRole)
		}
	}

	return nil
}

func onionbalanceRole(onion *torv1alpha2.OnionBalancedService) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      onion.RoleName(),
			Namespace: onion.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(onion, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    onion.Kind,
				}),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{torv1alpha2.GroupVersion.Group},
				Verbs:     []string{"get", "list", "watch"},
				Resources: []string{
					"onionbalancedservices",
				},
			},
			{
				APIGroups: []string{torv1alpha2.GroupVersion.Group},
				Verbs:     []string{"update", "patch"},
				Resources: []string{
					"onionbalancedservices/status",
				},
			},
			{
				APIGroups: []string{""},
				Verbs:     []string{"create", "update", "patch"},
				Resources: []string{"events"},
			},
		},
	}
}
