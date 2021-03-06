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
	"flag"
	"k8s.io/klog"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ryo-watanabe/logfilter-controller/pkg/signals"
)

var (
	masterURL  string
	kubeconfig string
	fluentbitimage string
	metricsimage string
	registrykey string
	kafkasecret string
	kafkasecretpath string
	namespace string
)

func main() {
	klog.InitFlags(nil)
	flag.Set("logtostderr", "true")
	flag.Parse()
	klog.Info("Set logs output to stderr.")
	klog.Flush()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	controller := NewController(kubeClient, fluentbitimage, metricsimage, registrykey, kafkasecret, kafkasecretpath, namespace)

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	//logfilterInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&fluentbitimage, "fluentbitimage", "", "Fluent-bit docker image")
	flag.StringVar(&metricsimage, "metricsimage", "", "Metrics (= Fluent-bit + curl + jq) docker image")
	flag.StringVar(&registrykey, "registrykey", "", "Local registry key secret name")
	flag.StringVar(&kafkasecret, "kafkasecret", "", "Kafka client certs secret name")
	flag.StringVar(&kafkasecretpath, "kafkasecretpath", "/fluent-bit/kafka/certs", "Kafka client certs mount path")
	flag.StringVar(&namespace, "namespace", "default", "Namespace for fluent-bit")
}
