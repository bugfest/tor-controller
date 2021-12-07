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

	tordaemon "example.com/null/tor-controller/agents/tor/tordaemon"

	torv1alpha2 "example.com/null/tor-controller/apis/tor/v1alpha2"
)

var (
	namespace, onionServiceName string
)

func init() {
	flag.StringVar(&namespace, "namespace", "",
		"The namespace of the OnionService to manage.")

	flag.StringVar(&onionServiceName, "name", "",
		"The name of the OnionService to manage.")

}

func GetClient() client.Client {
	scheme := runtime.NewScheme()
	torv1alpha2.AddToScheme(scheme)
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

	daemon tordaemon.Tor

	// controller loop
	controller *Controller
}

func New() *LocalManager {
	t := &LocalManager{
		kclient: GetClient(),
		stopCh:  make(chan struct{}),
		daemon:  tordaemon.Tor{},
	}
	return t
}

func (m *LocalManager) Run() error {
	var errors []error

	if onionServiceName == "" {
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

	err := os.Chmod("/run/tor/service", 0700)
	if err != nil {
		log.Error(err, "error changing /run/tor/service permissions")
	}

	// start watching for API server events that trigger applies
	m.onionServiceCRDWatcher(namespace)

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
			x.FieldSelector = fmt.Sprintf("metadata.name=%s", onionServiceName)
		})

	// "GroupVersionResource" to say what to watch e.g. "deployments.v1.apps" or "seldondeployments.v1.machinelearning.seldon.io"
	gvr, _ := schema.ParseResourceArg(resourceType)

	// Finally, create our informer for deployments!
	informer := factory.ForResource(*gvr)
	return informer, nil
}

func parseOnionService(obj interface{}) (torv1alpha2.OnionService, error) {
	d := torv1alpha2.OnionService{}
	// try following https://erwinvaneyk.nl/kubernetes-unstructured-to-typed/
	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(obj.(*unstructured.Unstructured).UnstructuredContent(), &d)
	if err != nil {
		fmt.Println("could not convert obj to OnionService")
		fmt.Print(err)
		return d, err
	}
	return d, nil
}

func (m *LocalManager) runOnionServiceCRDInformer(stopCh <-chan struct{}, s cache.SharedIndexInformer, namespace string) {

	// create the workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	indexers := cache.Indexers{}

	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			log.Debug("OnionService added")
			onionservice, err := parseOnionService(obj)
			if err == nil {
				log.Info(fmt.Sprintf("Added OnionService: %s/%s", onionservice.Namespace, onionservice.Name))
				key, err := cache.MetaNamespaceKeyFunc(onionservice.GetObjectMeta())
				if err != nil {
					log.Error(err)
				}
				queue.AddAfter(key, 2*time.Second)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			log.Debug("OnionService updated")
			onionservice, err := parseOnionService(newObj)
			if err == nil {
				log.Info(fmt.Sprintf("Updated OnionService: %s/%s", onionservice.Namespace, onionservice.Name))
				key, err := cache.MetaNamespaceKeyFunc(onionservice.GetObjectMeta())
				if err == nil {
					queue.AddAfter(key, 2*time.Second)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			log.Debug("OnionService deleted")
			onionservice, err := parseOnionService(obj)
			if err == nil {
				log.Info(fmt.Sprintf("Deleted OnionService: %s/%s", onionservice.Namespace, onionservice.Name))
				key, err := cache.MetaNamespaceKeyFunc(onionservice.GetObjectMeta())
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

func (m *LocalManager) onionServiceCRDWatcher(namespace string) {
	//dynamic informer needs to be told which type to watch
	onionserviceinformer, _ := GetDynamicInformer("onionservices.v1alpha2.tor.k8s.torproject.org", namespace)
	stopper := make(chan struct{})
	defer close(stopper)
	m.runOnionServiceCRDInformer(stopper, onionserviceinformer.Informer(), namespace)
}
