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
	"strings"
	"text/template"

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

{{- if .Tor.Spec.Client.DNS.Enable }}
# Client:DNS
{{- range $idx, $addr := .Tor.Spec.Client.DNS.Address }}
+DNSPort {{ $addr }}:{{ $.Tor.Spec.Client.DNS.Port }} {{ StringsJoin $.Tor.Spec.Client.DNS.Flags "," }}
{{- end }}
{{- end }}
{{- if .Tor.Spec.Client.NATD.Enable }}
# Client:NATD
{{- range $idx, $addr := .Tor.Spec.Client.NATD.Address }}
+NATDPort {{ $addr }}:{{ $.Tor.Spec.Client.NATD.Port }} {{ StringsJoin $.Tor.Spec.Client.NATD.Flags "," }}
{{- end }}
{{- end }}
{{- if .Tor.Spec.Client.HTTPTunnel.Enable }}
# Client:HTTPTunnel
{{- range $idx, $addr := .Tor.Spec.Client.HTTPTunnel.Address }}
+HTTPTunnelPort {{ $addr }}:{{ $.Tor.Spec.Client.HTTPTunnel.Port }} {{ StringsJoin $.Tor.Spec.Client.HTTPTunnel.Flags "," }}
{{- end }}
{{- end }}
{{- if .Tor.Spec.Client.Trans.Enable }}
# Client:Trans
{{- range $idx, $addr := .Tor.Spec.Client.Trans.Address }}
+TransPort {{ $addr }}:{{ $.Tor.Spec.Client.Trans.Port }} {{ StringsJoin $.Tor.Spec.Client.Trans.Flags "," }}
{{- end }}
+TransProxyType {{ .Tor.Spec.Client.TransProxyType }}
{{- end }}
{{- if .Tor.Spec.Client.Socks.Enable }}
# Client:Socks
{{- range $idx, $addr := .Tor.Spec.Client.Socks.Address }}
+SocksPort {{ $addr }}:{{ $.Tor.Spec.Client.Socks.Port }} {{ StringsJoin $.Tor.Spec.Client.Socks.Flags "," }}
{{- end }}
+SocksPolicy {{ StringsJoin .Tor.Spec.Client.Socks.Policy "," }}
{{- end }}

{{- if .Tor.Spec.Control.Enable }}
# Control
{{- range $idx, $addr := .Tor.Spec.Control.Address }}
+ControlPort {{ $addr }}:{{ $.Tor.Spec.Control.Port }} {{ StringsJoin $.Tor.Spec.Control.Flags "," }}
{{- end }}
{{- range .ControlHashedPasswords }}
+HashedControlPassword {{ . }}
{{- end }}
{{- end }}

{{- if .Tor.Spec.Metrics.Enable }}
# Metrics
{{- range $idx, $addr := .Tor.Spec.Metrics.Address }}
+MetricsPort {{ $addr }}:{{ $.Tor.Spec.Metrics.Port }} {{ StringsJoin $.Tor.Spec.Metrics.Flags "," }}
{{- end }}
+MetricsPortPolicy {{ StringsJoin .Tor.Spec.Metrics.Policy "," }}
{{- end }}

{{- if ne .Tor.Spec.Config "" }}
# Tor Custom config
{{ .Tor.Spec.Config }}
{{- end }}

{{- if ne (len .Tor.Spec.ConfigMapKeyRef) 0 }}
# Include Custom Configs mounted by ConfigMapKeyRef
%include /config/*/*.conf
{{- end }}
`

type torConfig struct {
	Tor                    *torv1alpha2.Tor
	ControlHashedPasswords []string
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
		log.Info(fmt.Sprintf("ConfigMap %s already exists and is not controller by %s", configmap.Name, Tor.Name))
		return nil
	}

	return nil
}

func torConfigFile(tor *torv1alpha2.Tor) string {

	s := torConfig{
		Tor:                    tor,
		ControlHashedPasswords: getTorControlHashedPasswords(tor),
	}

	var configTemplate = template.Must(
		template.New("config").Funcs(
			template.FuncMap{"StringsJoin": strings.Join},
		).Parse(torConfigFormat),
	)

	var tmp bytes.Buffer
	err := configTemplate.Execute(&tmp, s)
	if err != nil {
		return fmt.Sprintf("# error in template: %s", err)
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

func getTorControlHashedPasswords(tor *torv1alpha2.Tor) []string {
	hashes := []string{}

	for _, secret := range tor.Spec.Control.Secret {
		hash, err := doHashPassword(secret)
		if err == nil {
			hashes = append(hashes, hash)
		}
	}

	return hashes
}
