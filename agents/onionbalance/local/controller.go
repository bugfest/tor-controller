package local

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	config "github.com/bugfest/tor-controller/agents/onionbalance/config"
	v1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
)

type Controller struct {
	indexer      cache.Indexer
	queue        workqueue.RateLimitingInterface
	informer     cache.Controller
	localManager *LocalManager
}

func NewController(queue workqueue.RateLimitingInterface, informer cache.SharedIndexInformer, localManager *LocalManager) *Controller {
	return &Controller{
		informer:     informer,
		indexer:      informer.GetIndexer(),
		queue:        queue,
		localManager: localManager,
	}
}

func (c *Controller) processNextItem() bool {
	key, quit := c.queue.Get()
	if quit {
		log.Info("Queue quits")
		return false
	}

	defer c.queue.Done(key)

	err := c.sync(key.(string))
	c.handleErr(err, key)
	return true
}

func (c *Controller) sync(key string) error {
	log.Info(fmt.Sprintf("Getting key %s", key))
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		log.Error(fmt.Sprintf("Fetching object with key %s from store failed with %v", key, err))
		return err
	}

	if !exists {
		log.Warn(fmt.Sprintf("onionBalancedService %s does not exist anymore", key))
	} else {
		log.Debug(fmt.Sprintf("%v", obj))
		onionBalancedService, err := parseOnionBalancedService(obj)
		if err != nil {
			log.Error(fmt.Sprintf("Error in parseonionBalancedService: %s", err))
			return err
		}

		torConfig, err := config.OnionBalanceConfigForService(&onionBalancedService)
		if err != nil {
			log.Error(fmt.Sprintf("Generating config failed with %v", err))
			return err
		}

		reload := false

		torfile, err := ioutil.ReadFile("/run/onionbalance/config.yaml")
		if os.IsNotExist(err) {
			reload = true
		} else if err != nil {
			return err
		}

		if string(torfile) != torConfig {
			reload = true
		}

		if reload {
			log.Info(fmt.Sprintf("Updating onionbalance config for %s/%s", onionBalancedService.Namespace, onionBalancedService.Name))

			err = ioutil.WriteFile("/run/onionbalance/config.yaml", []byte(torConfig), 0644)
			if err != nil {
				log.Error(fmt.Sprintf("Writing config failed with %v", err))
				return err
			}

			c.localManager.daemon.Reload()
		}

		// err = c.updateOnionBalancedServiceStatus(&onionBalancedService)
		// if err != nil {
		// 	log.Error(fmt.Sprintf("Updating status failed with %v", err))
		// 	return err
		// }
	}
	return nil
}

func (c *Controller) updateOnionBalancedServiceStatus(onionBalancedService *v1alpha2.OnionBalancedService) error {
	hostname, err := ioutil.ReadFile("/run/onionbalance/key/onionAddress")
	if err != nil {
		log.Error(fmt.Sprintf("Got this error when trying to find hostname: %v", err))
		return err
	}

	newHostname := strings.TrimSpace(string(hostname))

	if newHostname != onionBalancedService.Status.Hostname {
		log.Info(fmt.Sprintf("Got new hostname: %s", newHostname))
		onionBalancedService.Status.Hostname = newHostname

		log.Debug(fmt.Sprintf("Updating onionBalancedService to: %v", onionBalancedService))
		err = c.localManager.kclient.Status().Update(context.Background(), onionBalancedService)
		if err != nil {
			log.Error(fmt.Sprintf("Error updating onionBalancedService: %s", err))
			return err
		}
	}
	return nil
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		c.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		log.Error(fmt.Sprintf("Error syncing onionBalancedService %v: %v", key, err))

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		// c.queue.AddRateLimited(key)
		c.queue.AddAfter(key, 3*time.Second)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	log.Info(fmt.Sprintf("Dropping onionBalancedService %q out of the queue: %v", key, err))
}

func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	log.Info("Starting controller")

	go c.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Info("Stopping controller")
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}
