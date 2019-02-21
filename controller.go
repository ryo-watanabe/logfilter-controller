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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	"github.com/ryo-watanabe/logfilter-controller/pkg/fluentbitcfg"
	"github.com/ryo-watanabe/logfilter-controller/pkg/resources"
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
	kubeclientset kubernetes.Interface
	fluentbitimage string
	namespace string
	currentfluentbitlua string
	labels map[string]string
}

// NewController returns a new sample controller
func NewController(
	kubeclientset kubernetes.Interface,
	fluentbitimage, namespace string) *Controller {

	controller := &Controller{
		kubeclientset: kubeclientset,
		fluentbitimage: fluentbitimage,
		namespace: namespace,
		currentfluentbitlua: "",
		labels: map[string]string{
			"app":        "fluent-bit",
			"controller": "logfilter-controller",
		},
	}

	return controller
}

// Run will set up the event handlers for logfilters.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting Logfilter controller")

  // Initial sync for lua-configmap and daemonset
	err := c.syncHandler()
	if err != nil {
		return err
	}

	klog.Info("Starting sync handler")
	// Launch worker to process logfilter configmaps
	go wait.Until(c.runWorker, time.Second*30, stopCh)

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

func (c *Controller) runWorker() {
	err := c.syncHandler()
	if err != nil {
		runtime.HandleError(err)
	}
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two.
func (c *Controller) syncHandler() error {

	// Load current logfilters
	selector := &metav1.LabelSelector{MatchLabels: map[string]string{"logfilter.ssl.com/filterdata": "true"}}
	listOptions := metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(selector)}
	logfilters, err := c.kubeclientset.CoreV1().ConfigMaps(c.namespace).List(listOptions)
	if err != nil {
		return fmt.Errorf("Failed to list ConfigMaps : " + err.Error())
	}
	lua := fluentbitcfg.MakeFluentbitIgnoreLua(logfilters)

  if lua["funcs.lua"] != c.currentfluentbitlua {
		klog.Info("Updating logfilter")

		// Update lua configmap
		getOptions := metav1.GetOptions{IncludeUninitialized: false}
		configmap, err := c.kubeclientset.CoreV1().ConfigMaps(c.namespace).Get("fluentbit-lua", getOptions)
		newConfigMap := resources.NewConfigMap("fluentbit-lua", c.namespace, lua)
		if errors.IsNotFound(err) {
			configmap, err = c.kubeclientset.CoreV1().ConfigMaps(c.namespace).Create(newConfigMap)
		} else {
			configmap, err = c.kubeclientset.CoreV1().ConfigMaps(c.namespace).Update(newConfigMap)
		}
		if err != nil {
			return fmt.Errorf("Failed to create/update ConfigMap for fluent-bit lua script : " + err.Error())
		}
		c.currentfluentbitlua = configmap.Data["funcs.lua"]
		//klog.Info("Current lua script : " + c.currentfluentbitlua)

	  // Check haproxy daemonset and start when it's not found - Always update at starting controller
		_, err = c.kubeclientset.AppsV1().DaemonSets(c.namespace).Get("fluent-bit", getOptions)
		newDaemonSet := resources.NewDaemonSet(c.labels, "fluent-bit", c.namespace, c.fluentbitimage)
		if errors.IsNotFound(err) {
			_, err = c.kubeclientset.AppsV1().DaemonSets(c.namespace).Create(newDaemonSet)
		} else {
			_, err = c.kubeclientset.AppsV1().DaemonSets(c.namespace).Update(newDaemonSet)
		}
		if err != nil {
			return fmt.Errorf("Failed to deploy/update fluent-bit daemonset : " + err.Error())
		}
	}

	return nil
}
