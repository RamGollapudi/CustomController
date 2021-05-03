package controller

import (
	"fmt"
	"sync"
	"time"

	deploymentconfigv1client "github.com/openshift/client-go/apps/clientset/versioned"
	gocache "github.com/patrickmn/go-cache"
	"gitscm.cisco.com/scm/eps-kube/cae-route-controller/pkg/errortypes"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	//podv1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

var (
	maxRetries = 10
)

type Controller struct {
	DeploymentConfigClient   *deploymentconfigv1client.Clientset
	KubeClient               *kubernetes.Clientset
	DeploymentConfigInformer cache.SharedIndexInformer
	DeploymentInformer       cache.SharedIndexInformer
	PodInformer              cache.SharedIndexInformer
	DeploymentConfigQueue    workqueue.RateLimitingInterface
	DeploymentQueue          workqueue.RateLimitingInterface
	PodQueue                 workqueue.RateLimitingInterface
	NamespaceLister          v1.NamespaceLister
	Gocache                  *gocache.Cache
	//PodClient        *podv1client.CoreV1Client
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threads int, stopCh <-chan struct{}) error {

	var (
		waitGroup sync.WaitGroup
	)

	defer utilruntime.HandleCrash()
	defer c.PodQueue.ShutDown()

	klog.Infof("Starting Pod Controller")
	if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	klog.Infof("Cache sync complete")

	// run 'threads' number of workers to process Route resources
	for i := 0; i < threads; i++ {
		createWorker(c.PodQueue, c.processPod, stopCh, &waitGroup)
		//createWorker(c.GlobalQueue, c.processGlobalRoute, stopCh, &waitGroup)
	}

	klog.Infof("Started Pod workers")
	<-stopCh
	klog.Infof("Shutting down workers")
	waitGroup.Wait()
	return nil
}

// HasSynced allows us to satisfy the Controller interface
// by wiring up the informer's HasSynced method to it
func (c *Controller) HasSynced() bool {
	return c.PodInformer.HasSynced()
}

// createWorker creates and runs a worker thread that just processes items in the
// specified queue. The worker will run until stopCh is closed. The worker will be
// added to the wait group when started and marked done when finished.
func createWorker(queue workqueue.RateLimitingInterface, reconciler func(key string) error, stopCh <-chan struct{}, waitGroup *sync.WaitGroup) {
	waitGroup.Add(1)
	go func() {
		wait.Until(runWorker(queue, reconciler), time.Second, stopCh)
		waitGroup.Done()
	}()
}

// runWorker retrieves each queued item and takes the necessary
// handler action based off if the item was created, updated, or deleted
func runWorker(queue workqueue.RateLimitingInterface, processItem func(key string) error) func() {
	return func() {
		running := true
		for running {
			running = func() bool {
				key, quit := queue.Get()

				// stop the worker loop if shutdown message was placed in queue
				if quit {
					return false
				}

				defer queue.Done(key)
				klog.Infof("Calling Deployment processItem")
				err := processItem(key.(string))

				if err == nil {
					// No error so tell the queue to stop tracking history
					queue.Forget(key)
				} else if _, ok := err.(*errortypes.NonRetryableError); ok {
					klog.Errorf("nonRetryableError with message: \"%v\"", err)
					queue.Forget(key)
				} else if queue.NumRequeues(key) < maxRetries {
					klog.Infof("Retrying - failed with message: \"%v\"", err)
					// requeue the item to work on later
					queue.AddRateLimited(key)
				} else {
					// err != nil and too many retries
					klog.Errorf("Exhausted all %v retries for route %s with error: \"%v\"", maxRetries, key, err)
					queue.Forget(key)
					utilruntime.HandleError(err)
				}

				return true
			}()
		}
	}
}

func (c *Controller) processPod(key string) error {
	//klog.Infof("Inside processPod")
	//klog.Infof("Key - %s", key)
	obj, exists, err := c.PodInformer.GetIndexer().GetByKey(key)
	if err != nil {
		return fmt.Errorf("Error fetching object with key %s from cache: %v", key, err)
	}
	if !exists {

	}
	// no need to differentiate between creates and updates
	//klog.Infof("Calling UpdatePod")
	if err := c.UpdatePod(obj, false); err != nil {
		return err
	}

	return nil
}
