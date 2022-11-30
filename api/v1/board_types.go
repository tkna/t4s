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

// BoardSpec defines the desired state of Board.
type BoardSpec struct {
	// Width of the board (default: 11)
	//+kubebuilder:validation:Minimum=3
	//+kubebuilder:default=11
	Width int `json:"width,omitempty"`

	// Height of the board (default: 20)
	//+kubebuilder:validation:Minimum=3
	//+kubebuilder:default=20
	Height int `json:"height,omitempty"`

	// Wait time when a mino falls in millisec (default: 1000). The lower the value, the faster the falling speed. This value is inherited by Cron.
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:default=1000
	Wait int `json:"wait,omitempty"`

	// Desired state of the board. Possible values are "Playing" and "GameOver".
	//+kubebuilder:default="GameOver"
	State BoardState `json:"state,omitempty"`
}

// BoardStatus defines the observed state of Board.
type BoardStatus struct {
	// Board Data
	Data [][]int `json:"data,omitempty"`

	// Current Mino Data
	CurrentMino []CurrentMino `json:"currentMino,omitempty"`

	// Current state of the board. Possible values are "Playing" and "GameOver".
	State BoardState `json:"state,omitempty"`
}

type Coord struct {
	X int `json:"x,omitempty"`
	Y int `json:"y,omitempty"`
}

// CurrentMino stores the current mino information.
type CurrentMino struct {
	MinoID         int     `json:"minoId,omitempty"`
	Center         Coord   `json:"center,omitempty"`
	RelativeCoords []Coord `json:"relativeCoords,omitempty"`
	AbsoluteCoords []Coord `json:"absoluteCoords,omitempty"`
}

func (mino CurrentMino) DeepCopy() CurrentMino {
	newMino := CurrentMino{
		MinoID:         mino.MinoID,
		Center:         Coord{X: mino.Center.X, Y: mino.Center.Y},
		RelativeCoords: []Coord{},
		AbsoluteCoords: []Coord{},
	}
	for _, v := range mino.RelativeCoords {
		newMino.RelativeCoords = append(newMino.RelativeCoords, Coord{X: v.X, Y: v.Y})
	}
	for _, v := range mino.AbsoluteCoords {
		newMino.AbsoluteCoords = append(newMino.AbsoluteCoords, Coord{X: v.X, Y: v.Y})
	}
	return newMino
}

// BoardState defines the state of Board
// +kubebuilder:validation:Enum=Playing;GameOver
type BoardState string

const (
	Playing  = BoardState("Playing")
	GameOver = BoardState("GameOver")
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="WIDTH",type="integer",JSONPath=".spec.width"
//+kubebuilder:printcolumn:name="HEIGHT",type="integer",JSONPath=".spec.height"
//+kubebuilder:printcolumn:name="WAIT",type="integer",JSONPath=".spec.wait"

// Board is the Schema for the boards API.
type Board struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BoardSpec   `json:"spec,omitempty"`
	Status BoardStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BoardList contains a list of Board.
type BoardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Board `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Board{}, &BoardList{})
}
