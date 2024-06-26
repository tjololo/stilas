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
	"cloud.google.com/go/run/apiv2/runpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CloudRunSpec defines the desired state of CloudRun
type CloudRunSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

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

	//+kubebuilder:validation:Optional
	//+kubebuilder:default:=1
	TrafficMode runpb.IngressTraffic `json:"trafficMode"`

	//+kubebuilder:validation:Required
	//+kubebuilder:default:={allUsers}
	InvokeMembers []string `json:"invokeMembers,omitempty"`
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

	//+kubebuilder:validation:Optional
	LivenessProbe *CloudRunProbe `json:"livenessProbe"`

	//+kubebuilder:validation:Optional
	StartupProbe *CloudRunProbe `json:"readinessProbe"`
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
	Percent int32 `json:"percent"`

	//LatestRevision is a flag to indicate if this is the latest revision
	//+kubebuilder:example:=true
	//+kubebuilder:validation:Required
	//+kubebuilder:default:=false
	LatestRevision bool `json:"latestRevision"`
}

type CloudRunProbe struct {
	//+kubebuilder:validation:Required
	ProbeSpec CloudRunProbeSpec `json:"probeSpec"`
	//+kubebuilder:validation:Optional
	//+kubebuilder:default:=0
	InitialDelaySeconds int32 `json:"initialDelaySeconds"`
	//+kubebuilder:validation:Optional
	//+kubebuilder:default:=5
	TimeoutSeconds int32 `json:"timeoutSeconds"`
	//+kubebuilder:validation:Optional
	//+kubebuilder:default:=10
	PeriodSeconds int32 `json:"periodSeconds"`
	//+kubebuilder:validation:Optional
	//+kubebuilder:default:=3
	FailureThreshold int32 `json:"failureThreshold"`
}

type CloudRunProbeSpec struct {
	//+kubebuilder:validation:Required
	ProbeType CloudRunProbeType `json:"probeType"`
	//+kubebuilder:validation:Required
	Port int32 `json:"port"`
	//+kubebuilder:validation:Optional
	Service *string `json:"service,omitempty"`
	//+kubebuilder:validation:Optional
	Path *string `json:"path,omitempty"`
}

type CloudRunProbeType string

const (
	CloudRunProbeType_HTTPGet   CloudRunProbeType = "HTTPGet"
	CloudRunProbeType_TCPSocket CloudRunProbeType = "TCPSocket"
	CloudRunProbeType_Grpc      CloudRunProbeType = "Grpc"
)

// CloudRunStatus defines the observed state of CloudRun
type CloudRunStatus struct {
	Ready bool `json:"ready"`
	//+kubebuilder:validation:Optional
	Reconciling bool `json:"reconciling"`
	//+kubebuilder:validation:Optional
	Operations []*CloudRunOperation `json:"operations"`
	//+kubebuilder:validation:Optional
	Uri string `json:"uri,omitempty"`
	//+kubebuilder:validation:Optional
	LatestReadyRevision string `json:"latestReadyRevision,omitempty"`
	//+kubebuilder:validation:Optional
	Revisions []string `json:"revisions,omitempty"`
}

type CloudRunOperation struct {
	//+kubebuilder:validation:Optional
	Name string `json:"name"`
	//+kubebuilder:validation:Optional
	Done bool `json:"done"`
	//+kubebuilder:validation:Optional
	OperationType CloudRunOperationType `json:"operationType"`
}

type CloudRunOperationType string

const (
	CloudRunOperationType_Create CloudRunOperationType = "create"
	CloudRunOperationType_Update CloudRunOperationType = "update"
	CloudRunOperationType_Delete CloudRunOperationType = "delete"
)

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
