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
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/runtime"
	k8slog "sigs.k8s.io/controller-runtime/pkg/log"

	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
	"github.com/cockroachdb/errors"
)

func (r *Reconciler) reconcileMetricsService(ctx context.Context, tor *torv1alpha2.Tor) error {
	logger := k8slog.FromContext(ctx)

	serviceName := tor.ServiceMetricsName()
	namespace := tor.Namespace

	if serviceName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(errors.New("service name must be specified"))

		return nil
	}

	var service corev1.Service
	err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, &service)

	newService := torMetricsService(tor)

	if apierrors.IsNotFound(err) {
		if !tor.Spec.Metrics.Enable {
			logger.Info("No metrics enabled, skipping metrics service for this tor instance")

			return nil
		}

		err := r.Create(ctx, newService)
		if err != nil {
			return errors.Wrapf(err, "failed to create Service %#v", newService)
		}

		service = *newService
	} else if err != nil {
		return errors.Wrapf(err, "failed to get Service %s", serviceName)
	}

	if !metav1.IsControlledBy(&service.ObjectMeta, tor) {
		logger.Info("Service already exists and is not controlled by",
			"service", service.Name,
			"controller", tor.Name)

		return nil
	}

	// If the service specs don't match, update
	if !serviceEqual(&service, newService) {
		err := r.Update(ctx, newService)
		if err != nil {
			return errors.Wrapf(err, "failed to update Service %#v", newService)
		}
	}

	return nil
}

func torMetricsService(onion *torv1alpha2.Tor) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      onion.ServiceMetricsName(),
			Namespace: onion.Namespace,
			Labels:    onion.ServiceMetricsSelector(),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(onion, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    "Tor",
				}),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: onion.ServiceSelector(),
			Ports: []corev1.ServicePort{{
				Name:       "metrics",
				TargetPort: intstr.FromInt(metricsPort),
				Port:       metricsPort,
			}},
		},
	}
}
