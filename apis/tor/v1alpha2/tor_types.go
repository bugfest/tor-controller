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

package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TorPodTemplate struct {
	// Metadata of the pods created from this template.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the behavior of a pod.
	// +optional
	Spec corev1.PodSpec `json:"spec,omitempty"`

	// Default resources for containers
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`
}

// TorSpec defines the desired state of Tor
type TorSpec struct {

	// Replicas.
	// +kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`

	// Template describes the pods that will be created.
	// +optional
	Template TorPodTemplate `json:"template,omitempty"`

	// Client type. Enabled by default if server options are not set.
	// +optional
	Client TorClientSpec `json:"client,omitempty"`

	// Server (ORPort)
	// +optional
	Server TorServerSpec `json:"server,omitempty"`

	// Control. Enabled by default.
	// +optional
	Control TorControlSpec `json:"control,omitempty"`

	// TODO: Secrets to be used as Control Port HashedControlPassword.
	// If not specified one will be created automatically.
	// +optional
	// ControlSecretRef []corev1.SecretKeySelector `json:"ControlSecretRef,omitempty"`

	// Metrics. Enabled by default.
	// +optional
	Metrics TorGenericPortWithFlagSpec `json:"metrics,omitempty"`

	// Create service monitor.
	// +optional
	// +kubebuilder:default:=false
	ServiceMonitor bool `json:"serviceMonitor,omitempty"`

	// Custom/advanced options.
	// Tor latest man page (asciidoc): https://gitlab.torproject.org/tpo/core/tor/-/blob/main/doc/man/tor.1.txt
	// +optional
	Config string `json:"config,omitempty"`

	// Custom/advanced options read from a ConfigMaps.
	// +optional
	ConfigMapKeyRef []corev1.ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`

	// Extra arguments to pass Tor's executable
	// +optional
	ExtraArgs []string `json:"extraArgs,omitempty"`
}

// TorStatus defines the observed state of Tor
type TorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Config string `json:"config,omitempty"`
}

// +kubebuilder:resource:shortName={"tor"}
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Tor is the Schema for the tor API
type Tor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TorSpec   `json:"spec,omitempty"`
	Status TorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TorList contains a list of Tor
type TorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tor{}, &TorList{})
}

type TorClientSpec struct {
	// DNSPort [address:]port|auto [isolation flags]
	// +optional
	DNS TorGenericPortWithFlagSpec `json:"dns,omitempty"`

	// NATDPort [address:]port|auto [isolation flags]
	// +optional
	NATD TorGenericPortWithFlagSpec `json:"natd,omitempty"`

	// SocksPort [address:]port|unix:path|auto [flags] [isolation flags]
	// +optional
	Socks TorGenericPortWithFlagSpec `json:"socks,omitempty"`

	// HTTPTunnelPort [address:]port|auto [isolation flags]
	// +optional
	HTTPTunnel TorGenericPortWithFlagSpec `json:"httptunnel,omitempty"`

	// TransPort [address:]port|auto [isolation flags]
	// +optional
	Trans TorGenericPortWithFlagSpec `json:"trans,omitempty"`

	// TransProxyType default|TPROXY|ipfw|pf-divert
	// +optional
	TransProxyType string `json:"transproxytype,omitempty"`
}

type TorGenericPortSpec struct {
	// +optional
	Enable bool `json:"enable,omitempty"`

	// +optional
	// +kubebuilder:default:=0
	Port int32 `json:"port,omitempty"`

	// +optional
	// +kubebuilder:default:="0.0.0.0"
	Address string `json:"address,omitempty"`
}

type TorGenericPortWithFlagSpec struct {
	TorGenericPortSpec `json:",inline"`
	Flags              []string `json:"flags,omitempty"`

	// Policy [address:]port|unix:path|auto [flags]
	// default: accept 0.0.0.0
	// +optional
	Policy []string `json:"policy,omitempty"`
}

type TorServerSpec struct {
	TorGenericPortWithFlagSpec `json:",inline"`
}

type TorControlSpec struct {
	TorGenericPortWithFlagSpec `json:",inline"`

	// Allowed control passwords as string
	// +optional
	Secret []string `json:"secret,omitempty"`

	// Allowed Control passwords as Secret object references
	// Reference to a key of a secret containing the hashed password
	// +optional
	SecretRef []corev1.SecretKeySelector `json:"secretRef,omitempty"`
}

// type TorMetricsSpec struct {
// 	TorGenericPortWithFlagSpec `json:",inline"`
// }

type TorGenericPortDef struct {
	Name     string
	Protocol string
	Port     TorGenericPortSpec
}
