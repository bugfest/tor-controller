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

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OnionBalancedServiceSpec defines the desired state of OnionBalancedService
type OnionBalancedServiceSpec struct {

	// "Tor onion service descriptors can include a maximum of 10 introduction points."
	// https://gitlab.torproject.org/tpo/core/onionbalance/-/blob/main/docs/v2/design.rst#L127
	// We set max to 8 to match onionbalance maximum allowed value

	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=8
	Replicas int32 `json:"replicas"`

	// +optional
	PrivateKeySecret SecretReference `json:"privateKeySecret,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum=3
	// +kubebuilder:default:=3
	Version int32 `json:"version"`

	// +optional
	Template TemplateReference `json:"template,omitempty"`
}

type TemplateReference struct {
	// +optional
	Spec OnionServiceSpec `json:"spec,omitempty"`
}

// OnionBalancedServiceStatus defines the observed state of OnionBalancedService
type OnionBalancedServiceStatus struct {
	// +optional
	Hostname string `json:"hostname,omitempty"`

	// +optional
	TargetClusterIP string `json:"targetClusterIP,omitempty"`

	// +optional
	Backends map[string]OnionServiceStatus `json:"backends,omitempty"`
}

// +kubebuilder:resource:shortName={"onionha","oha","obs"}
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Hostname",type=string,JSONPath=`.status.hostname`
// +kubebuilder:printcolumn:name="Replicas",type=string,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// OnionBalancedService is the Schema for the onionbalancedservices API
type OnionBalancedService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OnionBalancedServiceSpec   `json:"spec,omitempty"`
	Status OnionBalancedServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OnionBalancedServiceList contains a list of OnionBalancedService
type OnionBalancedServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OnionBalancedService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OnionBalancedService{}, &OnionBalancedServiceList{})
}
