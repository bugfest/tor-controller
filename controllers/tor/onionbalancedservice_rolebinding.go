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
	"strings"

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

func (r *OnionBalancedServiceReconciler) reconcileRolebinding(ctx context.Context, onionBalancedService *torv1alpha2.OnionBalancedService) error {
	logger := k8slog.FromContext(ctx)

	roleName := onionBalancedService.RoleName()
	namespace := onionBalancedService.Namespace

	if roleName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(errors.New("role name must be specified"))

		return nil
	}

	var roleBinding rbacv1.RoleBinding
	err := r.Get(ctx, types.NamespacedName{Name: roleName, Namespace: namespace}, &roleBinding)

	newRolebinding := onionbalanceRolebinding(onionBalancedService)
	if apierrors.IsNotFound(err) {
		err := r.Create(ctx, newRolebinding)
		if err != nil {
			return errors.Wrap(err, "failed to create Rolebinding")
		}

		roleBinding = *newRolebinding
	} else if err != nil {
		return errors.Wrap(err, "failed to get Rolebinding")
	}

	if !metav1.IsControlledBy(&roleBinding.ObjectMeta, onionBalancedService) {
		logger.Info("RoleBinding already exists and is not controlled by",
			"RoleBinding", roleBinding.Name,
			"controller", onionBalancedService.Name)

		return nil
	}

	// If the rolebinding specs don't match, update
	if !rolebindingEqual(&roleBinding, newRolebinding) {
		err := r.Update(ctx, newRolebinding)
		if err != nil {
			return errors.Wrapf(err, "filed to update Rolebinding %#v", newRolebinding)
		}
	}

	return nil
}

func onionbalanceRolebinding(onion *torv1alpha2.OnionBalancedService) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      onion.RoleName(),
			Namespace: onion.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(onion, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    strings.ToLower(onion.Kind),
				}),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: rbacv1.ServiceAccountKind,
				Name: onion.ServiceAccountName(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "Role",
			Name: onion.RoleName(),
		},
	}
}
