/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"

	datasetsv1 "github.com/nerdalize/nerd/crd/pkg/apis/stable.nerdalize.com/v1"
	clientset "github.com/nerdalize/nerd/crd/pkg/client/clientset/versioned"
	informers "github.com/nerdalize/nerd/crd/pkg/client/informers/externalversions"
	listers "github.com/nerdalize/nerd/crd/pkg/client/listers/stable.nerdalize.com/v1"
)

const (
	maxRetries = 5
)

// Controller is the controller implementation for Dataset resources
type Controller struct {
	// nerdalizeclientset is a clientset for our own API group
	nerdalizeclientset clientset.Interface
	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue      workqueue.RateLimitingInterface
	informer       cache.SharedIndexInformer
	datasetsLister listers.DatasetLister
	eventHandler   Handler
}

// NewController returns a new sample controller
func NewController(
	nerdalizeclientset clientset.Interface,
	datasetInformerFactory informers.SharedInformerFactory,
	eventHandler Handler) *Controller {

	glog.Info("Creating controller")

	// obtain references to shared index informers for the Dataset types.
	datasetInformer := datasetInformerFactory.Nerdalize().V1().Datasets()
	queue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Datasets")

	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return nerdalizeclientset.NerdalizeV1().Datasets(metav1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return nerdalizeclientset.NerdalizeV1().Datasets(metav1.NamespaceAll).Watch(options)
			},
		},
		&datasetsv1.Dataset{},
		0, //Skip resync
		cache.Indexers{},
	)

	controller := &Controller{
		nerdalizeclientset: nerdalizeclientset,
		datasetsLister:     datasetInformer.Lister(),
		informer:           informer,
		workqueue:          queue,
		eventHandler:       eventHandler,
	}

	glog.Info("Setting up event handlers")
	// Set up an event handler for when Dataset resources change
	datasetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	})

	return controller
}

// Run starts the dataset controller
func (c *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	glog.Info("Starting dataset controller")

	go c.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	glog.Info("Dataset controller synced and ready")

	wait.Until(c.runWorker, time.Second, stopCh)
}

// HasSynced is required for the cache.Controller interface.
func (c *Controller) HasSynced() bool {
	return c.informer.HasSynced()
}

// LastSyncResourceVersion is required for the cache.Controller interface.
func (c *Controller) LastSyncResourceVersion() string {
	return c.informer.LastSyncResourceVersion()
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
		// continue looping
	}
}

func (c *Controller) processNextItem() bool {
	key, quit := c.workqueue.Get()

	if quit {
		return false
	}
	defer c.workqueue.Done(key)

	err := c.processItem(key.(string), "datasets")
	if err == nil {
		// No error, reset the ratelimit counters
		c.workqueue.Forget(key)
	} else if c.workqueue.NumRequeues(key) < maxRetries {
		glog.Errorf("Error processing %s (will retry): %v", key, err)
		c.workqueue.AddRateLimited(key)
	} else {
		// err != nil and too many retries
		glog.Errorf("Error processing %s (giving up): %v", key, err)
		c.workqueue.Forget(key)
		utilruntime.HandleError(err)
	}

	return true
}

func (c *Controller) processItem(key string, kobj string) error {
	glog.Infof("Processing change to %v: %s", kobj, key)

	obj, exists, err := c.informer.GetIndexer().GetByKey(key)
	glog.Info(obj)
	if err != nil {
		return fmt.Errorf("Error fetching object with key %s from store: %v", key, err)
	}

	if !exists {
		glog.Info("calling object deleted")
		c.eventHandler.ObjectDeleted(obj)
		return nil
	}

	glog.Info("calling object created")
	c.eventHandler.ObjectCreated(obj)
	return nil
}
