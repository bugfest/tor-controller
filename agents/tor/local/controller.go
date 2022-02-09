package local

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
	"time"

	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	config "github.com/bugfest/tor-controller/agents/tor/config"
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
		log.Warn(fmt.Sprintf("OnionService %s does not exist anymore", key))
	} else {
		log.Debug(fmt.Sprintf("%v", obj))
		onionService, err := parseOnionService(obj)
		if err != nil {
			log.Error(fmt.Sprintf("Error in parseOnionService: %s", err))
			return err
		}

		// torfile
		torConfig, err := config.TorConfigForService(&onionService)
		if err != nil {
			log.Error(fmt.Sprintf("Generating config failed with %v", err))
			return err
		}

		reload := false

		torfile, err := ioutil.ReadFile("/run/tor/torfile")
		if os.IsNotExist(err) {
			reload = true
		} else if err != nil {
			return err
		} else if string(torfile) != torConfig {
			reload = true
		}

		if reload {
			log.Info(fmt.Sprintf("Updating tor config for %s/%s", onionService.Namespace, onionService.Name))

			err = ioutil.WriteFile("/run/tor/torfile", []byte(torConfig), 0644)
			if err != nil {
				log.Error(fmt.Sprintf("Writing config failed with %v", err))
				return err
			}

			c.localManager.daemon.Reload()
		}

		// update hostname
		copyIfNotExist(
			"/run/tor/service/key/hostname",
			"/run/tor/service/hostname",
		)

		// update private and public keys
		publicKeyFileName := "hs_ed25519_public_key"
		privateKeyFileName := "hs_ed25519_secret_key"
		if onionService.Spec.GetVersion() == 2 {
			publicKeyFileName = "public_key"
			privateKeyFileName = "private_key"
		}
		copyIfNotExist(
			fmt.Sprintf("/run/tor/service/key/%s", publicKeyFileName),
			fmt.Sprintf("/run/tor/service/%s", publicKeyFileName),
		)
		copyIfNotExist(
			fmt.Sprintf("/run/tor/service/key/%s", privateKeyFileName),
			fmt.Sprintf("/run/tor/service/%s", privateKeyFileName),
		)

		// ob_config needs to be created if this Hidden Service have a Master one in front
		if len(onionService.Spec.MasterOnionAddress) > 0 {
			obConfig, err := config.ObConfigForService(&onionService)
			if err != nil {
				log.Error(fmt.Sprintf("Generating ob_config failed with %v", err))
				return err
			}

			reload = false

			obfile, err := ioutil.ReadFile("/run/tor/service/ob_config")
			if os.IsNotExist(err) {
				reload = true
			} else if err != nil {
				return err
			} else if string(obfile) != obConfig {
				reload = true
			}

			if reload {
				log.Info(fmt.Sprintf("Updating onionbalance config for %s/%s", onionService.Namespace, onionService.Name))

				err = ioutil.WriteFile("/run/tor/service/ob_config", []byte(obConfig), 0644)
				if err != nil {
					log.Error(fmt.Sprintf("Writing config failed with %v", err))
					return err
				}

				c.localManager.daemon.Reload()
			}
		}

		err = c.updateOnionServiceStatus(&onionService)
		if err != nil {
			log.Error(fmt.Sprintf("Updating status failed with %v", err))
			return err
		}
	}
	return nil
}

// Generates ob_config file if this instance handles traffic on behalf of a master hidden service
func (c *Controller) onionBalanceConfig(onionService *v1alpha2.OnionService) string {
	const configFormat = `MasterOnionAddress {{.Spec.MasterOnionAddress}}`

	var configTemplate = template.Must(template.New("config").Parse(configFormat))
	var tmp bytes.Buffer
	err := configTemplate.Execute(&tmp, onionService)
	if err != nil {
		return ""
	}
	return tmp.String()
}

func (c *Controller) updateOnionServiceStatus(onionService *v1alpha2.OnionService) error {
	hostname, err := ioutil.ReadFile("/run/tor/service/hostname")
	if err != nil {
		log.Error(fmt.Sprintf("Got this error when trying to find hostname: %v", err))
		return err
	}

	newHostname := strings.TrimSpace(string(hostname))

	if newHostname != onionService.Status.Hostname {
		log.Info(fmt.Sprintf("Got new hostname: %s", newHostname))
		onionService.Status.Hostname = newHostname

		log.Debug(fmt.Sprintf("Updating onionService to: %v", onionService))
		err = c.localManager.kclient.Status().Update(context.Background(), onionService)
		if err != nil {
			log.Error(fmt.Sprintf("Error updating onionService: %s", err))
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
		log.Error(fmt.Sprintf("Error syncing onionservice %v: %v", key, err))

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		// c.queue.AddRateLimited(key)
		c.queue.AddAfter(key, 3*time.Second)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	log.Info(fmt.Sprintf("Dropping onionservice %q out of the queue: %v", key, err))
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

func copyIfNotExist(src string, dst string) error {
	_, err := ioutil.ReadFile(dst)
	if os.IsNotExist(err) {
		log.Info(fmt.Sprintf("Creating copy of %s at %s", src, dst))

		var err error
		var srcfd *os.File
		var dstfd *os.File
		var srcinfo os.FileInfo

		if srcfd, err = os.Open(src); err != nil {
			return err
		}
		defer srcfd.Close()

		if dstfd, err = os.Create(dst); err != nil {
			return err
		}
		defer dstfd.Close()

		if _, err = io.Copy(dstfd, srcfd); err != nil {
			return err
		}
		if srcinfo, err = os.Stat(src); err != nil {
			return err
		}
		return os.Chmod(dst, srcinfo.Mode())
	}
	return nil
}
