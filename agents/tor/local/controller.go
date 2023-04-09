package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	log "github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	config "github.com/bugfest/tor-controller/agents/tor/config"
	v1alpha2 "github.com/bugfest/tor-controller/apis/tor/v1alpha2"
)

const (
	authorizedClientsDir  = "/run/tor/service/authorized_clients"
	torFilePath           = "/run/tor/torfile"
	torServiceDir         = "/run/tor/service/"
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

//nolint:gocognit,nestif // this function is long for a reason
func (c *Controller) sync(key string) error {
	log.Infof("Getting key %s", key)

	// Get the OnionService resource with this key
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		log.Errorf("Fetching object with key %s from store failed with %v", key, err)

		return errors.Wrap(err, "fetching object from store")
	}

	if !exists {
		log.Warnf("OnionService %s does not exist anymore", key)

		return nil
	}

	log.Debugf("%v", obj)

	onionService, err := parseOnionService(obj)
	if err != nil {
		log.Error(fmt.Sprintf("Error in parseOnionService: %s", err))

		return errors.Wrap(err, "parsing onion service")
	}

	// torfile
	torConfig, err := config.TorConfigForService(&onionService)
	if err != nil {
		log.Error(fmt.Sprintf("Generating config failed with %v", err))

		return errors.Wrap(err, "generating config")
	}

	reload := false

	torfile, err := os.ReadFile(torFilePath)

	switch {
	case os.IsNotExist(err):
		reload = true
	case err != nil:
		return errors.Wrap(err, "reading torfile")
	case string(torfile) != torConfig:
		reload = true
	}

	if reload {
		err = serviceReload(&onionService, []byte(torConfig))
		if err != nil {
			log.Error(fmt.Sprintf("Reloading service failed with %v", err))

			return errors.Wrap(err, "reloading service")
		}
	}

	// update hostname
	err = copyIfNotExist(
		"/run/tor/service/key/hostname",
		"/run/tor/service/hostname",
	)
	if err != nil {
		log.Errorf("Updating hostname failed with %v", err)
	}

	// update private and public keys
	publicKeyFileName := "hs_ed25519_public_key"
	privateKeyFileName := "hs_ed25519_secret_key"

	if onionService.Spec.GetVersion() == 2 {
		publicKeyFileName = "public_key"
		privateKeyFileName = "private_key"
	}

	err = copyIfNotExist(
		path.Join(torServiceDir, "key", publicKeyFileName),
		path.Join(torServiceDir, publicKeyFileName),
	)
	if err != nil {
		log.Errorf("Updating public key failed with %v", err)
	}

	err = copyIfNotExist(
		path.Join(torServiceDir, "key", privateKeyFileName),
		path.Join(torServiceDir, privateKeyFileName),
	)
	if err != nil {
		log.Errorf("Updating private key failed with %v", err)
	}

	// copy authorized keys to the correct directory (/run/tor/service/authorized_keys)
	// as Tor requires this directory to be only accessible for the current user (0700)
	// and k8s does not allow to set the permissions of the directory where the projected
	// secrets are mounted
	files, err := os.ReadDir("/run/tor/service/.authorized_clients/")
	if err != nil {
		log.Info("No authorized keys found")
	} else {
		// Create `authorized_clients_dir` directory if it does not exist
		if _, err := os.Stat(authorizedClientsDir); errors.Is(err, os.ErrNotExist) {
			err := os.Mkdir(authorizedClientsDir, os.ModePerm)
			if err != nil {
				log.Fatalf("Creating directory %s failed with %v", authorizedClientsDir, err)
			}
		}

		// Copy *.auth files from mounted secrets into `authorized_clients_dir` directory
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".auth") {
				err = copyIfNotExist(
					path.Join("/run/tor/service/.authorized_clients/", file.Name()),
					path.Join("/run/tor/service/authorized_clients/", file.Name()),
				)
				if err != nil {
					log.Errorf("Copying authorized keys failed with %v", err)
				}
			}
		}
	}

	// ob_config needs to be created if this Hidden Service have a Master one in front
	if len(onionService.Spec.MasterOnionAddress) > 0 {
		obConfig, err := config.ObConfigForService(&onionService)
		if err != nil {
			log.Error(fmt.Sprintf("Generating ob_config failed with %v", err))

			return errors.Wrap(err, "generating ob_config")
		}

		obfile, err := os.ReadFile("/run/tor/service/ob_config")

		switch {
		case os.IsNotExist(err):
			reload = true
		case err != nil:
			return errors.Wrap(err, "reading ob_config")
		case string(obfile) != obConfig:
			reload = true
		}

		if reload {
			log.Infof("Updating onionbalance config for %s/%s", onionService.Namespace, onionService.Name)

			err = os.WriteFile("/run/tor/service/ob_config", []byte(obConfig), defaultUnixPermission)
			if err != nil {
				log.Error(fmt.Sprintf("Writing config failed with %v", err))

				return errors.Wrap(err, "writing ob_config")
			}
		}
	}

	if reload {
		c.localManager.daemon.Reload()
	}

	err = c.updateOnionServiceStatus(&onionService)
	if err != nil {
		log.Error(fmt.Sprintf("Updating status failed with %v", err))

		return errors.Wrap(err, "updating status")
	}

	return nil
}

func (c *Controller) updateOnionServiceStatus(onionService *v1alpha2.OnionService) error {
	hostname, err := os.ReadFile("/run/tor/service/hostname")
	if err != nil {
		log.Error(fmt.Sprintf("Got this error when trying to find hostname: %v", err))

		return errors.Wrap(err, "reading hostname")
	}

	newHostname := strings.TrimSpace(string(hostname))

	if newHostname != onionService.Status.Hostname {
		log.Infof("Got new hostname: %s", newHostname)
		onionService.Status.Hostname = newHostname

		log.Debug(fmt.Sprintf("Updating onionService to: %v", onionService))

		err = c.localManager.kclient.Status().Update(context.Background(), onionService)
		if err != nil {
			log.Error(fmt.Sprintf("Error updating onionService: %s", err))

			return errors.Wrap(err, "updating onionService")
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
	//nolint:gomnd // 5 is a reasonable number of retries
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
	log.Infof("Dropping onionservice %q out of the queue: %v", key, err)
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

func copyIfNotExist(src, dst string) error {
	_, err := os.Stat(dst)
	if !os.IsNotExist(err) {
		return errors.Wrap(err, "checking if file exists")
	}

	log.Infof("Creating copy of %s at %s", src, dst)

	srcfd, err := os.Open(src)
	if err != nil {
		return errors.Wrap(err, "opening source file")
	}
	defer srcfd.Close()

	dstfd, err := os.Create(dst)
	if err != nil {
		return errors.Wrap(err, "creating destination file")
	}
	defer dstfd.Close()

	_, err = io.Copy(dstfd, srcfd)
	if err != nil {
		return errors.Wrap(err, "copying file")
	}

	srcinfo, err := os.Stat(src)
	if err != nil {
		return errors.Wrap(err, "getting source file info")
	}

	err = dstfd.Chmod(srcinfo.Mode())
	if err != nil {
		return errors.Wrap(err, "setting destination file mode")
	}

	return nil
}

func serviceReload(onionService *v1alpha2.OnionService, configData []byte) error {
	log.Infof("Updating tor config for %s/%s", onionService.Namespace, onionService.Name)

	err := os.WriteFile(torFilePath, configData, defaultUnixPermission)
	if err != nil {
		log.Errorf("Writing config failed with %v", err)

		return errors.Wrap(err, "writing config")
	}

	return nil
}
