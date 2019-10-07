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
	"fmt"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corelisters "k8s.io/client-go/listers/core/v1"
	ref "k8s.io/client-go/tools/reference"
	"k8s.io/klog"

	ddp "github.com/mayadata-io/storage-provisioner/pkg/apis/ddp/v1alpha1"
)

// Reconciler manages reconciling storage API
// in kubernetes cluster
type Reconciler struct {
	// instances to invoke various Kubernetes APIs
	Clientset kubernetes.Interface
	PVCLister corelisters.PersistentVolumeClaimLister

	// storage that will get reconciled
	storage *ddp.Storage

	// reference to above storage object which has extra
	// information like APIVersion & Kind
	storageRef *v1.ObjectReference

	// name of the storage provider
	providerName string

	// name of the storage attacher
	attacherName string

	// name of the node where the storage gets attached to
	nodeName string
}

func (r *Reconciler) String() string {
	if r.storage == nil {
		return "StorageReconciler"
	}
	return fmt.Sprintf(
		"StorageReconciler %s/%s", r.storage.Namespace, r.storage.Name,
	)
}

// Reconcile accepts storage as the desired state and starts executing
// the reconcile logic based on this desired state
//
// NOTE:
//	Reconcile logic needs to be idempotent
func (r *Reconciler) Reconcile(stor *ddp.Storage) error {
	r.storage = stor

	var (
		found bool
		err   error
	)
	defer func() {
		if err != nil {
			errors.Wrapf(err, "%s: Reconcile failed", r)
		}
	}()

	r.storageRef, err = ref.GetReference(scheme.Scheme, r.storage)
	if err != nil {
		return err
	}

	if r.providerName, found = findProviderFromStorage(stor); !found {
		return errors.Errorf(
			"Missing annotation %q", storageclassProviderKey,
		)
	}

	if r.attacherName, found = findAttacherFromStorage(stor); !found {
		return errors.Errorf(
			"Missing annotation %q", storageCSIAttacherKey,
		)
	}

	// find if PVC is created in previous reconcile attempt
	pvc, err := r.findPVC()
	if err != nil {
		return err
	}

	// create PVC if not found
	if pvc == nil {
		return r.createPVC()
	}

	// update PVC if desired state was changed
	update, err := r.updatePVC(pvc)
	if !update {
		klog.V(3).Infof("%s: No change to desired state", r)
	}
	return err
}

// findPVC will list & find the correct PVC if available
func (r *Reconciler) findPVC() (*v1.PersistentVolumeClaim, error) {
	var err error

	defer func() {
		if err != nil {
			err = errors.Wrapf(err, "%s: Find PVC failed", r)
		}
	}()

	// PVC & storage must have same namespace
	list, err :=
		r.PVCLister.PersistentVolumeClaims(r.storage.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	for _, pvc := range list {
		isowner := isObjectReferenceAnOwner(pvc.OwnerReferences, r.storageRef)
		if isowner {
			return pvc, nil
		}
	}
	return nil, nil
}

// updatePVC updates the PVC if there are any changes to desired state
func (r *Reconciler) updatePVC(pvc *v1.PersistentVolumeClaim) (bool, error) {

	var err error
	defer func() {
		if err != nil {
			err = errors.Wrapf(err, "%s: Update PVC failed", r)
		}
	}()

	if pvc.Spec.Resources.Requests[v1.ResourceStorage] == r.storage.Spec.Capacity {
		// no changes
		return false, nil
	}

	copy := pvc.DeepCopy()
	copy.Spec.Resources.Requests[v1.ResourceStorage] = r.storage.Spec.Capacity

	// PVC & storage must have same namespace
	_, err =
		r.Clientset.CoreV1().PersistentVolumeClaims(r.storage.Namespace).Update(copy)
	return true, err
}

func (r *Reconciler) createPVC() error {
	var err error

	defer func() {
		if err != nil {
			err = errors.Wrapf(err, "%s: Create PVC failed", r)
		}
	}()

	r.nodeName = r.getNodeName()

	// build a new instance of PVC object
	pvc := r.newPVC()

	// PVC & storage must have same namespace
	_, err =
		r.Clientset.CoreV1().PersistentVolumeClaims(r.storage.Namespace).Create(pvc)
	return err
}

// getNodeName returns the node name that will be used to attach
// the storage
//
// TODO (@amitkumardas):
// 		Validate if this nodeName is allowed in storageclass (provider)
// allowed topologies
func (r *Reconciler) getNodeName() string {
	if r.storage.Spec.NodeName != nil {
		return *r.storage.Spec.NodeName
	}
	return ""
}

// newPVC returns a new instance of PVC API.
//
// NOTE:
//	This should be used only for PVC create case
func (r *Reconciler) newPVC() *v1.PersistentVolumeClaim {

	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: r.storageRef.Namespace + "-" + r.storageRef.Name + "-",
			Namespace:    r.storageRef.Namespace,
			Annotations: map[string]string{
				nodeNameKey:           r.nodeName,
				storageCSIAttacherKey: r.attacherName,
			},
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion:         r.storageRef.APIVersion,
					Kind:               r.storageRef.Kind,
					Name:               r.storageRef.Name,
					UID:                r.storageRef.UID,
					Controller:         boolPtr(true),
					BlockOwnerDeletion: boolPtr(true),
				},
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			Resources: v1.ResourceRequirements{
				Requests: map[v1.ResourceName]resource.Quantity{
					v1.ResourceStorage: r.storage.Spec.Capacity,
				},
			},
			StorageClassName: strPtr(r.providerName),
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
		},
	}
}
