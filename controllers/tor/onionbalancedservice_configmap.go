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
	"bytes"
	"context"
	"fmt"
	"html/template"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
)

func (r *OnionBalancedServiceReconciler) reconcileConfigMap(ctx context.Context, OnionBalancedService *torv1alpha2.OnionBalancedService) error {
	log := log.FromContext(ctx)

	configMapName := OnionBalancedService.ConfigMapName()
	namespace := OnionBalancedService.Namespace
	if configMapName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(fmt.Errorf("configMap name must be specified"))
		return nil
	}

	var configmap corev1.ConfigMap
	err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: namespace}, &configmap)

	newConfigMap := onionbalanceTorConfigMap(OnionBalancedService)
	if errors.IsNotFound(err) {
		err := r.Create(ctx, newConfigMap)
		if err != nil {
			return err
		}
		configmap = *newConfigMap
	} else if err != nil {
		return err
	}

	if !metav1.IsControlledBy(&configmap.ObjectMeta, OnionBalancedService) {
		// msg := fmt.Sprintf("Secret %s already exists and is not controller by %s", secret.Name, OnionBalancedService.Name)
		// TODO: generate MessageResourceExists event
		// msg := fmt.Sprintf(MessageResourceExists, service.Name)
		// bc.recorder.Event(OnionBalancedService, corev1.EventTypeWarning, ErrResourceExists, msg)
		// return fmt.Errorf(msg)
		log.Info(fmt.Sprintf("Secret %s already exists and is not controller by %s", configmap.Name, OnionBalancedService.Name))
		return nil
	}

	return nil
}

func onionbalanceTorConfig(onion *torv1alpha2.OnionBalancedService) string {
	const configFormat = `# Config automatically generated
SocksPort 0
ControlPort 127.0.0.1:6666
`

	var configTemplate = template.Must(template.New("config").Parse(configFormat))
	var tmp bytes.Buffer
	err := configTemplate.Execute(&tmp, onion)
	if err != nil {
		return ""
	}
	return tmp.String()
}

func onionbalanceTorConfigMap(onion *torv1alpha2.OnionBalancedService) *corev1.ConfigMap {

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      onion.ConfigMapName(),
			Namespace: onion.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(onion, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    "onionbalancedservice",
				}),
			},
		},
		Data: map[string]string{
			"torfile": onionbalanceTorConfig(onion),
		},
	}
}
