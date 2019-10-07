/*
Copyright 2019 The MayaData Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Storage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   StorageSpec   `json:"spec"`
	Status StorageStatus `json:"status,omitempty"`
}

type StorageSpec struct {
	// Capacity of the storage
	Capacity resource.Quantity `json:"capacity"`

	// Name of the node that should attach the storage
	//
	// This is optional
	NodeName *string `json:"nodeName,omitempty"`
}

// StoragePhase is a label for the condition of a storage at
// the current time.
type StoragePhase string

// These are the valid statuses of storage.
const (
	// StoragePending means the storage has been accepted by the system,
	// but one or more of the required resources has not been created.
	StoragePending StoragePhase = "Pending"

	// StorageAttached means the storage has been attached to a node.
	StorageAttached StoragePhase = "Attached"

	// StorageFailed indicates some failures with the controller, or
	// resources that are required to have this storage attached.
	StorageFailed StoragePhase = "Failed"
)

// StorageConditionType is a valid value for StorageCondition.Type
type StorageConditionType string

// These are valid conditions of storage.
const (
	// ResourcesCreated indicates whether all resources of the storage
	// are created.
	ResourcesCreated StorageConditionType = "ResourcesCreated"

	// PVCBound means the PVC associated with this storage
	// is bound against its associated PV
	PVCBound StorageConditionType = "PVCBound"

	// NodeSelected represents status if any node was selected to attach
	// this storage
	NodeSelected StorageConditionType = "NodeSelected"

	// NodeAvailable represents the status if the selected node is available
	NodeAvailable StorageConditionType = "NodeAvailable"

	// VolumeResize represents the status when this storage is undergoing
	// a resize operation
	VolumeResize StorageConditionType = "VolumeResize"
)

// ConditionStatus is a typed value to represent various condition statuses
type ConditionStatus string

// These are valid condition statuses.
const (
	// "ConditionTrue" means the storage is in the StorageConditionType
	ConditionTrue ConditionStatus = "True"

	// "ConditionFalse" means a storage is not in the StorageConditionType
	ConditionFalse ConditionStatus = "False"

	// "ConditionUnknown" means controller can't decide if storage
	// is in the StorageConditionType or not.
	ConditionUnknown ConditionStatus = "Unknown"
)

// StorageCondition contains details for the current condition of this storage.
type StorageCondition struct {
	// Type is the type of the condition.
	Type StorageConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=StorageConditionType"`

	// Status is the status of the condition.
	// Can be True, False, Unknown.
	Status ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=ConditionStatus"`

	// Last time we probed the condition.
	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty" protobuf:"bytes,3,opt,name=lastProbeTime"`

	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,4,opt,name=lastTransitionTime"`

	// Unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`

	// Human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}

// StorageStatus represents information about the status of the storage.
type StorageStatus struct {
	// The phase of a storage is a high-level summary of where the storage
	// is in its lifecycle.
	//
	// The conditions array, the reason and message fields, contain more
	// detail about the storage's status.
	Phase StoragePhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=StoragePhase"`

	// Current service state of storage.
	//
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []StorageCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,2,rep,name=conditions"`

	// A human readable message indicating details about why the storage
	// is in this condition.
	//
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`

	// A brief CamelCase message indicating details about why the storage
	// is in this state.
	//
	// e.g. 'Evicted'
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason"`

	// RFC 3339 date and time at which the object was acknowledged by its controller.
	StartTime *metav1.Time `json:"startTime,omitempty" protobuf:"bytes,7,opt,name=startTime"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type StorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Storage `json:"items"`
}
