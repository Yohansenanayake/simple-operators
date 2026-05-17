/*
Copyright 2026.

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

// Ec2InstanceSpec defines the desired state of Ec2Instance
type Ec2InstanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	InstanceType      string            `json:"instanceType"`
	AMIId             string            `json:"amiId"`
	Region            string            `json:"region"`
	AvailabilityZone  string            `json:"availabilityZone,omitempty"`
	KeyPair           string            `json:"keyPair,omitempty"`
	SecurityGroups    []string          `json:"securityGroups,omitempty"` //slice of strings
	Subnet            string            `json:"subnet,omitempty"`
	UserData          string            `json:"userData,omitempty"`
	Tags              map[string]string `json:"tags,omitempty"`              //map of string key-value pairs'
	Storage           StorageConfig     `json:"storage,omitempty"`           //nested struct for storage configuration
	AssociatePublicIP bool              `json:"associatePublicIP,omitempty"` //whether to associate a public IP address with the instance

}

// Ec2InstanceStatus defines the observed state of Ec2Instance.
type Ec2InstanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	InstanceID string       `json:"instanceId,omitempty"` // The ID of the EC2 instance in AWS
	State      string       `json:"state,omitempty"`      // The current state of the EC2 instance (e.g., "Pending", "Running", "Stopped", "Terminated")
	PublicIP   string       `json:"publicIp,omitempty"`   // The public IP address associated with the EC2 instance, if any
	PrivateIP  string       `json:"privateIp,omitempty"`  // The private IP address associated with the EC2 instance, if any
	PublicDNS  string       `json:"publicDns,omitempty"`  // The public DNS name associated with the EC2 instance, if any
	PrivateDNS string       `json:"privateDns,omitempty"` // The private DNS name associated with the EC2 instance, if any
	LaunchTime *metav1.Time `json:"launchTime,omitempty"` // The time when the EC2 instance was launched
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Ec2Instance is the Schema for the ec2instances API
type Ec2Instance struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Ec2Instance
	// +required
	Spec Ec2InstanceSpec `json:"spec"`

	// status defines the observed state of Ec2Instance
	// +optional
	Status Ec2InstanceStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// Ec2InstanceList contains a list of Ec2Instance
type Ec2InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Ec2Instance `json:"items"`
}

// Custom helper structs
type StorageConfig struct {
	RootVolume        VolumeConfig   `json:"rootVolume"`
	AdditionalVolumes []VolumeConfig `json:"additionalVolumes,omitempty"`
}

type VolumeConfig struct {
	Size       int32  `json:"size"`                 // Size in GB
	Type       string `json:"type,omitempty"`       // E.g., "gp2", "io1"
	DeviceName string `json:"deviceName,omitempty"` // E.g., "/dev/sda1"
	Encrypted  bool   `json:"encrypted,omitempty"`  // Whether the volume should be encrypted
}

func init() {
	SchemeBuilder.Register(&Ec2Instance{}, &Ec2InstanceList{})
}
