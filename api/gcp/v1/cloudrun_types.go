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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CloudRunSpec defines the desired state of CloudRun
type CloudRunSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//Name is the name of the Cloud Run service
	//+kubebuilder:example:=my-service
	//+kubebuilder:validation:Required
	Name string `json:"name"`

	//Location is the location of the Cloud Run service
	//+kubebuilder:example:=us-central1
	//+kubebuilder:validation:Required
	Location string `json:"location"`

	//Image is the container image to deploy
	//+kubebuilder:example:=gcr.io/my-project/my-image
	//+kubebuilder:validation:Required
	Containers []CloudRunContainer `json:"containers"`

	//ProjectID id of the gcp project
	//+kubebuilder:example:=my-project
	//+kubebuilder:validation:Required
	ProjectID string `json:"projectID"`

	//Traffic is the percentage of traffic to send to this service
	//+kubebuilder:validation:Optional
	Traffic []CloudRunTraffic `json:"traffic"`
}

// CloudRunContainer defines the container configuration for a Cloud Run service
type CloudRunContainer struct {
	//Image is the container image to deploy
	//+kubebuilder:example:=gcr.io/my-project/my-image
	//+kubebuilder:validation:Required
	Image string `json:"image"`

	//Port is the port the container listens on
	//+kubebuilder:example:=8080
	//+kubebuilder:validation:Optional
	Port int32 `json:"port"`

	//Name is the name of the container
	//+kubebuilder:example:=my-container
	//+kubebuilder:validation:Required
	Name string `json:"name"`
}

// CloudRunTraffic defines the traffic configuration for a Cloud Run service
type CloudRunTraffic struct {
	//Revision is the name of the revision
	//+kubebuilder:example:=my-revision
	//+kubebuilder:validation:Optional
	Revision string `json:"revision"`

	//Percent is the percentage of traffic to send to this revision
	//+kubebuilder:example:=50
	//+kubebuilder:validation:Required
	Percent int `json:"percent"`

	//LatestRevision is a flag to indicate if this is the latest revision
	//+kubebuilder:example:=true
	//+kubebuilder:validation:Required
	//+kubebuilder:default:=false
	LatestRevision bool `json:"latestRevision"`
}

// CloudRunStatus defines the observed state of CloudRun
type CloudRunStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Done           bool   `json:"done"`
	OperationsName string `json:"operationsName"`
	Success        bool   `json:"success"`
	Uri            string `json:"uri"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CloudRun is the Schema for the cloudruns API
type CloudRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudRunSpec   `json:"spec,omitempty"`
	Status CloudRunStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CloudRunList contains a list of CloudRun
type CloudRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudRun{}, &CloudRunList{})
}
