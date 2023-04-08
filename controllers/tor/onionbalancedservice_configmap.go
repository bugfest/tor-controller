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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	k8slog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cockroachdb/errors"

	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
)

const configFormat = `# Config automatically generated
SocksPort {{ .SocksPort }}
ControlPort {{ .ControlPort }}
MetricsPort {{ .MetricsPort }}
MetricsPortPolicy {{ .MetricsPortPolicy }}
`

type onionBalancedServiceTorConfig struct {
	SocksPort         string
	ControlPort       string
	MetricsPort       string
	MetricsPortPolicy string
}

func (r *OnionBalancedServiceReconciler) reconcileConfigMap(
	ctx context.Context, onionBalancedService *torv1alpha2.OnionBalancedService,
) error {
	logger := k8slog.FromContext(ctx)

	configMapName := onionBalancedService.ConfigMapName()
	namespace := onionBalancedService.Namespace

	if configMapName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(errors.New("configMap name must be specified"))

		return nil
	}

	var configmap corev1.ConfigMap
	err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: namespace}, &configmap)

	newConfigMap := onionbalanceTorConfigMap(onionBalancedService)
	if apierrors.IsNotFound(err) {
		err := r.Create(ctx, newConfigMap)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to create configmap %s", configMapName))
		}

		configmap = *newConfigMap
	} else if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to get configmap %s", configMapName))
	}

	if !metav1.IsControlledBy(&configmap.ObjectMeta, onionBalancedService) {
		logger.Info("configmap already exists and is not controlled by onionbalancedservice",
			"configmap", configmap.Name,
			"onionbalancedservice", onionBalancedService.Name,
		)

		return nil
	}

	return nil
}

func onionbalanceTorConfig(_ *torv1alpha2.OnionBalancedService) string {
	serviceConfig := onionBalancedServiceTorConfig{
		SocksPort:         "0",
		ControlPort:       "127.0.0.1:9051",
		MetricsPort:       "0.0.0.0:9035",
		MetricsPortPolicy: "accept 0.0.0.0/0",
	}

	configTemplate := template.Must(template.New("config").Parse(configFormat))

	var tmp bytes.Buffer

	err := configTemplate.Execute(&tmp, serviceConfig)
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
