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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1 "k8s.io/api/core/v1"

	configv2 "github.com/bugfest/tor-controller/apis/config/v2"
	// torv1alpha1 "github.com/bugfest/tor-controller/apis/tor/v1alpha1"
	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
)

// OnionBalancedServiceReconciler reconciles a OnionBalancedService object
type OnionBalancedServiceReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	ProjectConfig configv2.ProjectConfig
}

//+kubebuilder:rbac:groups=tor.k8s.torproject.org,resources=onionbalancedservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tor.k8s.torproject.org,resources=onionbalancedservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tor.k8s.torproject.org,resources=onionbalancedservices/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OnionBalancedService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *OnionBalancedServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	//namespace, name := req.Namespace, req.Name
	var OnionBalancedService torv1alpha2.OnionBalancedService

	err := r.Get(ctx, req.NamespacedName, &OnionBalancedService)
	if err != nil {
		// The OnionBalancedService resource may no longer exist, in which case we stop
		// processing.

		log.Error(err, "unable to fetch OnionBalancedService")

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	namespace := OnionBalancedService.Namespace

	// it is important to reconcile secret first, as it will define the
	// onion hostname of the master instance
	err = r.reconcileSecret(ctx, &OnionBalancedService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileServiceAccount(ctx, &OnionBalancedService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileRole(ctx, &OnionBalancedService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileRolebinding(ctx, &OnionBalancedService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileService(ctx, &OnionBalancedService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileConfigMap(ctx, &OnionBalancedService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileDeployment(ctx, &OnionBalancedService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileBackends(ctx, &OnionBalancedService)
	if err != nil {
		return ctrl.Result{}, err
	}

	// bc.recorder.Event(OnionBalancedService, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)

	// Finally, we update the status block of the OnionBalancedService resource to reflect the
	// current state of the world
	OnionBalancedServiceCopy := OnionBalancedService.DeepCopy()
	serviceName := OnionBalancedService.ServiceName()

	var service corev1.Service
	err = r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, &service)

	clusterIP := ""
	if errors.IsNotFound(err) {
		clusterIP = "0.0.0.0"
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		clusterIP = service.Spec.ClusterIP
	}

	OnionBalancedServiceCopy.Status.TargetClusterIP = clusterIP
	// hostname := "test.onion"
	// OnionBalancedService.Status.Hostname = hostname

	// Update backends
	var onionServiceList torv1alpha2.OnionServiceList
	filter := []client.ListOption{
		client.InNamespace(req.Namespace),
		// client.MatchingLabels{"instance": req.NamespacedName.Name},
		// client.MatchingFields{"status.phase": "Running"},
	}

	err = r.List(ctx, &onionServiceList, filter...)
	if err != nil {
		// The OnionService (backends) resource may no longer exist, in which case we stop
		// processing.
		log.Error(err, "unable to list OnionServices")
	}

	backends := map[string]torv1alpha2.OnionServiceStatus{}
	log.Info(fmt.Sprintf("Found %d backends", len(onionServiceList.Items)))

	for _, onionService := range onionServiceList.Items {
		backends[onionService.Name] = *onionService.Status.DeepCopy()
	}
	OnionBalancedServiceCopy.Status.Backends = backends

	if err := r.Status().Update(ctx, OnionBalancedServiceCopy); err != nil {
		log.Error(err, "unable to update OnionBalancedService status")
		return ctrl.Result{}, err
	}

	if !OnionBalancedServiceCopy.IsSynced() {
		return ctrl.Result{
			RequeueAfter: 3 * time.Second,
		}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OnionBalancedServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&torv1alpha2.OnionBalancedService{}).
		WithEventFilter(pred).
		Complete(r)
}
