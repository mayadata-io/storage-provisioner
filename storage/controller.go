/*
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

package storage

import (
	"strings"

	"k8s.io/klog"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	ddpinformers "github.com/mayadata-io/storage-provisioner/client/generated/informer/externalversions"
	ddplisters "github.com/mayadata-io/storage-provisioner/client/generated/lister/ddp/v1alpha1"
	ddp "github.com/mayadata-io/storage-provisioner/pkg/apis/ddp/v1alpha1"
)

const (
	// default controller name
	defaultCtrlName string = "StorageController"
)

// storageQueueKey returns a key in string format corresponding to the
// given storage. This string form is suitable to be used as a key.
func storageQueueKey(s *ddp.Storage) string {
	return s.Namespace + ":" + s.Name
}

// parseQueueKey evaluates the namespace & name from the given
// key
func parseQueueKey(key string) (namespace, name string) {
	splits := strings.Split(key, ":")
	if len(splits) != 2 {
		return "", ""
	}
	return splits[0], splits[1]
}

// pvcQueueKey returns a key in string format corresponding to the
// given PVC. This string form is suitable to be used as a key.
func pvcQueueKey(p *v1.PersistentVolumeClaim) string {
	return p.Namespace + ":" + p.Name
}

// Controller to add / remove storage
type Controller struct {
	// Name of this controller
	Name string

	// Various informer factories required by this controller
	InformerFactory    informers.SharedInformerFactory
	DDPInformerFactory ddpinformers.SharedInformerFactory

	// core reconciliation logic
	StorageReconcilerFn func(*ddp.Storage) error
	PVCReconcilerFn     func(*v1.PersistentVolumeClaim) error

	// Queues to queue reconcile keys before invoking reconciliation
	StorageQueue workqueue.RateLimitingInterface
	PVCQueue     workqueue.RateLimitingInterface

	storageLister       ddplisters.StorageLister
	storageListerSynced cache.InformerSynced
	pvcLister           corelisters.PersistentVolumeClaimLister
	pvcListerSynced     cache.InformerSynced
}

// String implements Stringer interface
func (ctrl *Controller) String() string {
	return ctrl.Name
}

// Init initializes the storage controller with required properties
//
// NOTE:
//	Init must be invoked by caller before invoking any other
// methods of this instance
func (ctrl *Controller) Init() error {
	if ctrl.Name == "" {
		ctrl.Name = defaultCtrlName
	}

	if ctrl.InformerFactory == nil {
		return errors.Errorf("%s: Init failed: Nil informer factory", ctrl)
	}
	if ctrl.DDPInformerFactory == nil {
		return errors.Errorf("%s: Init failed: Nil ddp informer factory", ctrl)
	}
	if ctrl.StorageReconcilerFn == nil {
		return errors.Errorf("%s: Init failed: Nil storage reconciler", ctrl)
	}
	if ctrl.PVCReconcilerFn == nil {
		return errors.Errorf("%s: Init failed: Nil pvc reconciler", ctrl)
	}
	if ctrl.StorageQueue == nil {
		return errors.Errorf("%s: Init failed: Nil storage queue", ctrl)
	}
	if ctrl.PVCQueue == nil {
		return errors.Errorf("%s: Init failed: Nil pvc queue", ctrl)
	}

	storageInformer := ctrl.DDPInformerFactory.Ddp().V1alpha1().Storages()
	pvcInformer := ctrl.InformerFactory.Core().V1().PersistentVolumeClaims()

	storageInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctrl.storageAdded,
		UpdateFunc: ctrl.storageUpdated,
	})
	ctrl.storageLister = storageInformer.Lister()
	ctrl.storageListerSynced = storageInformer.Informer().HasSynced

	pvcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctrl.pvcAdded,
		UpdateFunc: ctrl.pvcUpdated,
	})
	ctrl.pvcLister = pvcInformer.Lister()
	ctrl.pvcListerSynced = pvcInformer.Informer().HasSynced

	return nil
}

// Run starts provisioner and listens on channel events
func (ctrl *Controller) Run(workers int, stopCh <-chan struct{}) {
	// shutdown the queues
	defer ctrl.StorageQueue.ShutDown()
	defer ctrl.PVCQueue.ShutDown()

	klog.Infof("Starting %s", ctrl)
	defer klog.Infof("Shutting down %s", ctrl)

	if !cache.WaitForCacheSync(stopCh, ctrl.storageListerSynced, ctrl.pvcListerSynced) {
		klog.Errorf("%s: Cannot sync caches", ctrl)
		return
	}

	for i := 0; i < workers; i++ {
		// run all reconcile funcs in a continuous loop
		go wait.Until(ctrl.syncStorage, 0, stopCh)
		go wait.Until(ctrl.syncPVC, 0, stopCh)
	}

	// block till stop is invoked
	<-stopCh
}

// storageAdded reacts to a storage creation
func (ctrl *Controller) storageAdded(obj interface{}) {
	stor := obj.(*ddp.Storage)
	ctrl.StorageQueue.Add(storageQueueKey(stor))
}

// storageAdded reacts to a storage update
func (ctrl *Controller) storageUpdated(old, new interface{}) {
	ctrl.storageAdded(new)
}

// pvcAdded reacts to a PVC creation
func (ctrl *Controller) pvcAdded(obj interface{}) {
	pvc := obj.(*v1.PersistentVolumeClaim)

	if !isStorageKindOwnerOfPVC(pvc) {
		// this PVC does not belong to storage API
		klog.V(3).Infof(
			"%s: Ignoring PVC %s/%s: Storage is not owner",
			ctrl, pvc.Namespace, pvc.Name,
		)
		return
	}

	ctrl.PVCQueue.Add(pvcQueueKey(pvc))
}

// pvcUpdated reacts to a PVC update
func (ctrl *Controller) pvcUpdated(old, new interface{}) {
	ctrl.pvcAdded(new)
}

// syncStorage starts reconciliation of storage as per the needs of
// storage controller
func (ctrl *Controller) syncStorage() {
	key, quit := ctrl.StorageQueue.Get()
	if quit {
		// nothing to do
		return
	}
	defer ctrl.StorageQueue.Done(key)

	storName := key.(string)
	var err error
	handleErr := func() {
		if err != nil {
			if apierrs.IsNotFound(err) {
				// Storage was deleted in the meantime, ignore.
				klog.V(3).Infof(
					"%s: Sync ignored: Storage %q does not exist", ctrl, storName,
				)
				return
			}
			klog.Errorf(
				"%s: Sync failed: Will re-queue storage %q: %v", ctrl, storName, err,
			)
			ctrl.StorageQueue.AddRateLimited(key)
			return
		}
	}
	defer handleErr()

	klog.V(4).Infof("%s: Sync started: Storage %q", ctrl, storName)
	ns, name := parseQueueKey(storName)

	// get storage to process further
	stor, err := ctrl.storageLister.Storages(ns).Get(name)
	if err != nil {
		return
	}

	err = ctrl.StorageReconcilerFn(stor)
	if err != nil {
		return
	}

	// The operation has finished successfully, reset exponential backoff
	ctrl.StorageQueue.Forget(key)
	klog.V(4).Infof("%s: Sync completed: Storage %q", ctrl, storName)
}

// syncPVC starts reconciliation of PVC as per the needs of storage
// controller
func (ctrl *Controller) syncPVC() {
	key, quit := ctrl.PVCQueue.Get()
	if quit {
		// nothing to do
		return
	}
	defer ctrl.PVCQueue.Done(key)

	pvcName := key.(string)
	var err error
	handleErr := func() {
		if err != nil {
			if apierrs.IsNotFound(err) {
				// PV was deleted in the meantime, ignore.
				klog.V(3).Infof(
					"%s: Sync ignored: PVC %q does not exist", ctrl, pvcName,
				)
				return
			}
			klog.Errorf(
				"%s: Sync failed: Will re-queue PVC %q: %v", ctrl, pvcName, err,
			)
			ctrl.PVCQueue.AddRateLimited(key)
		}
	}
	defer handleErr()

	klog.V(4).Infof("%s: Sync started: PVC %q", ctrl, pvcName)
	ns, name := parseQueueKey(pvcName)

	// get PVC to process
	pvc, err := ctrl.pvcLister.PersistentVolumeClaims(ns).Get(name)
	if err != nil {
		return
	}

	err = ctrl.PVCReconcilerFn(pvc)
	if err != nil {
		return
	}

	// The operation has finished successfully, reset exponential backoff
	ctrl.PVCQueue.Forget(key)
	klog.V(4).Infof("%s: Sync completed: PVC %q", ctrl, pvcName)
}
