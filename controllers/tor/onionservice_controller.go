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
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1 "k8s.io/api/core/v1"

	// torv1alpha1 "example.com/null/tor-controller/apis/tor/v1alpha1"

	torv1alpha2 "example.com/null/tor-controller/apis/tor/v1alpha2"
)

// OnionServiceReconciler reconciles a OnionService object
type OnionServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=tor.k8s.torproject.org,resources=onionservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tor.k8s.torproject.org,resources=onionservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tor.k8s.torproject.org,resources=onionservices/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch

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
	log := log.FromContext(ctx)

	//namespace, name := req.Namespace, req.Name
	var onionService torv1alpha2.OnionService

	err := r.Get(ctx, req.NamespacedName, &onionService)
	if err != nil {
		// The OnionService resource may no longer exist, in which case we stop
		// processing.

		log.Error(err, "unable to fetch OnionService")

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// err = bc.reconcileServiceAccount(onionService)
	// if err != nil {
	// 	return err
	// }

	// err = bc.reconcileRole(onionService)
	// if err != nil {
	// 	return err
	// }

	// err = bc.reconcileRolebinding(onionService)
	// if err != nil {
	// 	return err
	// }

	// err = bc.reconcileService(onionService)
	// if err != nil {
	// 	return err
	// }

	// err = bc.reconcileDeployment(onionService)
	// if err != nil {
	// 	return err
	// }

	// Finally, we update the status block of the OnionService resource to reflect the
	// current state of the world
	// err = bc.updateOnionServiceStatus(onionService)
	// if err != nil {
	// 	return err
	// }

	// bc.recorder.Event(onionService, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)

	namespace := onionService.Namespace

	for num, rule := range onionService.Spec.Rules {

		log.Info(fmt.Sprintf("rule %d: %#v", num, rule))

		serviceName := rule.Backend.Service.Name
		var service corev1.Service

		if err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, &service); err != nil {
			log.Error(err, "service not found")
			return ctrl.Result{}, err
		}

		if !portExists(
			service.Spec.Ports,
			corev1.ServicePort{
				Name:     rule.Backend.Service.Port.Name,
				Port:     rule.Backend.Service.Port.Number,
				Protocol: "TCP",
			}) {
			log.Error(err, "port not in service")
			return ctrl.Result{}, err
		}
		log.Info("port in service OK")
	}

	clusterIP := ""
	if errors.IsNotFound(err) {
		clusterIP = "0.0.0.0"
	} else {
		// clusterIP = service.Spec.ClusterIP
		clusterIP = "1.2.3.5"
	}

	if onionService.Status.TargetClusterIP != clusterIP {
		onionService.Status.TargetClusterIP = clusterIP
	}

	hostname := "test.onion"
	if onionService.Status.TargetClusterIP != hostname {
		onionService.Status.Hostname = hostname
	}

	if err := r.Status().Update(ctx, &onionService); err != nil {
		log.Error(err, "unable to update OnionService status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OnionServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.GenerationChangedPredicate{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&torv1alpha2.OnionService{}).
		WithEventFilter(pred).
		Complete(r)
}

func itemExists(slice interface{}, item interface{}) bool {
	s := reflect.ValueOf(slice)

	if s.Kind() != reflect.Slice {
		panic("Invalid data-type")
	}

	for i := 0; i < s.Len(); i++ {
		if s.Index(i).Interface() == item {
			return true
		}
	}

	return false
}

func portExists(slice []corev1.ServicePort, item corev1.ServicePort) bool {
	s := reflect.ValueOf(slice)

	if s.Kind() != reflect.Slice {
		panic("Invalid data-type")
	}

	for _, p := range slice {
		if p.Protocol == item.Protocol {
			if p.Port == item.Port {
				return true
			}
			if p.Name == item.Name {
				return true
			}
		}
	}

	return false
}
