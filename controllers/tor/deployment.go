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

	configv2 "example.com/null/tor-controller/apis/config/v2"
	torv1alpha2 "example.com/null/tor-controller/apis/tor/v1alpha2"
)

const (
	privateKeyVolume = "private-key"
	torConfigVolume  = "tor-config"
)

func (r *OnionServiceReconciler) reconcileDeployment(ctx context.Context, onionService *torv1alpha2.OnionService) error {
	deploymentName := onionService.DeploymentName()
	namespace := onionService.Namespace
	if deploymentName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(fmt.Errorf("%s/%s: deployment name must be specified", onionService.Namespace, onionService.Name))
		return nil
	}

	var deployment appsv1.Deployment
	err := r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespace}, &deployment)

	// If the deployment doesn't exist, we'll create it
	projectConfig := r.ProjectConfig
	newDeployment := torDeployment(onionService, projectConfig)
	if apierrors.IsNotFound(err) {
		err := r.Create(ctx, newDeployment)
		if err != nil {
			return err
		}
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// If the Deployment is not controlled by this Foo resource, we should log
	// a warning to the event recorder and ret
	if !metav1.IsControlledBy(&deployment.ObjectMeta, onionService) {
		msg := fmt.Sprintf("Deployment %s slready exists", deployment.Name)
		// TODO: generate MessageResourceExists event
		// msg := fmt.Sprintf(MessageResourceExists, deployment.Name)
		// bc.recorder.Event(onionService, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}

	// If the deployment specs don't match, update
	if !deploymentEqual(&deployment, newDeployment) {
		err := r.Update(ctx, newDeployment)
		if err != nil {
			return fmt.Errorf("Filed to update Deployment %#v", newDeployment)
		}
	}

	// If an error occurs during Update, we'll requeue the item so we can
	// attempt processing again later. THis could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	return nil
}

func deploymentEqual(a, b *appsv1.Deployment) bool {
	// TODO: actually detect differences
	return false
}

func torDeployment(onion *torv1alpha2.OnionService, projectConfig configv2.ProjectConfig) *appsv1.Deployment {

	privateKeyMountPath := "/run/tor/service/hs_ed25519_secret_key"
	if onion.Spec.GetVersion() == 2 {
		privateKeyMountPath = "/run/tor/service/private_key"
	}

	// allow not specifying a private key
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	if onion.Spec.PrivateKeySecret != (torv1alpha2.SecretReference{}) {
		volumes = []corev1.Volume{
			{
				Name: privateKeyVolume,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: onion.Spec.PrivateKeySecret.Name,
					},
				},
			},
		}

		volumeMounts = []corev1.VolumeMount{
			{
				Name:      privateKeyVolume,
				MountPath: privateKeyMountPath,
				SubPath:   onion.Spec.PrivateKeySecret.Key,
			},
		}
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      onion.DeploymentName(),
			Namespace: onion.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(onion, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    "OnionService",
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
							Name:  "tor",
							Image: fmt.Sprintf("%s", projectConfig.TorDaemonManager.Image),
							Args: []string{
								"-name",
								onion.Name,
								"-namespace",
								onion.Namespace,
							},
							ImagePullPolicy: "Always",
							VolumeMounts:    volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}
}
