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
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ServicePodTemplate struct {
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

// OnionServiceSpec defines the desired state of OnionService.
type OnionServiceSpec struct {
	// +patchMergeKey=port
	// +patchStrategy=merge
	Rules []ServiceRule `json:"rules,omitempty" pathchStrategy:"merge" patchMergeKey:"port"`

	// Template describes the pods that will be created.
	// +optional
	Template ServicePodTemplate `json:"template,omitempty"`

	// +optional
	PrivateKeySecret SecretReference `json:"privateKeySecret,omitempty"`

	// +optional
	AuthorizedClients []SecretReference `json:"authorizedClients,omitempty"`

	// +optional
	// +kubebuilder:validation:Enum=0;2;3
	// +kubebuilder:default:=3
	Version int32 `json:"version,omitempty"`

	// +optional
	MasterOnionAddress string `json:"masterOnionAddress,omitempty"`

	// +optional
	// +kubebuilder:default:=false
	ServiceMonitor bool `json:"serviceMonitor,omitempty"`

	// +optional
	ExtraConfig string `json:"extraConfig,omitempty"`
}

type ServiceRule struct {
	// Port publish as
	Port networkingv1.ServiceBackendPort `json:"port,omitempty"`

	// Backend selector
	Backend networkingv1.IngressBackend `json:"backend,omitempty"`
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
	// +optional
	Hostname string `json:"hostname,omitempty"`

	// +optional
	TargetClusterIP string `json:"targetClusterIp,omitempty"`
}

// +kubebuilder:resource:shortName={"onion","os"}
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Hostname",type=string,JSONPath=`.status.hostname`
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
