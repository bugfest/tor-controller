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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	k8slog "sigs.k8s.io/controller-runtime/pkg/log"

	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
	"github.com/cockroachdb/errors"
)

func (r *TorReconciler) reconcileServiceAccount(ctx context.Context, tor *torv1alpha2.Tor) error {
	logger := k8slog.FromContext(ctx)

	serviceAccountName := tor.ServiceAccountName()
	namespace := tor.Namespace

	if serviceAccountName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(errors.New("serviceAccount name must be specified"))

		return nil
	}

	var serviceAccount corev1.ServiceAccount
	err := r.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: namespace}, &serviceAccount)

	newServiceAccount := torServiceAccount(tor)
	if apierrors.IsNotFound(err) {
		err := r.Create(ctx, newServiceAccount)
		if err != nil {
			return errors.Wrapf(err, "failed to create ServiceAccount %#v", newServiceAccount)
		}
		serviceAccount = *newServiceAccount
	} else if err != nil {
		return errors.Wrapf(err, "failed to get ServiceAccount %s", serviceAccountName)
	}

	if !metav1.IsControlledBy(&serviceAccount.ObjectMeta, tor) {
		logger.Info("ServiceAccount already exists and is not controlled by",
			"serviceAccount", serviceAccount.Name,
			"controller", tor.Name)

		return nil
	}

	// If the serviceAccount specs don't match, update
	if !serviceAccountEqual(&serviceAccount, newServiceAccount) {
		err := r.Update(ctx, newServiceAccount)
		if err != nil {
			return errors.Wrapf(err, "failed to update ServiceAccount %#v", newServiceAccount)
		}
	}

	return nil
}

func torServiceAccount(tor *torv1alpha2.Tor) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tor.ServiceAccountName(),
			Namespace: tor.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tor, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    "Tor",
				}),
			},
		},
	}
}
