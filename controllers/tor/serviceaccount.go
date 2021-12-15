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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	torv1alpha2 "example.com/null/tor-controller/apis/tor/v1alpha2"
)

func (r *OnionServiceReconciler) reconcileServiceAccount(ctx context.Context, onionService *torv1alpha2.OnionService) error {
	log := log.FromContext(ctx)

	serviceAccountName := onionService.ServiceAccountName()
	namespace := onionService.Namespace
	if serviceAccountName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(fmt.Errorf("serviceAccount name must be specified"))
		return nil
	}

	var serviceAccount corev1.ServiceAccount
	err := r.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: namespace}, &serviceAccount)

	newServiceAccount := torServiceAccount(onionService)
	if errors.IsNotFound(err) {
		err := r.Create(ctx, newServiceAccount)
		if err != nil {
			return err
		}
		serviceAccount = *newServiceAccount
	} else if err != nil {
		return err
	}

	if !metav1.IsControlledBy(&serviceAccount.ObjectMeta, onionService) {
		log.Info(fmt.Sprintf("ServiceAccount %s already exists and is not controller by %s", serviceAccount.Name, onionService.Name))
		return nil
	}

	// If the serviceAccount specs don't match, update
	if !serviceAccountEqual(&serviceAccount, newServiceAccount) {
		err := r.Update(ctx, newServiceAccount)
		if err != nil {
			return fmt.Errorf("Filed to update ServiceAccount %#v", newServiceAccount)
		}
	}

	return nil
}

func serviceAccountEqual(a, b *corev1.ServiceAccount) bool {
	// TODO: actually detect differences
	return true
}

func torServiceAccount(onion *torv1alpha2.OnionService) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      onion.ServiceAccountName(),
			Namespace: onion.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(onion, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    "OnionService",
				}),
			},
		},
	}
}
