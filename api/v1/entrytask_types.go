/*
Copyright 2024.

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

package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EntryTaskSpec defines the desired state of EntryTask
type EntryTaskSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	DesiredReplicas int32                 `json:"desired_replicas"`
	Image           string                `json:"image"`
	Selector        *metav1.LabelSelector `json:"selector"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Template v1.PodTemplateSpec `json:"template"`
}

// EntryTaskStatus defines the observed state of EntryTask
type EntryTaskStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ActualReplicas int32    `json:"actualReplicas"`
	Endpoints      []string `json:"endpoints"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// EntryTask is the Schema for the entrytasks API
type EntryTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EntryTaskSpec   `json:"spec,omitempty"`
	Status EntryTaskStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EntryTaskList contains a list of EntryTask
type EntryTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EntryTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EntryTask{}, &EntryTaskList{})
}
