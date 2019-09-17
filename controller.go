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
	corev1 "k8s.io/api/core/v1"
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
	metricsimage string
	registrykey string
	kafkasecret string
	kafkasecretpath string
	namespace string
	currentfluentbitlua string
	currentfluentbitmetricconfig string
	currentfluentbitconfig map[string]string
	labels map[string]string
	metricslabels map[string]string
}

// NewController returns a new sample controller
func NewController(
	kubeclientset kubernetes.Interface,
	fluentbitimage, metricsimage, registrykey, kafkasecret, kafkasecretpath, namespace string) *Controller {

	controller := &Controller{
		kubeclientset: kubeclientset,
		fluentbitimage: fluentbitimage,
		metricsimage: metricsimage,
		registrykey: registrykey,
		kafkasecret: kafkasecret,
		kafkasecretpath: kafkasecretpath,
		namespace: namespace,
		currentfluentbitlua: "",
		currentfluentbitmetricconfig: "",
		currentfluentbitconfig: map[string]string{},
		labels: map[string]string{
			"app":        "fluent-bit",
			"controller": "logfilter-controller",
		},
		metricslabels: map[string]string{
			"app":        "fluent-bit-metrics",
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
	//klog.Info("Starting sync handler")
	err := c.syncHandler()
	if err != nil {
		runtime.HandleError(err)
	}
}

func (c *Controller) loadConfigMaps(label string) (*corev1.ConfigMapList, error) {
	selector := &metav1.LabelSelector{MatchLabels: map[string]string{label: "true"}}
	listOptions := metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(selector)}
	configmaps, err := c.kubeclientset.CoreV1().ConfigMaps(c.namespace).List(listOptions)
	if err != nil {
		return nil, fmt.Errorf("Failed to list ConfigMaps : " + err.Error())
	}
	return configmaps, err
}

func (c *Controller) updateConfigMap(name string, data map[string]string) (*corev1.ConfigMap, error) {
	klog.Info("Updating ConfigMap : " + name)
	getOptions := metav1.GetOptions{IncludeUninitialized: false}
	configmap, err := c.kubeclientset.CoreV1().ConfigMaps(c.namespace).Get(name, getOptions)
	newConfigMap := resources.NewConfigMap(name, c.namespace, data)
	if errors.IsNotFound(err) {
		configmap, err = c.kubeclientset.CoreV1().ConfigMaps(c.namespace).Create(newConfigMap)
	} else {
		configmap, err = c.kubeclientset.CoreV1().ConfigMaps(c.namespace).Update(newConfigMap)
	}
	if err != nil {
		return nil, fmt.Errorf("Failed to create/update ConfigMap " + name + " : " + err.Error())
	}
	return configmap, err
}

func (c *Controller) updateDaemonSet(name, config_name string, nodegroup map[string]string) error {
	tolerations := ""
	node_selector := ""
	if val, ok := nodegroup["tolerations"]; ok {
		tolerations = val
	}
	if val, ok := nodegroup["node_selector"]; ok {
		node_selector = val
	}

	getOptions := metav1.GetOptions{IncludeUninitialized: false}
	_, err := c.kubeclientset.AppsV1().DaemonSets(c.namespace).Get(name, getOptions)
	newDaemonSet := resources.NewDaemonSet(c.labels, name, c.namespace,
		c.fluentbitimage, c.kafkasecret, c.kafkasecretpath, c.registrykey, tolerations, node_selector, config_name)
	if errors.IsNotFound(err) {
		_, err = c.kubeclientset.AppsV1().DaemonSets(c.namespace).Create(newDaemonSet)
	} else {
		_, err = c.kubeclientset.AppsV1().DaemonSets(c.namespace).Update(newDaemonSet)
	}
	if err != nil {
		return fmt.Errorf("Failed to deploy/update fluent-bit daemonset : " + err.Error())
	}
	return err
}

func (c *Controller) updateDeployment(name, config_name string) error {
	getOptions := metav1.GetOptions{IncludeUninitialized: false}
	_, err := c.kubeclientset.AppsV1().Deployments(c.namespace).Get(name, getOptions)
	newDeployment := resources.NewDeployment(c.metricslabels, name, c.namespace, c.metricsimage, c.kafkasecret,
		c.kafkasecretpath, c.registrykey, config_name)
	if errors.IsNotFound(err) {
		_, err = c.kubeclientset.AppsV1().Deployments(c.namespace).Create(newDeployment)
	} else {
		_, err = c.kubeclientset.AppsV1().Deployments(c.namespace).Update(newDeployment)
	}
	if err != nil {
		return fmt.Errorf("Failed to deploy/update fluent-bit-metrics deployment : " + err.Error())
	}
	return err
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two.
func (c *Controller) syncHandler() error {

	// Load/update logfilters
	logfilters, err := c.loadConfigMaps("logfilter.ssl.com/filterdata")
	if err != nil {
		return err
	}
	lua := fluentbitcfg.MakeFluentbitIgnoreLua(logfilters)
  	luaupdated := false
	if lua["funcs.lua"] != c.currentfluentbitlua {
		configmap, err := c.updateConfigMap("fluentbit-lua", lua)
		if err != nil {
			return err
		}
		c.currentfluentbitlua = configmap.Data["funcs.lua"]
		luaupdated = true
		klog.Info("Logfilter updated.")
	}

  	// Load/update configmaps
	nodegroups, err := c.loadConfigMaps("logfilter.ssl.com/nodegroup")
	if err != nil {
		return err
	}
	logs, err := c.loadConfigMaps("logfilter.ssl.com/log")
	if err != nil {
		return err
	}
	procs, err := c.loadConfigMaps("logfilter.ssl.com/proc")
	if err != nil {
		return err
	}
	os_monits, err := c.loadConfigMaps("logfilter.ssl.com/os")
	if err != nil {
		return err
	}
	metrics, err := c.loadConfigMaps("logfilter.ssl.com/metric")
	if err != nil {
		return err
	}
	apps, err := c.loadConfigMaps("logfilter.ssl.com/app")
	if err != nil {
		return err
	}
	outputs, err := c.loadConfigMaps("logfilter.ssl.com/es")
	if err != nil {
		return err
	}
	kafkas, err := c.loadConfigMaps("logfilter.ssl.com/kafka")
	if err != nil {
		return err
	}

	// load template
	templateConfigMap, err := c.kubeclientset.CoreV1().ConfigMaps(c.namespace).Get("templates", metav1.GetOptions{})
	if err != nil {
		return err
	}
	if templateConfigMap.Data["daemonset_fluent-bit.conf"] == "" {
		return fmt.Errorf("Cannot find 'daemonset_fluent-bit.conf' in ConfigMap 'templates'")
	}
	if templateConfigMap.Data["deployment_fluent-bit.conf"] == "" {
		return fmt.Errorf("Cannot find 'deployment_fluent-bit.conf' in ConfigMap 'templates'")
	}

  	// Log, Proc fluent-bit daemonsets for node groups
	for _, nodegroup := range nodegroups.Items {

		// make config
		group := nodegroup.ObjectMeta.Name
		cfg := fluentbitcfg.MakeFluentbitConfig(templateConfigMap.Data["daemonset_fluent-bit.conf"],
			logs, procs, os_monits, outputs, kafkas, group)

		configupdated := false
		_, ok := c.currentfluentbitconfig[group]
		if !ok || cfg["fluent-bit.conf"] != c.currentfluentbitconfig[group] {
			configmap, err := c.updateConfigMap("fluentbit-config-" + group, cfg)
			if err != nil {
				return err
			}
			c.currentfluentbitconfig[group] = configmap.Data["fluent-bit.conf"]
			configupdated = true
		}

		if luaupdated || configupdated {
			klog.Info("Updating daemonset.")
			err = c.updateDaemonSet("fluent-bit-" + group, "fluentbit-config-" + group, nodegroup.Data)
			if err != nil {
				return err
			}
		}
	}

  	// Metrics fluent-bit deployment.
	metricscfg := fluentbitcfg.MakeFluentbitMetricsConfig(templateConfigMap.Data["deployment_fluent-bit.conf"],
		metrics, apps, outputs, kafkas)
	if metricscfg["fluent-bit.conf"] != c.currentfluentbitmetricconfig {
		configmap, err := c.updateConfigMap("fluentbit-metrics-config", metricscfg)
		if err != nil {
			return err
		}
		c.currentfluentbitmetricconfig = configmap.Data["fluent-bit.conf"]
		klog.Info("Updating metrics deployment.")
		err = c.updateDeployment("fluent-bit-metrics", "fluentbit-metrics-config")
		if err != nil {
			return err
		}
	}

	return nil
}
