package local

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"

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

	tordaemon "github.com/bugfest/tor-controller/agents/tor/tordaemon"
	torv1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
)

var namespace, onionServiceName string

func init() {
	flag.StringVar(&namespace, "namespace", "",
		"The namespace of the OnionService to manage.")

	flag.StringVar(&onionServiceName, "name", "",
		"The name of the OnionService to manage.")
}

// GetClient returns a client for the torv1alpha2 OnionService CRD
func GetClient() client.Client {
	scheme := runtime.NewScheme()
	kubeconfig := ctrl.GetConfigOrDie()

	err := torv1alpha2.AddToScheme(scheme)
	if err != nil {
		log.Println(err)
	}

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
	return &LocalManager{
		kclient: GetClient(),
		stopCh:  make(chan struct{}),
		daemon:  tordaemon.Tor{},
	}
}

func (manager *LocalManager) Run() error {
	var runErrors []error

	if onionServiceName == "" {
		runErrors = append(runErrors, errors.New("-name flag cannot be empty"))
	}

	if namespace == "" {
		runErrors = append(runErrors, errors.New("-namespace flag cannot be empty"))
	}

	if err := utilerrors.NewAggregate(runErrors); err != nil {
		return errors.Wrap(err, "error parsing flags")
	}

	// listen to signals
	signalCh := make(chan os.Signal, 1)
	// signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGHUP)
	manager.signalHandler(signalCh)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.daemon.SetContext(ctx)

	err := os.Chmod("/run/tor/service", 0o700)
	if err != nil {
		log.Error(err, "error changing /run/tor/service permissions")
	}

	// start watching for API server events that trigger applies
	manager.onionServiceCRDWatcher(namespace)

	// Wait for all goroutines to exit
	<-manager.stopCh

	return nil
}

func (manager *LocalManager) Must(err error) *LocalManager {
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	return manager
}

func (manager *LocalManager) signalHandler(ch chan os.Signal) {
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

func GetDynamicInformer(resourceType string, namespace string) (informers.GenericInformer, error) {
	cfg := ctrl.GetConfigOrDie()

	// Grab a dynamic interface that we can create informers from
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "error creating dynamic client")
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
	service := torv1alpha2.OnionService{}
	// try following https://erwinvaneyk.nl/kubernetes-unstructured-to-typed/
	err := runtime.DefaultUnstructuredConverter.
		FromUnstructured(obj.(*unstructured.Unstructured).UnstructuredContent(), &service)
	if err != nil {
		log.Println("could not convert obj to OnionService")
		log.Print(err)

		return service, errors.Wrap(err, "could not convert obj to OnionService")
	}

	return service, nil
}

// onionServiceCRDWatcher watches for OnionService CRD events
func (manager *LocalManager) runOnionServiceCRDInformer(stopCh <-chan struct{}, sharedIndexInformer cache.SharedIndexInformer, _ string) {
	// create the workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	indexers := cache.Indexers{}

	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			log.Debug("OnionService added")
			onionservice, err := parseOnionService(obj)
			if err == nil {
				log.Infof("Added OnionService: %s/%s", onionservice.Namespace, onionservice.Name)
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
				log.Infof("Updated OnionService: %s/%s", onionservice.Namespace, onionservice.Name)
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
				log.Infof("Deleted OnionService: %s/%s", onionservice.Namespace, onionservice.Name)
				key, err := cache.MetaNamespaceKeyFunc(onionservice.GetObjectMeta())
				if err == nil {
					queue.AddAfter(key, 2*time.Second)
				}
			}
		},
	}
	sharedIndexInformer.AddEventHandler(handlers)
	sharedIndexInformer.AddIndexers(indexers)

	go sharedIndexInformer.Run(stopCh)

	log.Info("Listening for events")

	manager.controller = NewController(queue, sharedIndexInformer, manager)

	log.Info("Running event controller")
	go manager.controller.Run(1, manager.stopCh)
	<-stopCh
}

func (m *LocalManager) onionServiceCRDWatcher(namespace string) {
	// dynamic informer needs to be told which type to watch
	onionserviceinformer, _ := GetDynamicInformer("onionservices.v1alpha2.tor.k8s.torproject.org", namespace)
	stopper := make(chan struct{})
	defer close(stopper)
	m.runOnionServiceCRDInformer(stopper, onionserviceinformer.Informer(), namespace)
}
