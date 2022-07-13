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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1 "k8s.io/api/core/v1"

	configv2 "github.com/bugfest/tor-controller/apis/config/v2"
	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
)

// TorReconciler reconciles a Tor object
type TorReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	ProjectConfig configv2.ProjectConfig
}

//+kubebuilder:rbac:groups=tor.k8s.torproject.org,resources=tors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tor.k8s.torproject.org,resources=tors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tor.k8s.torproject.org,resources=tors/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Tor object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *TorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	//namespace, name := req.Namespace, req.Name
	var tor torv1alpha2.Tor

	err := r.Get(ctx, req.NamespacedName, &tor)
	if err != nil {
		// The Tor resource may no longer exist, in which case we stop
		// processing.

		log.Error(err, "unable to fetch Tor")

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	namespace := tor.Namespace

	// err = r.reconcileServiceAccount(ctx, &tor)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }

	// err = r.reconcileRole(ctx, &tor)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }

	// err = r.reconcileRolebinding(ctx, &tor)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }

	err = r.reconcileService(ctx, &tor)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileConfigMap(ctx, &tor)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileDeployment(ctx, &tor)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileMetricsService(ctx, &tor)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileServiceMonitor(ctx, &tor)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Finally, we update the status block of the Tor resource to reflect the
	// current state of the world
	torCopy := tor.DeepCopy()
	instanceName := tor.InstanceName()

	var service corev1.Service
	if err := r.Get(ctx, types.NamespacedName{Name: instanceName, Namespace: namespace}, &service); err != nil {
		log.Error(err, "unable to get service")
		return ctrl.Result{}, err
	}

	torCopy.Status.Config = "updateme"

	if err := r.Status().Update(ctx, torCopy); err != nil {
		log.Error(err, "unable to update Tor status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&torv1alpha2.Tor{}).
		WithEventFilter(pred).
		Complete(r)
}
