package local

import (
	"os"
	"time"

	"github.com/cockroachdb/errors"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	config "github.com/bugfest/tor-controller/agents/onionbalance/config"
)

const (
	defaultUnixPermission = 0o600
)

type Controller struct {
	indexer      cache.Indexer
	queue        workqueue.RateLimitingInterface
	informer     cache.Controller
	localManager *Manager
}

func NewController(queue workqueue.RateLimitingInterface, informer cache.SharedIndexInformer, localManager *Manager) *Controller {
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

	keyString, ok := key.(string)
	if !ok {
		log.Errorf("Key is not a string: %v", key)

		return false
	}

	err := c.sync(keyString)
	c.handleErr(err, key)

	return true
}

func (c *Controller) sync(key string) error {
	log.Infof("Getting key %s", key)

	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		log.Errorf("Fetching object with key %s from store failed with %v", key, err)

		return errors.Wrapf(err, "fetching object with key %s from store failed", key)
	}

	if !exists {
		log.Warnf("onionBalancedService %s does not exist anymore", key)

		return nil
	}

	log.Debugf("%v", obj)

	onionBalancedService, err := parseOnionBalancedService(obj)
	if err != nil {
		log.Errorf("Error in parseonionBalancedService: %s", err)

		return errors.Wrapf(err, "error in parseonionBalancedService")
	}

	torConfig, err := config.OnionBalanceConfigForService(&onionBalancedService)
	if err != nil {
		log.Errorf("Generating config failed with %v", err)

		return errors.Wrapf(err, "generating config failed")
	}

	torfile, err := os.ReadFile("/run/onionbalance/config.yaml")
	if err != nil && !os.IsNotExist(err) {
		log.Errorf("Failed to read config file: %v", err)

		return errors.Wrapf(err, "failed to read config file")
	}

	if string(torfile) != torConfig {
		// Configuration has changed, save new configs and reload the daemon.
		log.Infof("Updating onionbalance config for %s/%s", onionBalancedService.Namespace, onionBalancedService.Name)

		err = os.WriteFile("/run/onionbalance/config.yaml", []byte(torConfig), defaultUnixPermission)
		if err != nil {
			log.Errorf("Writing config failed with %v", err)

			return errors.Wrapf(err, "writing config failed")
		}

		c.localManager.daemon.Reload()
	} else {
		// Config was already set correctly, lets just ensure the daemon is (still) running.
		c.localManager.daemon.EnsureRunning()
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
	//nolint:gomnd // just tries
	if c.queue.NumRequeues(key) < 5 {
		log.Errorf("Error syncing onionBalancedService %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		// c.queue.AddRateLimited(key)
		//nolint:mnd // just seconds
		c.queue.AddAfter(key, 3*time.Second)

		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	log.Infof("Dropping onionBalancedService %q out of the queue: %v", key, err)
}

func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	log.Info("Starting controller")

	go c.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(errors.New("timed out waiting for caches to sync"))

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
