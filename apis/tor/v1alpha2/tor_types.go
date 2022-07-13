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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TorSpec defines the desired state of Tor
type TorSpec struct {

	// Replicas
	// +kubebuilder:default:=1
	Replicas int32 `json:"replicas,omitempty"`

	// Client
	Client TorClientSpec `json:"client,omitempty"`

	// Server
	// +optional
	Server TorServerSpec `json:"server,omitempty"`

	// Control
	// +optional
	Control TorControlSpec `json:"control,omitempty"`

	// Metrics
	// +optional
	Metrics TorMetricsSpec `json:"metrics,omitempty"`

	// +optional
	// +kubebuilder:default:=false
	ServiceMonitor bool `json:"serviceMonitor,omitempty"`

	// Other options
	// Tor latest man page (asciidoc): https://gitlab.torproject.org/tpo/core/tor/-/blob/main/doc/man/tor.1.txt
	// +optional
	Config string `json:"config,omitempty"`

	// +optional
	Args []string `json:"args,omitempty"`
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
	HTTPtunnel TorGenericPortWithFlagSpec `json:"httptunnel,omitempty"`

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
}

type TorServerSpec struct {
	TorGenericPortWithFlagSpec `json:",inline"`
}

type TorControlSpec struct {
	TorGenericPortSpec `json:",inline"`
}

type TorMetricsSpec struct {
	TorGenericPortWithFlagSpec `json:",inline"`

	// MetricsPortPolicy [address:]port|unix:path|auto [flags]
	// +optional
	// +kubebuilder:default:="accept 0.0.0.0/0"
	Policy string `json:"policy,omitempty"`
}

type TorGenericPortDef struct {
	Name     string
	Protocol string
	Port     TorGenericPortSpec
}
