/*
Copyright 2017 The Kubernetes Authors.
Copyright 2019 The MayaData Authors.

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
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"github.com/kubernetes-csi/csi-lib-utils/leaderelection"

	"github.com/mayadata-io/storage-provisioner/build"
	ddpkubernetes "github.com/mayadata-io/storage-provisioner/client/generated/clientset/versioned"
	ddpscheme "github.com/mayadata-io/storage-provisioner/client/generated/clientset/versioned/scheme"
	ddpinformers "github.com/mayadata-io/storage-provisioner/client/generated/informer/externalversions"
	"github.com/mayadata-io/storage-provisioner/storage"
)

const (
	leaderElectionTypeLeases     = "leases"
	leaderElectionTypeConfigMaps = "configmaps"

	controllerName = "ddp-storage-provisioner"
)

// Command line flags
var (
	kubeconfig = flag.String(
		"kubeconfig", "",
		`Absolute path to the kubeconfig file. 
		Required only when running outside of cluster.`,
	)

	resync = flag.Duration(
		"resync", 10*time.Minute,
		"Resync interval of the controller.",
	)

	showVersion = flag.Bool("version", false, "Shows storage-provisioner's version.")

	workerThreads = flag.Uint(
		"worker-threads", 25,
		"Number of storage provisioner worker threads",
	)

	retryIntervalStart = flag.Duration(
		"retry-interval-start", time.Second,
		`Initial retry interval of failed create volume or delete volume. 
		It doubles with each failure, up to retry-interval-max.`,
	)

	retryIntervalMax = flag.Duration(
		"retry-interval-max", 5*time.Minute,
		"Maximum retry interval of failed create volume or delete volume.",
	)

	enableLeaderElection = flag.Bool(
		"leader-election", false,
		"Enable leader election.",
	)

	leaderElectionNamespace = flag.String(
		"leader-election-namespace", "",
		`Namespace where the leader election resource lives. 
		Defaults to this pod namespace if not set.`,
	)
)

type leaderElection interface {
	Run() error
	WithNamespace(namespace string)
}

func main() {
	klog.InitFlags(nil)
	flag.Set("logtostderr", "true")
	flag.Parse()

	if *showVersion {
		fmt.Println(os.Args[0], build.Hash)
		return
	}
	klog.Infof("Version: %s", build.Hash)

	// Create the kubernetes client config.
	// Use kubeconfig if given, otherwise assume in-cluster.
	config, err := buildConfig(*kubeconfig)
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	if *workerThreads == 0 {
		klog.Error("option -worker-threads must be greater than zero")
		os.Exit(1)
	}

	utilruntime.Must(ddpscheme.AddToScheme(scheme.Scheme))

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	ddpClientset, err := ddpkubernetes.NewForConfig(config)
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	factory := informers.NewSharedInformerFactory(clientset, *resync)
	ddpFactory := ddpinformers.NewSharedInformerFactory(ddpClientset, *resync)

	storageQ := workqueue.NewNamedRateLimitingQueue(
		workqueue.NewItemExponentialFailureRateLimiter(*retryIntervalStart, *retryIntervalMax),
		"ddp-storage-q",
	)
	pvcQ := workqueue.NewNamedRateLimitingQueue(
		workqueue.NewItemExponentialFailureRateLimiter(*retryIntervalStart, *retryIntervalMax),
		"ddp-pvc-q",
	)

	// new instance of storage reconciler
	storageReconciler := &storage.Reconciler{
		Clientset: clientset,
		PVCLister: factory.Core().V1().PersistentVolumeClaims().Lister(),
	}

	// new instance of storage reconciler
	pvcReconciler := &storage.PVCReconciler{
		Clientset: clientset,
		VALister:  factory.Storage().V1beta1().VolumeAttachments().Lister(),
	}

	// new instance of storage controller
	ctrl := &storage.Controller{
		Name:                controllerName,
		InformerFactory:     factory,
		DDPInformerFactory:  ddpFactory,
		StorageQueue:        storageQ,
		PVCQueue:            pvcQ,
		StorageReconcilerFn: storageReconciler.Reconcile,
		PVCReconcilerFn:     pvcReconciler.Reconcile,
	}

	// initialize the controller before running
	err = ctrl.Init()
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	// define the controller run func, It is a wrapper over original
	// controller run function with context management
	ctrlRun := func(ctx context.Context) {
		// create a stop channel & pass this wherever needed
		stopCh := ctx.Done()

		factory.Start(stopCh)
		ddpFactory.Start(stopCh)

		// run the storage controller
		ctrl.Run(int(*workerThreads), stopCh)
	}

	if !*enableLeaderElection {
		ctrlRun(context.TODO())
	} else {
		// Name of config map with leader election lock
		lockName := controllerName + "-leader"
		le := leaderelection.NewLeaderElection(clientset, lockName, ctrlRun)

		if *leaderElectionNamespace != "" {
			le.WithNamespace(*leaderElectionNamespace)
		}

		if err := le.Run(); err != nil {
			klog.Fatalf("Leader election failed: %v", err)
		}
	}
}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}
