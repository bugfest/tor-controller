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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	k8slog "sigs.k8s.io/controller-runtime/pkg/log"

	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
	"github.com/cockroachdb/errors"
)

func (r *OnionServiceReconciler) reconcileSecret(ctx context.Context, onionService *torv1alpha2.OnionService) error {
	logger := k8slog.FromContext(ctx)

	secretName := onionService.SecretName()
	namespace := onionService.Namespace

	if secretName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(errors.New("secret name must be specified"))

		return nil
	}

	var secret corev1.Secret
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, &secret)

	newSecret := torOnionServiceSecret(onionService)
	if apierrors.IsNotFound(err) {
		err := r.Create(ctx, newSecret)
		if err != nil {
			return errors.Wrap(err, "failed to create secret")
		}
		secret = *newSecret
	} else if err != nil {
		return errors.Wrap(err, "failed to get secret")
	}

	if !metav1.IsControlledBy(&secret.ObjectMeta, onionService) {
		// msg := fmt.Sprintf("Secret %s already exists and is not controller by %s", secret.Name, onionService.Name)
		// TODO: generate MessageResourceExists event
		// msg := fmt.Sprintf(MessageResourceExists, service.Name)
		// bc.recorder.Event(onionService, corev1.EventTypeWarning, ErrResourceExists, msg)
		// return errors.New(msg)
		logger.Info("Secret already exists and is not controlled by",
			"secret", secret.Name,
			"controller", onionService.Name)

		return nil
	}

	return nil
}

func torOnionServiceSecret(onion *torv1alpha2.OnionService) *corev1.Secret {
	onionv3, err := GenerateOnionV3()
	if err != nil {
		k8slog.Log.Error(err, "error generating Onion keys")

		return nil
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      onion.SecretName(),
			Namespace: onion.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(onion, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    "OnionService",
				}),
			},
		},
		Type: "tor.k8s.torproject.org/onion-v3",
		Data: map[string][]byte{
			"onionAddress":   []byte(onionv3.onionAddress),
			"publicKey":      onionv3.publicKey,
			"privateKey":     onionv3.privateKey,
			"publicKeyFile":  onionv3.publicKeyFile,
			"privateKeyFile": onionv3.privateKeyFile,
		},
	}
}
