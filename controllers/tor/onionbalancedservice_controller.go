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
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8slog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/cockroachdb/errors"

	configv2 "github.com/bugfest/tor-controller/apis/config/v2"
	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
)

const (
	defaultClusterIP = "0.0.0.0"
)

// OnionBalancedServiceReconciler reconciles a OnionBalancedService object.
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
//+kubebuilder:rbac:groups="monitoring.coreos.com",resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apiextensions.k8s.io",resources=customresourcedefinitions,verbs=get;list;watch

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
	logger := k8slog.FromContext(ctx)

	var OnionBalancedService torv1alpha2.OnionBalancedService

	err := r.Get(ctx, req.NamespacedName, &OnionBalancedService)
	if err != nil {
		// The OnionBalancedService resource may no longer exist, in which case we stop
		// processing.
		logger.Error(err, "unable to fetch OnionBalancedService")

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, errors.Wrap(client.IgnoreNotFound(err), "unable to fetch OnionBalancedService")
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

	err = r.reconcileMetricsService(ctx, &OnionBalancedService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileServiceMonitor(ctx, &OnionBalancedService)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Finally, we update the status block of the OnionBalancedService resource to reflect the
	// current state of the world
	OnionBalancedServiceCopy := OnionBalancedService.DeepCopy()
	serviceName := OnionBalancedService.ServiceName()

	var service corev1.Service
	err = r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, &service)

	var clusterIP string

	switch {
	case apierrors.IsNotFound(err):
		clusterIP = defaultClusterIP
	case err != nil:
		return ctrl.Result{}, errors.Wrap(err, "unable to get service")
	default:
		clusterIP = service.Spec.ClusterIP
	}

	OnionBalancedServiceCopy.Status.TargetClusterIP = clusterIP

	// Update backends
	var onionServiceList torv1alpha2.OnionServiceList

	filter := []client.ListOption{
		client.InNamespace(req.Namespace),
	}

	err = r.List(ctx, &onionServiceList, filter...)
	if err != nil {
		// The OnionService (backends) resource may no longer exist, in which case we stop
		// processing.
		logger.Error(err, "unable to list OnionServices")
	}

	backends := map[string]torv1alpha2.OnionServiceStatus{}

	logger.Info("found backends",
		"count", len(onionServiceList.Items))

	for index := range onionServiceList.Items {
		backends[onionServiceList.Items[index].Name] = *onionServiceList.Items[index].Status.DeepCopy()
	}

	OnionBalancedServiceCopy.Status.Backends = backends

	if err := r.Status().Update(ctx, OnionBalancedServiceCopy); err != nil {
		logger.Error(err, "unable to update OnionBalancedService status")

		return ctrl.Result{}, errors.Wrap(err, "unable to update OnionBalancedService status")
	}

	if !OnionBalancedServiceCopy.IsSynced() {
		return ctrl.Result{
			//nolint:gomnd // 3 seconds
			RequeueAfter: 3 * time.Second,
		}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OnionBalancedServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.GenerationChangedPredicate{}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&torv1alpha2.OnionBalancedService{}).
		WithEventFilter(pred).
		Complete(r)
	if err != nil {
		return errors.Wrap(err, "unable to create OnionBalancedService controller")
	}

	return nil
}
