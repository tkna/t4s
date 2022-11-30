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

// MinoSpec defines the desired state of Mino.
type MinoSpec struct {
	// Id of the Mino. It must be greater than or equal to 1, as 0 is treated as a blank cell on the board.
	MinoID int `json:"minoId,omitempty"`

	// (Relative) coordinates of the Mino
	Coords []Coord `json:"coords,omitempty"`

	// Color of the Mino. It must be a string that Javascript recognizes as color, for instance "blue", "#0000FF" or "rgb(0, 0, 255)".
	Color string `json:"color,omitempty"`
}

// MinoStatus defines the observed state of Mino.
type MinoStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Mino is the Schema for the minoes API.
type Mino struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MinoSpec   `json:"spec,omitempty"`
	Status MinoStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MinoList contains a list of Mino.
type MinoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Mino `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Mino{}, &MinoList{})
}
