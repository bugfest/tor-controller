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

func (r *TorReconciler) reconcileDeployment(ctx context.Context, tor *torv1alpha2.Tor) error {
	log := log.FromContext(ctx)

	deploymentName := tor.DeploymentName()
	namespace := tor.Namespace
	if deploymentName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(fmt.Errorf("%s/%s: deployment name must be specified", tor.Namespace, tor.Name))
		return nil
	}

	var deployment appsv1.Deployment
	err := r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespace}, &deployment)

	// If the deployment doesn't exist, we'll create it
	projectConfig := r.ProjectConfig
	newDeployment := torDeployment(tor, projectConfig)
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
	if !metav1.IsControlledBy(&deployment.ObjectMeta, tor) {
		log.Info(fmt.Sprintf("Deployment %s already exists and not controlled by %s - skipping update", deployment.Name, tor.Name))
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

func torDeployment(tor *torv1alpha2.Tor, projectConfig configv2.ProjectConfig) *appsv1.Deployment {

	// new deployment
	if tor.Spec.Replicas == 0 {
		tor.Spec.Replicas = 1
	}

	torConfigMountDir := "/run/tor"
	torVolumeMounts := []corev1.VolumeMount{
		{
			Name:      torConfigVolume,
			MountPath: torConfigMountDir,
		},
	}

	torArgs := append(
		[]string{"-f", "/run/tor/torfile"},
		tor.Spec.Args...,
	)

	volumes := []corev1.Volume{
		{
			Name: torConfigVolume,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: tor.ConfigMapName(),
					},
					Items: []corev1.KeyToPath{{
						Key:  "torfile",
						Path: "torfile",
					}},
				},
			},
		},
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tor.DeploymentName(),
			Namespace: tor.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tor, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    "Tor",
				}),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: tor.DeploymentLabels(),
			},
			Replicas: &tor.Spec.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: tor.DeploymentLabels(),
				},
				Spec: corev1.PodSpec{
					// ServiceAccountName: tor.ServiceAccountName(),
					Containers: []corev1.Container{
						{
							Name:            "tor",
							Image:           projectConfig.TorDaemon.Image,
							Args:            torArgs,
							ImagePullPolicy: corev1.PullAlways,
							VolumeMounts:    torVolumeMounts,
							Ports:           getTorContainerPortList(tor),
						},
					},
					Volumes: volumes,
				},
			},
		},
	}
}

func getTorContainerPortList(tor *torv1alpha2.Tor) []corev1.ContainerPort {
	ports := []corev1.ContainerPort{}

	for _, r := range tor.GetAllPorts() {
		if r.Port.Enable {
			port := corev1.ContainerPort{
				Name:          r.Name,
				Protocol:      corev1.Protocol(r.Protocol),
				ContainerPort: r.Port.Port,
			}
			ports = append(ports, port)
		}
	}

	return ports
}
