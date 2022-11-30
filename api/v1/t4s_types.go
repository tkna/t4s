/*
Copyright 2022.

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

// T4sSpec defines the desired state of T4s.
type T4sSpec struct {
	// Width of the board (default: 10). This value is inherited by Board.
	//+kubebuilder:validation:Minimum=4
	//+kubebuilder:validation:Maximum=20
	//+kubebuilder:default=11
	Width int `json:"width,omitempty"`

	// Height of the board (default: 20). This value is inherited by Board.
	//+kubebuilder:validation:Minimum=4
	//+kubebuilder:validation:Maximum=30
	//+kubebuilder:default=20
	Height int `json:"height,omitempty"`

	// Wait time when a mino falls in millisec (default: 1000). The lower the value, the faster the falling speed. This value is inherited by Board and Cron.
	//+kubebuilder:validation:Minimum=200
	//+kubebuilder:default=1000
	Wait int `json:"wait,omitempty"`

	// Type of the Service to which a user accesses to (default: NodePort). Supported values are "NodePort" and "LoadBalancer".
	ServiceType string `json:"serviceType,omitempty"`

	// Specifies NodePort value when serviceType is "NodePort". If not specified, it is allocated automatically by Kubernetes' NodePort mechanism.
	NodePort int32 `json:"nodePort,omitempty"`

	// Specifies LoadBalancerIP value when serviceType is "LoadBalancer".
	LoadBalancerIP string `json:"loadBalancerIP,omitempty"`

	// Specifies LoadBalancerSourceRanges when serviceType is "LoadBalancer".
	LoadBalancerSourceRanges []string `json:"loadBalancerSourceRanges,omitempty"`
}

// T4sStatus defines the observed state of T4s.
type T4sStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="WIDTH",type="integer",JSONPath=".spec.width"
//+kubebuilder:printcolumn:name="HEIGHT",type="integer",JSONPath=".spec.height"
//+kubebuilder:printcolumn:name="WAIT",type="integer",JSONPath=".spec.wait"

// T4s is the Schema for the T4s API.
type T4s struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   T4sSpec   `json:"spec,omitempty"`
	Status T4sStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// T4sList contains a list of T4s.
type T4sList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []T4s `json:"items"`
}

func init() {
	SchemeBuilder.Register(&T4s{}, &T4sList{})
}
