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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BoardSpec defines the desired state of Board
type BoardSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Width of the board
	//+kubebuilder:validation:Required
	Width uint `json:"width"`

	// Height of the board
	//+kubebuilder:validation:Required
	Height uint `json:"height"`
}

// BoardStatus defines the observed state of Board
type BoardStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	
	// Board Data
	Data [][]uint 	  `json:"data,omitempty"`

	// Current Mino Data
	CurrentMino []Mino  `json:"currentMino,omitempty"`
}

type Coord struct {
	X int	`json:"x,omitempty"`
	Y int	`json:"y,omitempty"`
}

type Mino struct {
	MinoId uint  `json:"minoId,omitempty"`
	Center Coord `json:"center,omitempty"`
	RelativeCoords []Coord  `json:"relativeCoords,omitempty"`
	AbsoluteCoords []Coord  `json:"absoluteCoords,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="WIDTH",type="integer",JSONPath=".spec.width"
//+kubebuilder:printcolumn:name="HEIGHT",type="integer",JSONPath=".spec.height"

// Board is the Schema for the boards API
type Board struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BoardSpec   `json:"spec,omitempty"`
	Status BoardStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BoardList contains a list of Board
type BoardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Board `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Board{}, &BoardList{})
}
