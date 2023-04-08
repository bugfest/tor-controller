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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	k8slog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cockroachdb/errors"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
)

func (r *OnionBalancedServiceReconciler) reconcileServiceMonitor(
	ctx context.Context,
	onionBalancedService *torv1alpha2.OnionBalancedService,
) error {
	logger := k8slog.FromContext(ctx)

	if !r.monitoringInstalled(ctx) {
		// Service Monitor cannot be created; monitoring CRDs are not installed
		return nil
	}

	serviceName := onionBalancedService.ServiceMetricsName()
	namespace := onionBalancedService.Namespace

	if serviceName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(errors.New("service monitor name must be specified"))

		return nil
	}

	var service monitoringv1.ServiceMonitor
	err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, &service)

	newService := obsTorServiceMonitor(onionBalancedService)

	if apierrors.IsNotFound(err) {
		if !onionBalancedService.Spec.ServiceMonitor {
			// ServiceMonitor is not requested, skipping
			return nil
		}

		err := r.Create(ctx, newService)
		if err != nil {
			return errors.Wrapf(err, "failed to create ServiceMonitor %s", newService.Name)
		}

		service = *newService
	} else if err != nil {
		return errors.Wrapf(err, "failed to get ServiceMonitor %s", service.Name)
	}

	if !metav1.IsControlledBy(&service.ObjectMeta, onionBalancedService) {
		logger.Info("ServiceMonitor already exists and is not controlled by",
			"ServiceMonitor", service.Name,
			"controller", onionBalancedService.Name)

		return nil
	}

	if !onionBalancedService.Spec.ServiceMonitor {
		// ServiceMonitor is not requested but exists, deleting
		err = r.Delete(ctx, &service)
		if err != nil {
			return errors.Wrapf(err, "failed to delete ServiceMonitor %s", service.Name)
		}

		return nil
	}

	// If the service specs don't match, update
	if !monitorServiceEqual(&service, newService) {
		err := r.Update(ctx, newService)
		if err != nil {
			return errors.Wrapf(err, "failed to update ServiceMonitor %s", service.Name)
		}
	}

	return nil
}

// It requires fix for "metrics: Prometheus output needs to quote the label's value"
// (tor-0.4.6.10) https://gitlab.torproject.org/tpo/core/tor/-/issues/40552
func obsTorServiceMonitor(onion *torv1alpha2.OnionBalancedService) *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      onion.ServiceMetricsName(),
			Namespace: onion.Namespace,
			Labels:    onion.ServiceMetricsSelector(),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(onion, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    "OnionBalancedService",
				}),
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: onion.ServiceMetricsSelector(),
			},
			Endpoints: []monitoringv1.Endpoint{
				{
					Port: "metrics",
					Path: "/metrics",
				},
			},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{
					onion.Namespace,
				},
			},
		},
	}
}

func (r *OnionBalancedServiceReconciler) monitoringInstalled(ctx context.Context) bool {
	var monitoring apiextensionsv1.CustomResourceDefinition
	err := r.Get(ctx, types.NamespacedName{Name: "servicemonitors.monitoring.coreos.com", Namespace: "default"}, &monitoring)
	// if err != nil {
	// 	log := k8slog.FromContext(ctx)
	// 	log.Error(err, "error at monitoringInstalled")
	// }
	return !apierrors.IsNotFound(err)
}
