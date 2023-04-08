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

const (
	authTypeLabel   = "authType"
	keyTypeLabel    = "keyType"
	publicKeyLabel  = "publicKey"
	privateKeyLabel = "privateKey"
	authKeyLabel    = "authKey"

	authTypeDefault = "descriptor"
	keyTypeDefault  = "x25519"
)

func (r *OnionServiceReconciler) reconcileSecretAuthorizedClients(ctx context.Context, onionService *torv1alpha2.OnionService) error {
	logger := k8slog.FromContext(ctx)

	secretName := onionService.AuthorizedClientsSecretName()
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

	authorizedClients := map[string][]byte{}

	var authorizedClientSecret corev1.Secret

	for idx, authorizedClientSecretRef := range onionService.Spec.AuthorizedClients {
		acErr := r.Get(ctx, types.NamespacedName{Name: authorizedClientSecretRef.Name, Namespace: namespace}, &authorizedClientSecret)

		if acErr != nil {
			logger.Info("authorizedClientSecretRef not found - skipping",
				"authorizedClientSecretRef", authorizedClientSecretRef.Name)
		} else {
			// expeted keys:
			// - authType (optional, default="descriptor") -> skipped, we use default
			// - keyType (optional, default="x25519") -> skipped, we use default
			// - publicKey (optional)
			// - privateKey (optional)
			// - authKey (optional) -> example: "descriptor:x25519:N2NU7BSRL6YODZCYPN4CREB54TYLKGIE2KYOQWLFYC23ZJVCE5DQ"

			if len(authorizedClientSecretRef.Key) > 0 {
				// if the secret key is specified, we assume it's an authKey
				if authKey, authKeyExists := authorizedClientSecret.Data[authorizedClientSecretRef.Key]; authKeyExists {
					authorizedClients[fmt.Sprintf("client-%d.auth", idx)] = authKey
				} else {
					logger.Info("authorizedClientSecretRef not found - skipping",
						"authorizedClientSecretRef", authorizedClientSecretRef.Name,
						"key", authorizedClientSecretRef.Key)
				}
			} else {
				// secretRef does not specify a key. Check if "authKey" key exists
				if authKey, authKeyExists := authorizedClientSecret.Data[authKeyLabel]; authKeyExists {
					authorizedClients[fmt.Sprintf("client-%d.auth", idx)] = authKey
				} else {
					// if "authKey" key is not present, try to get "publicKey" to generate a valid authKey string instead
					if publicKey, publicKeyExists := authorizedClientSecret.Data[publicKeyLabel]; publicKeyExists {
						authorizedClients[fmt.Sprintf("client-%d.auth", idx)] = []byte(fmt.Sprintf("%s:%s:%s", authTypeDefault, keyTypeDefault, publicKey))
					} else {
						logger.Info("authorizedClientSecretRef is not valid - skipping",
							"authorizedClientSecretRef", authorizedClientSecretRef.Name)
					}
				}
			}
		}
	}

	newSecret := torOnionServiceSecretAuthorizedClients(onionService, authorizedClients)
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

func torOnionServiceSecretAuthorizedClients(onion *torv1alpha2.OnionService, authorizedClients map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      onion.AuthorizedClientsSecretName(),
			Namespace: onion.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(onion, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    "OnionService",
				}),
			},
		},
		Type: "tor.k8s.torproject.org/authorized-clients-v3",
		Data: authorizedClients,
	}
}
