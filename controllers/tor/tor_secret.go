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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
	"github.com/m1/go-generate-password/generator"
)

func (r *TorReconciler) reconcileControlSecret(ctx context.Context, Tor *torv1alpha2.Tor) error {
	log := log.FromContext(ctx)

	secretName := Tor.SecretName()
	namespace := Tor.Namespace
	if secretName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(fmt.Errorf("secret name must be specified"))
		return nil
	}

	var secret corev1.Secret
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, &secret)

	password := generateRandomPassword()
	newSecret := torSecret(Tor, password)
	if errors.IsNotFound(err) {
		err := r.Create(ctx, newSecret)
		if err != nil {
			return err
		}
		secret = *newSecret
	} else if err != nil {
		return err
	}

	var tmpSecret corev1.Secret
	for _, secretRef := range Tor.Spec.Control.SecretRef {
		err := r.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: namespace}, &tmpSecret)
		if err != nil {
			return err
		}

		for k := range tmpSecret.Data {
			if k == secretRef.Key {
				// Adds all the referenced secrets to Tor.Spec.Control.Secret for later use
				Tor.Spec.Control.Secret = append(Tor.Spec.Control.Secret, string(tmpSecret.Data[k]))
			}
		}
	}

	if len(Tor.Spec.Control.Secret) == 0 && len(Tor.Spec.Control.SecretRef) == 0 {
		// If the user did not define any password in the Control spec,
		// update Control Secrets with the generated one
		for _, s := range secret.Data {
			Tor.Spec.Control.Secret = append(Tor.Spec.Control.Secret, string(s))
		}
	}

	if !metav1.IsControlledBy(&secret.ObjectMeta, Tor) {
		// msg := fmt.Sprintf("Secret %s already exists and is not controller by %s", secret.Name, OnionBalancedService.Name)
		// TODO: generate MessageResourceExists event
		// msg := fmt.Sprintf(MessageResourceExists, service.Name)
		// bc.recorder.Event(OnionBalancedService, corev1.EventTypeWarning, ErrResourceExists, msg)
		// return fmt.Errorf(msg)
		log.Info(fmt.Sprintf("Secret %s already exists and is not controller by %s", secret.Name, Tor.Name))
		return nil
	}

	return nil
}

func torSecret(tor *torv1alpha2.Tor, password string) *corev1.Secret {

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tor.SecretName(),
			Namespace: tor.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(tor, schema.GroupVersionKind{
					Group:   torv1alpha2.GroupVersion.Group,
					Version: torv1alpha2.GroupVersion.Version,
					Kind:    "Tor",
				}),
			},
		},
		Type: "tor.k8s.torproject.org/control-password",
		Data: map[string][]byte{
			"control": []byte(password),
		},
	}
}

func generateRandomPassword() string {
	config := generator.Config{
		Length:                     16,
		IncludeSymbols:             false,
		IncludeNumbers:             true,
		IncludeLowercaseLetters:    true,
		IncludeUppercaseLetters:    true,
		ExcludeSimilarCharacters:   true,
		ExcludeAmbiguousCharacters: true,
	}
	g, _ := generator.New(&config)

	pwd, _ := g.Generate()
	return *pwd
}
