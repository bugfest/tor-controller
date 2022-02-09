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
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	configv2 "github.com/bugfest/tor-controller/apis/config/v2"
	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
)

func (r *OnionBalancedServiceReconciler) reconcileDeployment(ctx context.Context, OnionBalancedService *torv1alpha2.OnionBalancedService) error {
	log := log.FromContext(ctx)

	deploymentName := OnionBalancedService.DeploymentName()
	namespace := OnionBalancedService.Namespace
	if deploymentName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(fmt.Errorf("%s/%s: deployment name must be specified", OnionBalancedService.Namespace, OnionBalancedService.Name))
		return nil
	}

	var deployment appsv1.Deployment
	err := r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespace}, &deployment)

	// If the deployment doesn't exist, we'll create it
	projectConfig := r.ProjectConfig
	newDeployment := onionbalanceDeployment(OnionBalancedService, projectConfig)

	// log.Info(fmt.Sprintf(" %#v", *newDeployment))

	if apierrors.IsNotFound(err) {
		err := r.Create(ctx, newDeployment)
		if err != nil {
			return err
		}
		deployment = *newDeployment
	} else if err != nil {
		// If an error occurs during Get/Create, we'll requeue the item so we can
		// attempt processing again later. This could have been caused by a
		// temporary network failure, or any other transient reason.
		return err
	}

	// If the Deployment is not controlled by this Foo resource, we should log
	// a warning to the event recorder and ret
	if !metav1.IsControlledBy(&deployment.ObjectMeta, OnionBalancedService) {
		log.Info(fmt.Sprintf("Deployment %s already exists and not controlled by %s - skipping update", deployment.Name, OnionBalancedService.Name))
		return nil
	}

	// If the deployment specs don't match, update
	if !deploymentEqual(&deployment, newDeployment) {
		err := r.Update(ctx, newDeployment)
		if err != nil {
			return fmt.Errorf("filed to update Deployment %#v", newDeployment)
		}
	}

	return nil
}

func onionbalanceDeployment(onion *torv1alpha2.OnionBalancedService, projectConfig configv2.ProjectConfig) *appsv1.Deployment {

	onionBalanceConfigMountPath := "/run/onionbalance/"
	onionBalanceSecretMountPath := "/run/onionbalance/key"

	torConfigMountDir := "/run/tor"
	privateKeyMounPath := "/run/tor/key"

	volumes := []corev1.Volume{
		{
			Name: onionBalanceConfigVolume,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: torConfigVolume,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: onion.ConfigMapName(),
					},
					Items: []corev1.KeyToPath{{
						Key:  "torfile",
						Path: "torfile",
					}},
				},
			},
		},
		{
			Name: onionBalanceSecretVolume,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: onion.SecretName(),
				},
			},
		},
		{
			Name: privateKeyVolume,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: onion.SecretName(),
					Items: []corev1.KeyToPath{
						{
							Key:  "privateKeyFile",
							Path: "private_key",
						},
						{
							Key:  "onionAddress",
							Path: "hostname",
						},
					},
				},
			},
		},
	}

	onionBalanceVolumeMounts := []corev1.VolumeMount{
		{
			Name:      onionBalanceConfigVolume,
			MountPath: onionBalanceConfigMountPath,
		},
		{
			Name:      onionBalanceSecretVolume,
			MountPath: onionBalanceSecretMountPath,
		},
	}

	torVolumeMounts := []corev1.VolumeMount{
		{
			Name:      torConfigVolume,
			MountPath: torConfigMountDir,
		},
		{
			Name:      privateKeyVolume,
			MountPath: privateKeyMounPath,
		},
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      onion.DeploymentName(),
			Namespace: onion.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(onion, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    "OnionBalancedService",
				}),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: onion.DeploymentLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: onion.DeploymentLabels(),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: onion.ServiceAccountName(),
					Containers: []corev1.Container{
						{
							Name:  "onionbalance",
							Image: projectConfig.TorOnionbalanceManager.Image,
							Args: []string{
								"-name", onion.Name,
								"-namespace", onion.Namespace,
							},
							ImagePullPolicy: "Always",
							VolumeMounts:    onionBalanceVolumeMounts,
						},
						{
							Name:    "tor",
							Image:   projectConfig.TorDaemonManager.Image, // TODO: use a dedicated Tor image
							Command: []string{"/usr/local/bin/tor"},
							Args: []string{
								"-f", "/run/tor/torfile",
							},
							ImagePullPolicy: "Always",
							VolumeMounts:    torVolumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}
}
