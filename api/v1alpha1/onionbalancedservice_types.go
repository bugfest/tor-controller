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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OnionBalancedServiceSpec defines the desired state of OnionBalancedService
type OnionBalancedServiceSpec struct {
	// +optional
	PrivateKeySecret SecretReference `json:"privateKeySecret,omitempty"`

	// +kubebuilder:validation:Enum=3
	Version int32 `json:"version"`

	// +kubebuilder:validation:MaxItems=64
	// +kubebuilder:validation:MinItems=0
	Selector []string `json:"selector,omitempty"`
}

// OnionBalancedServiceStatus defines the observed state of OnionBalancedService
type OnionBalancedServiceStatus struct {
	Hostname string            `json:"hostname"`
	Backends map[string]string `json:"backends"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Hostname",type=string,JSONPath=`.status.hostname`
// +kubebuilder:printcolumn:name="Backends",type=string,JSONPath=`.status.backends`
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
