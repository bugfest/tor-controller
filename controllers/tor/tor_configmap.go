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

const torConfigFormat = `# Config automatically generated
# {{ .Tor.Namespace }}/{{ .Tor.Name }}
SocksPort {{ .SocksPort }}
ControlPort {{ .ControlPort }}
MetricsPort {{ .MetricsPort }}
MetricsPortPolicy {{ .MetricsPortPolicy }}

# Tor Custom config goes here
{{ .Tor.Spec.Config }}
`

type torConfig struct {
	Tor               *torv1alpha2.Tor
	SocksPort         string
	ControlPort       string
	MetricsPort       string
	MetricsPortPolicy string
}

func (r *TorReconciler) reconcileConfigMap(ctx context.Context, Tor *torv1alpha2.Tor) error {
	log := log.FromContext(ctx)

	configMapName := Tor.ConfigMapName()
	namespace := Tor.Namespace
	if configMapName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(fmt.Errorf("configMap name must be specified"))
		return nil
	}

	var configmap corev1.ConfigMap
	err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: namespace}, &configmap)

	newConfigMap := torConfigMap(Tor)
	if errors.IsNotFound(err) {
		err := r.Create(ctx, newConfigMap)
		if err != nil {
			return err
		}
		configmap = *newConfigMap
	} else if err != nil {
		return err
	}

	if !metav1.IsControlledBy(&configmap.ObjectMeta, Tor) {
		// msg := fmt.Sprintf("Secret %s already exists and is not controller by %s", secret.Name, Tor.Name)
		// TODO: generate MessageResourceExists event
		// msg := fmt.Sprintf(MessageResourceExists, service.Name)
		// bc.recorder.Event(Tor, corev1.EventTypeWarning, ErrResourceExists, msg)
		// return fmt.Errorf(msg)
		log.Info(fmt.Sprintf("Secret %s already exists and is not controller by %s", configmap.Name, Tor.Name))
		return nil
	}

	return nil
}

func torConfigFile(tor *torv1alpha2.Tor) string {

	s := torConfig{
		Tor:               tor,
		SocksPort:         "0.0.0.0:9050",
		ControlPort:       "0.0.0.0:9051",
		MetricsPort:       "0.0.0.0:9035",
		MetricsPortPolicy: "accept 0.0.0.0/0",
	}

	var configTemplate = template.Must(template.New("config").Parse(torConfigFormat))
	var tmp bytes.Buffer
	err := configTemplate.Execute(&tmp, s)
	if err != nil {
		return ""
	}
	return tmp.String()
}

func torConfigMap(tor *torv1alpha2.Tor) *corev1.ConfigMap {

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tor.ConfigMapName(),
			Namespace: tor.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tor, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    "Tor",
				}),
			},
		},
		Data: map[string]string{
			"torfile": torConfigFile(tor),
		},
	}
}
