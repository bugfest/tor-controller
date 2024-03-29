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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8slog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1 "k8s.io/api/core/v1"

	configv2 "github.com/bugfest/tor-controller/apis/config/v2"
	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
	"github.com/cockroachdb/errors"
)

// OnionServiceReconciler reconciles a OnionService object.
type OnionServiceReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	ProjectConfig configv2.ProjectConfig
}

//+kubebuilder:rbac:groups=tor.k8s.torproject.org,resources=onionservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tor.k8s.torproject.org,resources=onionservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tor.k8s.torproject.org,resources=onionservices/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
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
// the OnionService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *OnionServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := k8slog.FromContext(ctx)

	var onionService torv1alpha2.OnionService

	err := r.Get(ctx, req.NamespacedName, &onionService)
	if err != nil {
		// The OnionService resource may no longer exist, in which case we stop
		// processing.
		logger.Error(err, "unable to fetch OnionService")

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, errors.Wrap(client.IgnoreNotFound(err), "unable to fetch OnionService")
	}

	namespace := onionService.Namespace

	for _, rule := range onionService.Spec.Rules {
		serviceName := rule.Backend.Service.Name

		var service corev1.Service

		if err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, &service); err != nil {
			logger.Error(err, "service not found")

			return ctrl.Result{}, errors.Wrap(err, "service not found")
		}

		ruleBackendService := corev1.ServicePort{
			Name:     rule.Backend.Service.Port.Name,
			Port:     rule.Backend.Service.Port.Number,
			Protocol: "TCP",
		}

		if !portExists(service.Spec.Ports, &ruleBackendService) {
			logger.Error(err, "Port not found in target service rule",
				"ruleBackendService", ruleBackendService)

			return ctrl.Result{}, errors.Wrapf(err, "port in service rule %#v not found in target service", ruleBackendService)
		}
	}

	err = r.reconcileSecretAuthorizedClients(ctx, &onionService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileSecret(ctx, &onionService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileServiceAccount(ctx, &onionService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileRole(ctx, &onionService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileRolebinding(ctx, &onionService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileService(ctx, &onionService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileDeployment(ctx, &onionService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileMetricsService(ctx, &onionService)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileServiceMonitor(ctx, &onionService)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Finally, we update the status block of the OnionService resource to reflect the
	// current state of the world
	onionServiceCopy := onionService.DeepCopy()
	serviceName := onionService.ServiceName()

	var service corev1.Service

	err = r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, &service)
	if err != nil {
		logger.Error(err, "unable to get service")
	}

	var clusterIP string

	switch {
	case apierrors.IsNotFound(err):
		clusterIP = defaultClusterIP
	case err != nil:
		return ctrl.Result{}, errors.Wrap(err, "unable to get service")
	default:
		clusterIP = service.Spec.ClusterIP
	}

	onionServiceCopy.Status.TargetClusterIP = clusterIP

	if err := r.Status().Update(ctx, onionServiceCopy); err != nil {
		logger.Error(err, "unable to update OnionService status")

		return ctrl.Result{}, errors.Wrap(err, "unable to update OnionService status")
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OnionServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.GenerationChangedPredicate{}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&torv1alpha2.OnionService{}).
		WithEventFilter(pred).
		Complete(r)
	if err != nil {
		return errors.Wrap(err, "unable to create OnionService controller")
	}

	return nil
}
