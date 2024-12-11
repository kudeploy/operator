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

// ServicePort defines a port mapping
type ServicePort struct {
	// Protocol defines the protocol used for the port (TCP/UDP)
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=TCP;UDP
	// +kubebuilder:default=TCP
	Protocol string `json:"protocol,omitempty"`

	// ServicePort is the port number to expose
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	ServicePort int32 `json:"hostPort,omitempty"`

	// ContainerPort is the port number to access on the container
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	ContainerPort int32 `json:"containerPort"`
}

// EnvVar represents an environment variable present in a Container.
type EnvVar struct {
	// Name of the environment variable. Must be a C_IDENTIFIER.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Variable references $(VAR_NAME) are expanded
	// using the previously defined environment variables in the container and
	// any service environment variables. If a variable cannot be resolved,
	// the reference in the input string will be unchanged. Double $$ are reduced
	// to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e.
	// "$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)".
	// Escaped references will never be expanded, regardless of whether the variable
	// exists or not.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	Value string `json:"value,omitempty"`
}

// ServiceSpec defines the desired state of Service.
type ServiceSpec struct {
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// +kubebuilder:default=IfNotPresent
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// +kubebuilder:validation:Optional
	Ports []ServicePort `json:"ports,omitempty"`

	// +kubebuilder:validation:Optional
	Env []EnvVar `json:"env,omitempty"`
}

// ServiceStatus defines the observed state of Service.
type ServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Service is the Schema for the services API.
type Service struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceSpec   `json:"spec,omitempty"`
	Status ServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceList contains a list of Service.
type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Service `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Service{}, &ServiceList{})
}
