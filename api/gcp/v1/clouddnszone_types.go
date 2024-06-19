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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CloudDnsZoneSpec defines the desired state of CloudDnsZone
type CloudDnsZoneSpec struct {
	//ProjectID id of the gcp project
	//+kubebuilder:example:=my-project
	//+kubebuilder:validation:Required
	ProjectID string `json:"projectID"`
	//DnsName defines the name of the zone. Must be a valid DNS name
	// +kubebuilder:validation:Patter=^(?!:\/\/)(?=.{1,255}$)((.{1,63}\.){1,127}(?![0-9]*$)[a-z0-9-]+\.?)$
	// +kubebuilder:validation:Required
	DnsName string `json:"dnsName"`
	//PrivateZone defines if the zone is private or public
	// +kubebuilder:default=false
	PrivateZone bool `json:"public"`
	//DnsSecSpec defines the DNSSEC configuration for the zone
	// +kubebuilder:validation:Optional
	DnsSecSpec DnsSecSpec `json:"dnsSecSpec,omitempty"`
	//CleanupOnDelete defines if the zone should be deleted when the resource is deleted
	// +kubebuilder:default=false
	// +kubebuilder:validation:Optional
	CleanupOnDelete bool `json:"cleanupOnDelete,omitempty"`
}

type DnsSecSpec struct {
	// +kubebuilder:default=On
	// +kubebuilder:validation:Enum=On;Off;Transfer
	//State specifies whether DNSSEC is enabled, and what mode it is in
	State string `json:"state,omitempty"`
	// +kubebuilder:default=true
	//NonExistence defines if the NSEC3 record should be included in the response
	NonExistence bool `json:"nonExistence,omitempty"`
}

// CloudDnsZoneStatus defines the observed state of CloudDnsZone
type CloudDnsZoneStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +kubebuilder:validation:Optional
	Nameservers []string `json:"nameservers,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CloudDnsZone is the Schema for the clouddnszones API
type CloudDnsZone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudDnsZoneSpec   `json:"spec,omitempty"`
	Status CloudDnsZoneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudDnsZoneList contains a list of CloudDnsZone
type CloudDnsZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudDnsZone `json:"items"`
}

func (c *CloudDnsZone) GetCloudDnsZoneFullName() string {
	return fmt.Sprintf("%s-%s", c.Namespace, c.Name)
}

func init() {
	SchemeBuilder.Register(&CloudDnsZone{}, &CloudDnsZoneList{})
}
