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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ddp "github.com/mayadata-io/storage-provisioner/pkg/apis/ddp/v1alpha1"
)

const (
	// storageclassProviderKey holds the storageclass name. Here
	// storageclass is the provider of storage.
	//
	// This key is expected to be present in API annotations
	storageclassProviderKey string = "storageprovisioner.ddp.mayadata.io/storageclass-name"

	// storageCSIAttacherKey holds the name of the CSI attacher that will
	// be responsible to attach the storage
	storageCSIAttacherKey string = "storageprovisioner.ddp.mayadata.io/csi-attacher-name"

	// nodeNameKey holds the name of node where storage should
	// get attached
	nodeNameKey string = "storageprovisioner.ddp.mayadata.io/node-name"
)

// boolPtr returns a pointer to a bool
func boolPtr(b bool) *bool {
	o := b
	return &o
}

// strPtr returns a pointer to a string
func strPtr(s string) *string {
	o := s
	return &o
}

// findValueFromDict finds the value corresponding to the
// given key
func findValueFromDict(dict map[string]string, key string) (string, bool) {
	if len(dict) == 0 {
		return "", false
	}
	val, found := dict[key]
	return val, found
}

// findProviderFromStorage finds the storage provider name from
// storage API
func findProviderFromStorage(storage *ddp.Storage) (string, bool) {
	anns := storage.GetAnnotations()
	return findValueFromDict(anns, storageclassProviderKey)
}

// findAttacherFromStorage finds the attacher name from Storage API
func findAttacherFromStorage(storage *ddp.Storage) (string, bool) {
	anns := storage.GetAnnotations()
	return findValueFromDict(anns, storageCSIAttacherKey)
}

// findAttacherFromPVC finds the attacher name from PVC API
func findAttacherFromPVC(pvc *v1.PersistentVolumeClaim) (string, bool) {
	anns := pvc.GetAnnotations()
	return findValueFromDict(anns, storageCSIAttacherKey)
}

// findNodeNameFromPVC finds the node name from
// PVC API
func findNodeNameFromPVC(pvc *v1.PersistentVolumeClaim) (string, bool) {
	anns := pvc.GetAnnotations()
	return findValueFromDict(anns, nodeNameKey)
}

// containsOwner returns true if the given owner is present in the
// given list of owners
func containsOwner(owners []metav1.OwnerReference, given metav1.OwnerReference) bool {
	for _, o := range owners {
		if o.APIVersion == given.APIVersion &&
			o.Kind == given.Kind &&
			o.Name == given.Name &&
			o.UID == given.UID {
			return true
		}
	}
	return false
}

// isObjectReferenceAnOwner returns true if given ObjectReference
// is present in the given list of owners
func isObjectReferenceAnOwner(
	owners []metav1.OwnerReference, ref *v1.ObjectReference,
) bool {

	return containsOwner(owners, metav1.OwnerReference{
		APIVersion: ref.APIVersion,
		Kind:       ref.Kind,
		Name:       ref.Name,
		UID:        ref.UID,
	})
}

// isStorageKindOwnerOfPVC returns true if the given PVC instance
// is owned by any storage
func isStorageKindOwnerOfPVC(pvc *v1.PersistentVolumeClaim) bool {
	owners := pvc.GetOwnerReferences()

	for _, o := range owners {
		if o.Kind == "Storage" &&
			o.APIVersion == ddp.SchemeGroupVersion.String() {
			return true
		}
	}
	return false
}
