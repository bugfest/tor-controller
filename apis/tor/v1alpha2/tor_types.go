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

	// Image
	// +optional
	Image string `json:"image,omitempty"`

	// Client
	Client TorClientSpec `json:"client,inline,omitempty"`

	// Server
	// +optional
	Server TorServerSpec `json:"server,inline,omitempty"`

	// Control
	// +optional
	Control TorControlSpec `json:"control,inline,omitempty"`

	// Metrics
	// +optional
	Metrics TorMetricsSpec `json:"metrics,inline,omitempty"`

	// Other options
	// Tor latest man page (asciidoc): https://gitlab.torproject.org/tpo/core/tor/-/blob/main/doc/man/tor.1.txt
	// +optional
	Config string `json:"config,omitempty"`
}

// TorStatus defines the observed state of Tor
type TorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Config string `json:"config,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Tor is the Schema for the tor API
type Tor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"Metadata,omitempty"`

	Spec   TorSpec   `json:"Spec,omitempty"`
	Status TorStatus `json:"Status,omitempty"`
}

//+kubebuilder:object:root=true

// TorList contains a list of Tor
type TorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"Metadata,omitempty"`
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
	Enabled bool `json:"enabled,omitempty"`

	Port int `json:"port,omitempty"`

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
