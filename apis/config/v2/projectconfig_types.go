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

package v2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

// // ProjectConfigSpec defines the desired state of ProjectConfig
// type ProjectConfigSpec struct {
// }

// // ProjectConfigStatus defines the observed state of ProjectConfig
// type ProjectConfigStatus struct {
// }

// +kubebuilder:storageversion
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ProjectConfig is the Schema for the projectconfigs API
type ProjectConfig struct {
	metav1.TypeMeta `json:",inline"`

	// ControllerManagerConfigurationSpec returns the contfigurations for controllers
	cfg.ControllerManagerConfigurationSpec `json:",inline"`

	// +optional
	TorDaemonManager TorDaemonManagerType `json:"torDaemonManager,omitempty"`
}

type TorDaemonManagerType struct {
	// +optional
	// +kubebuilder:default:="quay.io/kragniz/tor-daemon-manager:master"
	Image string `json:"image,omitempty"`
}

// // +kubebuilder:object:root=true
// // ProjectConfigList contains a list of OnionService
// type ProjectConfigList struct {
// 	metav1.TypeMeta `json:",inline"`
// 	metav1.ListMeta `json:"metadata,omitempty"`
// 	Items           []ProjectConfig `json:"items"`
// }

func init() {
	SchemeBuilder.Register(&ProjectConfig{})
}
