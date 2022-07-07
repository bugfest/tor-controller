package local

import (
	"context"
	"flag"
	"fmt"

	// "log"
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

	onionbalancedaemon "github.com/bugfest/tor-controller/agents/onionbalance/onionbalancedaemon"
	torv1alpha3 "github.com/bugfest/tor-controller/apis/tor/v1alpha3"
)

var (
	namespace, onionBalancedServiceName string
)

func init() {
	flag.StringVar(&namespace, "namespace", "",
		"The namespace of the onionBalancedService to manage.")

	flag.StringVar(&onionBalancedServiceName, "name", "",
		"The name of the onionBalancedService to manage.")

}

func GetClient() client.Client {
	scheme := runtime.NewScheme()
	torv1alpha3.AddToScheme(scheme)
	kubeconfig := ctrl.GetConfigOrDie()
	controllerClient, err := client.New(kubeconfig, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatal(err)

		return nil
	}
	return controllerClient
}

type LocalManager struct {
	kclient client.Client

	stopCh chan struct{}

	daemon onionbalancedaemon.OnionBalance

	// controller loop
	controller *Controller
}

func New() *LocalManager {
	t := &LocalManager{
		kclient: GetClient(),
		stopCh:  make(chan struct{}),
		daemon:  onionbalancedaemon.OnionBalance{},
	}
	return t
}

func (m *LocalManager) Run() error {
	var errors []error

	if onionBalancedServiceName == "" {
		errors = append(errors, fmt.Errorf("-name flag cannot be empty"))
	}
	if namespace == "" {
		errors = append(errors, fmt.Errorf("-namespace flag cannot be empty"))
	}
	if err := utilerrors.NewAggregate(errors); err != nil {
		return err
	}

	// listen to signals
	signalCh := make(chan os.Signal, 1)
	// signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGHUP)
	m.signalHandler(signalCh)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.daemon.SetContext(ctx)

	// start watching for API server events that trigger applies
	m.onionBalancedServiceCRDWatcher(namespace)

	// Wait for all goroutines to exit
	<-m.stopCh

	return nil
}

func (m *LocalManager) Must(err error) *LocalManager {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return m
}

func (m *LocalManager) signalHandler(ch chan os.Signal) {
	go func() {
		select {
		case <-m.stopCh:
			break
		case sig := <-ch:
			switch sig {
			case syscall.SIGHUP:
				fmt.Println("received SIGHUP")

			case syscall.SIGINT:
				fmt.Println("received SIGINT")
				close(m.stopCh)

			case syscall.SIGTERM:
				fmt.Println("received SIGTERM")
				close(m.stopCh)
			}
		}
	}()
}

func GetDynamicInformer(resourceType string, namespace string) (informers.GenericInformer, error) {
	cfg := ctrl.GetConfigOrDie()

	// Grab a dynamic interface that we can create informers from
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	// Create a factory object that can generate informers for resource types

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dc, 0,
		namespace,
		func(x *metav1.ListOptions) {
			x.FieldSelector = fmt.Sprintf("metadata.name=%s", onionBalancedServiceName)
		})

	// "GroupVersionResource" to say what to watch e.g. "deployments.v1.apps" or "seldondeployments.v1.machinelearning.seldon.io"
	gvr, _ := schema.ParseResourceArg(resourceType)

	// Finally, create our informer for deployments!
	informer := factory.ForResource(*gvr)
	return informer, nil
}

func parseOnionBalancedService(obj interface{}) (torv1alpha3.OnionBalancedService, error) {
	d := torv1alpha3.OnionBalancedService{}
	// try following https://erwinvaneyk.nl/kubernetes-unstructured-to-typed/
	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(obj.(*unstructured.Unstructured).UnstructuredContent(), &d)
	if err != nil {
		fmt.Println("could not convert obj to onionBalancedService")
		fmt.Print(err)
		return d, err
	}
	return d, nil
}

func (m *LocalManager) runOnionBalancedServiceCRDInformer(stopCh <-chan struct{}, s cache.SharedIndexInformer, namespace string) {

	// create the workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	indexers := cache.Indexers{}

	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			log.Debug("onionBalancedService added")
			onionBalancedService, err := parseOnionBalancedService(obj)
			if err == nil {
				log.Info(fmt.Sprintf("Added onionBalancedService: %s/%s", onionBalancedService.Namespace, onionBalancedService.Name))
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
				log.Info(fmt.Sprintf("Updated onionBalancedService: %s/%s", onionBalancedService.Namespace, onionBalancedService.Name))
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
				log.Info(fmt.Sprintf("Deleted onionBalancedService: %s/%s", onionBalancedService.Namespace, onionBalancedService.Name))
				key, err := cache.MetaNamespaceKeyFunc(onionBalancedService.GetObjectMeta())
				if err == nil {
					queue.AddAfter(key, 2*time.Second)
				}
			}
		},
	}
	s.AddEventHandler(handlers)
	s.AddIndexers(indexers)
	go s.Run(stopCh)
	log.Info("Listening for events")

	m.controller = NewController(queue, s, m)

	log.Info("Running event controller")
	go m.controller.Run(1, m.stopCh)
	<-stopCh
}

func (m *LocalManager) onionBalancedServiceCRDWatcher(namespace string) {
	//dynamic informer needs to be told which type to watch
	onionBalancedServiceinformer, _ := GetDynamicInformer("onionbalancedservices.v1alpha2.tor.k8s.torproject.org", namespace)
	stopper := make(chan struct{})
	defer close(stopper)
	m.runOnionBalancedServiceCRDInformer(stopper, onionBalancedServiceinformer.Informer(), namespace)
}
