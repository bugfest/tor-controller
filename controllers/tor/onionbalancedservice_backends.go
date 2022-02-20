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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	configv2 "github.com/bugfest/tor-controller/apis/config/v2"
	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
)

func (r *OnionBalancedServiceReconciler) reconcileBackends(ctx context.Context, OnionBalancedService *torv1alpha2.OnionBalancedService) error {
	log := log.FromContext(ctx)

	// Reconcile each backend
	for idx := int32(1); idx <= OnionBalancedService.Spec.Backends; idx++ {
		_, err := r.reconcileBackend(ctx, OnionBalancedService, idx)
		if err != nil {
			log.Error(err, fmt.Sprintf("unable reconcile backend idx=%d", idx))
		}
	}

	return nil
}

func (r *OnionBalancedServiceReconciler) reconcileBackend(ctx context.Context, OnionBalancedService *torv1alpha2.OnionBalancedService, idx int32) (*torv1alpha2.OnionService, error) {
	// log := log.FromContext(ctx)

	onionServiceName := OnionBalancedService.OnionServiceBackendName(idx)
	namespace := OnionBalancedService.Namespace
	if onionServiceName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(fmt.Errorf("%s/%s: onionService name must be specified", OnionBalancedService.Namespace, OnionBalancedService.Name))
		return nil, nil
	}

	var onionServiceBackend torv1alpha2.OnionService
	err := r.Get(ctx, types.NamespacedName{Name: onionServiceName, Namespace: namespace}, &onionServiceBackend)

	// We need a master address
	if len(OnionBalancedService.Status.Hostname) == 0 {
		return nil, fmt.Errorf("OnionBalancedService Hostname is not set")
	}

	// If the onionService doesn't exist, we'll create it
	projectConfig := r.ProjectConfig
	newOnionServiceBackend := onionBalancedServiceBackend(OnionBalancedService, projectConfig, idx)
	if apierrors.IsNotFound(err) {
		err := r.Create(ctx, newOnionServiceBackend)
		if err != nil {
			return nil, err
		}
		onionServiceBackend = *newOnionServiceBackend
	} else if err != nil {
		// If an error occurs during Get/Create, we'll requeue the item so we can
		// attempt processing again later. This could have been caused by a
		// temporary network failure, or any other transient reason.
		return nil, err
	}
	return &onionServiceBackend, nil
}

func onionBalancedServiceBackend(onion *torv1alpha2.OnionBalancedService, projectConfig configv2.ProjectConfig, idx int32) *torv1alpha2.OnionService {
	return &torv1alpha2.OnionService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      onion.OnionServiceBackendName(idx),
			Namespace: onion.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(onion, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    "OnionBalancedService",
				}),
			},
		},
		Spec: torv1alpha2.OnionServiceSpec{
			Rules:              onion.Spec.Template.Spec.Rules,
			Version:            onion.Spec.Version,
			MasterOnionAddress: onion.Status.Hostname,
			ServiceMonitor:     onion.Spec.Template.Spec.ServiceMonitor,
		},
	}
}
