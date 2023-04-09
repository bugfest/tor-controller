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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OnionServiceSpec defines the desired state of OnionService.
type OnionServiceSpec struct {
	// The list of ports that are exposed by this service.
	// +patchMergeKey=publicPort
	// +patchStrategy=merge
	Ports []ServicePort `json:"ports,omitempty" patchStrategy:"merge" patchMergeKey:"publicPort"`

	Selector map[string]string `json:"selector,omitempty"`

	// +optional
	PrivateKeySecret SecretReference `json:"privateKeySecret,omitempty"`

	// +kubebuilder:validation:Enum=0;2;3
	Version int32 `json:"version"`

	// +optional
	ExtraConfig string `json:"extraConfig,omitempty"`
}

type ServicePort struct {
	// Optional if only one ServicePort is defined on this service.
	// +optional
	Name string `json:"name,omitempty"`

	// The port that will be exposed by this service.
	PublicPort int32 `json:"publicPort"`
	// Number or name of the port to access on the pods targeted by the service.
	// Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
	// If this is a string, it will be looked up as a named port in the
	// target Pod's container ports. If this is not specified, the value
	// of the 'port' field is used (an identity map).
	// This field is ignored for services with clusterIP=None, and should be
	// omitted or set equal to the 'port' field.
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service
	// +optional
	TargetPort int32 `json:"targetPort,omitempty"`
}

// SecretReference represents a Secret Reference.
type SecretReference struct {
	// Name is unique within a namespace to reference a secret resource.
	Name string `json:"name,omitempty"`

	Key string `json:"key,omitempty"`
}

// OnionServiceStatus defines the observed state of OnionService.
type OnionServiceStatus struct {
	Hostname        string `json:"hostname"`
	TargetClusterIP string `json:"targetClusterIp"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Hostname",type=string,JSONPath=`.status.hostname`
// +kubebuilder:printcolumn:name="TargetClusterIP",type=string,JSONPath=`.status.targetClusterIP`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// OnionService is the Schema for the onionservices API.
type OnionService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OnionServiceSpec   `json:"spec,omitempty"`
	Status OnionServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OnionServiceList contains a list of OnionService.
type OnionServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OnionService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OnionService{}, &OnionServiceList{})
}
