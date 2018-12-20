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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/runtime/schema"
	//"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	//appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	//appslisters "k8s.io/client-go/listers/apps/v1"
	//"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	logfiltersv1alpha1 "k8s.io/sample-controller/pkg/apis/logfilters/v1alpha1"
	clientset "k8s.io/sample-controller/pkg/client/clientset/versioned"
	logfilterscheme "k8s.io/sample-controller/pkg/client/clientset/versioned/scheme"
	informers "k8s.io/sample-controller/pkg/client/informers/externalversions/logfilters/v1alpha1"
	listers "k8s.io/sample-controller/pkg/client/listers/logfilters/v1alpha1"
)

const controllerAgentName = "logfilter-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a Foo is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Foo fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Logfilters"
	// MessageResourceSynced is the message used for an Event fired when a Foo
	// is synced successfully
	MessageResourceSynced = "Logfilter synced successfully"
)

// Controller is the controller implementation for Foo resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// sampleclientset is a clientset for our own API group
	logfilterclientset clientset.Interface

	//daemonSetsLister appslisters.DaemonSetLister
	//daemonSetSynced cache.InformerSynced
	logfilterLister        listers.FooLister
	logfiltersSynced        cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	fluentbitimage string
	namespace string
	currentfluentbitlua string
	labels map[string]string
}

// NewController returns a new sample controller
func NewController(
	kubeclientset kubernetes.Interface,
	logfilterclientset clientset.Interface,
	fooInformer informers.FooInformer,
	haproxyimage, proxydomain, proxynamespace, proxymode string, proxyport int,
	rancherhttpclient *rancherhttpclient.Rancherhttpclient) *Controller {

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	utilruntime.Must(samplescheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:     kubeclientset,
		logfilterclientset:   logfilterclientset,
		logfilterLister:        logfilterInformer.Lister(),
		logfiltersSynced:        logfilterInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Foos"),
		recorder:          recorder,
		fluentbitimage:      haproxyimage,
		namespace:    namespace,
		currentfluentbitlua: "",
		labels:  map[string]string{
			"app":        "cluster-api-proxy",
			"controller": "cluster-api-proxy-controller",
		},
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when Foo resources change
	logfilterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueLogfilter,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueLogfilter(new)
		},
		DeleteFunc: controller.enqueueLogfilter,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	listOptions := metav1.ListOptions{IncludeUninitialized: false}
	getOptions := metav1.GetOptions{IncludeUninitialized: false}

  // ConfigMap haproxy.cfg - Always update at starting controller
	logfilters, err := c.logfilterclientset.LogfiltersV1alpha1().Logfilters(c.namespace).List(listOptions)
	if err != nil {
		return fmt.Errorf("Failed to list Logfilter : " + err.Error())
	}
	for _, logfilter := range logfilters.Items {
		klog.Info("Logfilter : " + logfilter.ObjectMeta.Name)
	}
	lua := fluentbitcfg.MakeFluentbitIgnoreLua(logfilters)
	configmap, err := c.kubeclientset.CoreV1().ConfigMaps(c.proxynamespace).Get("fluentbit-lua", getOptions)
	newConfigMap := resources.NewConfigMap(lua, "fluentbit-lua", c.namespace, c.labels)
	if errors.IsNotFound(err) {
		configmap, err = c.kubeclientset.CoreV1().ConfigMaps(c.namespace).Create(newConfigMap)
	} else {
		configmap, err = c.kubeclientset.CoreV1().ConfigMaps(c.namespace).Update(newConfigMap)
	}
	if err != nil {
		return fmt.Errorf("Failed to create ConfigMap for fluent-bit lua script : " + err.Error())
	}
	c.currentfluentbitlua = configmap.Data["funcs.lua"]
	klog.Info("Current lua script : " + c.currentfluentbitlua)

  // Check haproxy daemonset and start when it's not found - Always update at starting controller
	_, err = c.kubeclientset.AppsV1().DaemonSets(c.namespace).Get("fluent-bit", getOptions)
	newDaemonSet = resources.NewDaemonSet(c.labels, "fluent-bit", c.namespace)
	if errors.IsNotFound(err) {
		_, err = c.kubeclientset.AppsV1().DaemonSets(c.proxynamespace).Create(newDaemonSet)
	} else {
		_, err = c.kubeclientset.AppsV1().DaemonSets(c.proxynamespace).Update(newDaemonSet)
	}
	if err != nil {
		return fmt.Errorf("Failed to deploy fluent-bit daemonset : " + err.Error())
	}

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Logfilter controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	//if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.foosSynced); !ok {
	if ok := cache.WaitForCacheSync(stopCh, c.logfiltersSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process Foo resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Foo resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the Proxy resource with this namespace/name.
	logfilter, err := c.logfiltersLister.Logfilters(namespace).Get(name)

	// Check deleted and update proxy setting.
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Info(key +" no longer exists. Update Logfilter.")
			err = c.updateLogfilter()
			if err != nil {
				return err
			}
			// Exit sync handler here when updating proxy for delete.
			return nil
		} else {
			return err
		}
	}

	// Check if foo is set in haproxy.cfg and ready, then update proxy.
	if !fluentbitcfg.IsValidInIgnoreLua(Logfilter) {
		err = c.updateLogfilter()
		if err != nil {
			return err
		}
	}

	// Finally, we update the status block of the Foo resource to reflect the
	// current state of the world
  err = c.updateLogfilterStatus(logfilter, "Configured")
	if err != nil {
		return err
	}

	c.recorder.Event(foo, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *Controller) updateLogfilterStatus(logfilter *logfiltersv1alpha1.Logfilter, phase string) error {
	logfilterCopy := logfilter.DeepCopy()
	logfilterCopy.Status.Phase = phase
	_, err := c.logfilterclientset.LogfiltersV1alpha1().Logfilters(logfilter.Namespace).Update(logfilterCopy)
	return err
}

// enqueueFoo takes a Foo resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Foo.
func (c *Controller) enqueueLogfilter(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// Rewrite haproxy.cfg and reload haproxy when Proxies added/edited/delated
func (c *Controller) updateLogfilter() error {
	// Get All Proxy
	listOptions := metav1.ListOptions{IncludeUninitialized: false}
	logfilter, err := c.logfilterclientset.LogfiltersV1alpha1().Logfilters(c.namespace).List(listOptions)
	if err != nil {
		return fmt.Errorf("Failed to list Logfilter : " + err.Error())
	}

	// Update Configmap haproxy.cfg
	lua := haproxycfg.MakeFluentbitIgnoreLua(logfilters)
	configmap, err := c.kubeclientset.CoreV1().ConfigMaps(c.proxynamespace).Update(
		resources.NewConfigMap(lua, "fluentbit-lua", c.namespace, c.labels))
	if err != nil {
		return fmt.Errorf("Failed to update ConfigMap for fluent-bit lua script : " + err.Error())
	}
	c.currentfluentbitlua = configmap.Data["funcs.lua"]
	klog.Info("Current fluent-bit lua script : " + c.currentffluentbitlua)

	// Restert haproxy daemonset
	_, err = c.kubeclientset.AppsV1().DaemonSets(c.namespace).Update(
		resources.NewDaemonSet(c.labels, "fluent-bit", c.namespace, c.fluentbitimage))
	if err != nil {
		return fmt.Errorf("Failed to restart fluent-bit daemonset : " + err.Error())
	}
	return err
}
