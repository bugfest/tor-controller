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

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OnionBalancedServiceSpec defines the desired state of OnionBalancedService
type OnionBalancedServiceSpec struct {

	// "Tor onion service descriptors can include a maximum of 10 introduction points."
	// https://gitlab.torproject.org/tpo/core/onionbalance/-/blob/main/docs/v2/design.rst#L127
	// We set max to 8 to match onionbalance maximum allowed value

	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=8
	Backends int32 `json:"backends"`

	// +optional
	PrivateKeySecret SecretReference `json:"privateKeySecret,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum=3
	// +kubebuilder:default:=3
	Version int32 `json:"version"`

	// +optional
	// +kubebuilder:default:=false
	ServiceMonitor bool `json:"serviceMonitor,omitempty"`

	// +optional
	Template TemplateReference `json:"template,omitempty"`

	// Template describes the balancer daemon pods that will be created.
	// +optional
	BalancerTemplate BalancerTemplate `json:"balancerTemplate,omitempty"`
}

type TemplateReference struct {
	// +optional
	Spec OnionServiceSpec `json:"spec,omitempty"`
}

// Template for the daemon pods
type BalancerTemplate struct {
	// Metadata of the pods created from this template.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the behavior of a pod.
	// +optional
	Spec corev1.PodSpec `json:"spec,omitempty"`

	// Default resources for tor containers
	// +optional
	TorResources corev1.ResourceRequirements `json:"torResources,omitempty" protobuf:"bytes,8,opt,name=resources"`

	// Default resources for onionbalance containers
	// +optional
	BalancerResources corev1.ResourceRequirements `json:"balancerResources,omitempty" protobuf:"bytes,8,opt,name=resources"`
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
// +kubebuilder:printcolumn:name="Backends",type=string,JSONPath=`.spec.backends`
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
