package local

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cockroachdb/errors"

	onionbalancedaemon "github.com/bugfest/tor-controller/agents/onionbalance/onionbalancedaemon"
	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
)

var namespace, onionBalancedServiceName string

func init() {
	flag.StringVar(&namespace, "namespace", "",
		"The namespace of the onionBalancedService to manage.")

	flag.StringVar(&onionBalancedServiceName, "name", "",
		"The name of the onionBalancedService to manage.")
}

func GetClient() client.Client {
	scheme := runtime.NewScheme()

	err := torv1alpha2.AddToScheme(scheme)
	if err != nil {
		log.Println(err)
	}

	kubeconfig := ctrl.GetConfigOrDie()

	controllerClient, err := client.New(kubeconfig, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatal(err)

		return nil
	}

	return controllerClient
}

// Manager is a local onionbalance manager.
type Manager struct {
	kclient client.Client

	stopCh chan struct{}

	daemon onionbalancedaemon.OnionBalance

	// controller loop
	controller *Controller
}

func New() *Manager {
	return &Manager{
		kclient: GetClient(),
		stopCh:  make(chan struct{}),
		daemon:  onionbalancedaemon.OnionBalance{},
	}
}

func (manager *Manager) Run() error {
	var runErrors []error

	if onionBalancedServiceName == "" {
		runErrors = append(runErrors, errors.New("-name flag cannot be empty"))
	}

	if namespace == "" {
		runErrors = append(runErrors, errors.New("-namespace flag cannot be empty"))
	}

	if err := utilerrors.NewAggregate(runErrors); err != nil {
		return err
	}

	// listen to signals
	signalCh := make(chan os.Signal, 1)

	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGHUP)
	manager.signalHandler(signalCh)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.daemon.SetContext(ctx)

	// start watching for API server events that trigger applies
	manager.onionBalancedServiceCRDWatcher(namespace)

	// Wait for all goroutines to exit
	<-manager.stopCh

	return nil
}

func (manager *Manager) Must(err error) *Manager {
	if err != nil {
		log.Fatal(err)
	}

	return manager
}

func (manager *Manager) signalHandler(ch chan os.Signal) {
	go func() {
		select {
		case <-manager.stopCh:
			break
		case sig := <-ch:
			switch sig {
			case syscall.SIGHUP:
				log.Println("received SIGHUP")

			case syscall.SIGINT:
				log.Println("received SIGINT")
				close(manager.stopCh)

			case syscall.SIGTERM:
				log.Println("received SIGTERM")
				close(manager.stopCh)
			}
		}
	}()
}

func GetDynamicInformer(resourceType, namespace string) (informers.GenericInformer, error) {
	cfg := ctrl.GetConfigOrDie()

	// Grab a dynamic interface that we can create informers from
	dynamicConfig, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "could not create dynamic client")
	}
	// Create a factory object that can generate informers for resource types

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicConfig, 0,
		namespace,
		func(x *metav1.ListOptions) {
			x.FieldSelector = "metadata.name" + onionBalancedServiceName
		})

	// "GroupVersionResource" to say what to watch e.g. "deployments.v1.apps" or "seldondeployments.v1.machinelearning.seldon.io"
	gvr, _ := schema.ParseResourceArg(resourceType)

	// Finally, create our informer for deployments!
	informer := factory.ForResource(*gvr)

	return informer, nil
}

func parseOnionBalancedService(obj interface{}) (torv1alpha2.OnionBalancedService, error) {
	onionBalancedService := torv1alpha2.OnionBalancedService{}
	// try following https://erwinvaneyk.nl/kubernetes-unstructured-to-typed/

	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Println("could not convert obj to unstructured")

		return onionBalancedService, errors.New("could not convert obj to unstructured")
	}

	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(unstructuredObj.UnstructuredContent(), &onionBalancedService)
	if err != nil {
		log.Println("could not convert obj to onionBalancedService")
		log.Print(err)

		return onionBalancedService, errors.Wrap(err, "could not convert obj to onionBalancedService")
	}

	return onionBalancedService, nil
}

func (manager *Manager) runOnionBalancedServiceCRDInformer(stopCh <-chan struct{}, sharedIndexInformer cache.SharedIndexInformer, _ string) {
	// create the workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	indexers := cache.Indexers{}

	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			log.Debug("onionBalancedService added")
			onionBalancedService, err := parseOnionBalancedService(obj)
			if err == nil {
				log.Infof("Added onionBalancedService: %s/%s", onionBalancedService.Namespace, onionBalancedService.Name)
				key, err := cache.MetaNamespaceKeyFunc(onionBalancedService.GetObjectMeta())
				if err != nil {
					log.Error(err)
				}
				queue.AddAfter(key, 2*time.Second)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			log.Debug("onionBalancedService updated")
			onionBalancedService, err := parseOnionBalancedService(newObj)
			if err == nil {
				log.Infof("Updated onionBalancedService: %s/%s", onionBalancedService.Namespace, onionBalancedService.Name)
				key, err := cache.MetaNamespaceKeyFunc(onionBalancedService.GetObjectMeta())
				if err == nil {
					queue.AddAfter(key, 2*time.Second)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			log.Debug("onionBalancedService deleted")
			onionBalancedService, err := parseOnionBalancedService(obj)
			if err == nil {
				log.Infof("Deleted onionBalancedService: %s/%s", onionBalancedService.Namespace, onionBalancedService.Name)
				key, err := cache.MetaNamespaceKeyFunc(onionBalancedService.GetObjectMeta())
				if err == nil {
					queue.AddAfter(key, 2*time.Second)
				}
			}
		},
	}

	sharedIndexInformer.AddEventHandler(handlers)

	err := sharedIndexInformer.AddIndexers(indexers)
	if err != nil {
		log.Errorf("Error adding indexers: %s", err)
	}

	go sharedIndexInformer.Run(stopCh)

	log.Info("Listening for events")

	manager.controller = NewController(queue, sharedIndexInformer, manager)

	log.Info("Running event controller")

	go manager.controller.Run(1, manager.stopCh)

	<-stopCh
}

func (manager *Manager) onionBalancedServiceCRDWatcher(namespace string) {
	// dynamic informer needs to be told which type to watch
	onionBalancedServiceinformer, _ := GetDynamicInformer("onionbalancedservices.v1alpha2.tor.k8s.torproject.org", namespace)
	stopper := make(chan struct{})

	defer close(stopper)

	manager.runOnionBalancedServiceCRDInformer(stopper, onionBalancedServiceinformer.Informer(), namespace)
}
