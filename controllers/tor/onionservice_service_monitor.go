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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
)

func (r *OnionServiceReconciler) reconcileServiceMonitor(ctx context.Context, onionService *torv1alpha2.OnionService) error {
	log := log.FromContext(ctx)

	if !r.monitoringInstalled(ctx) {
		// Service Monitor cannot be created; monitoring CRDs are not installed
		return nil
	}

	serviceName := onionService.ServiceMetricsName()
	namespace := onionService.Namespace
	if serviceName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(fmt.Errorf("service monitor name must be specified"))
		return nil
	}

	var service monitoringv1.ServiceMonitor
	err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, &service)

	newService := os_torServiceMonitor(onionService)
	if errors.IsNotFound(err) {

		if !onionService.Spec.ServiceMonitor {
			// ServiceMonitor is not requested, skipping
			return nil
		}

		err := r.Create(ctx, newService)
		if err != nil {
			return err
		}
		service = *newService
	} else if err != nil {
		return err
	}

	if !metav1.IsControlledBy(&service.ObjectMeta, onionService) {
		log.Info(fmt.Sprintf("ServiceMonitor %s already exists and is not controller by %s", service.Name, onionService.Name))
		return nil
	}

	if !onionService.Spec.ServiceMonitor {
		// ServiceMonitor is not requested but exists, deleting
		err = r.Delete(ctx, &service)
		if err != nil {
			return fmt.Errorf("failed to delete Service %#v", service)
		}
		return nil
	}

	// If the service specs don't match, update
	if !monitorServiceEqual(&service, newService) {
		err := r.Update(ctx, newService)
		if err != nil {
			return fmt.Errorf("failed to update Service %#v", newService)
		}
	}

	return nil
}

// It requires fix for "metrics: Prometheus output needs to quote the label's value"
// (tor-0.4.6.10) https://gitlab.torproject.org/tpo/core/tor/-/issues/40552
func os_torServiceMonitor(onion *torv1alpha2.OnionService) *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      onion.ServiceMetricsName(),
			Namespace: onion.Namespace,
			Labels:    onion.ServiceMetricsSelector(),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(onion, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    "OnionService",
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

func (r *OnionServiceReconciler) monitoringInstalled(ctx context.Context) bool {
	var monitoring apiextensionsv1.CustomResourceDefinition
	err := r.Get(ctx, types.NamespacedName{Name: "servicemonitors.monitoring.coreos.com", Namespace: "default"}, &monitoring)
	// if err != nil {
	// 	log := log.FromContext(ctx)
	// 	log.Error(err, "error at monitoringInstalled")
	// }
	return !errors.IsNotFound(err)
}
