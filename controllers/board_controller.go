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

package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	tetrisv1 "github.com/tkna/tetris-operator/api/v1"
)

// BoardReconciler reconciles a Board object
type BoardReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=tetris.tkna.net,resources=boards,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tetris.tkna.net,resources=boards/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tetris.tkna.net,resources=boards/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Board object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *BoardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconcile Board")

	var board tetrisv1.Board
	err := r.Get(ctx, req.NamespacedName, &board)
	if errors.IsNotFound(err) {
		logger.Error(err, "Board not found", "name", req.NamespacedName)
		return ctrl.Result{}, nil
	}
	if err != nil {
		logger.Error(err, "unable to get Board", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}
	if !board.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	if board.Status.Data == nil {
		board.Status.Data = [][]uint{
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 1, 1, 1, 1, 0, 0},
		}
	}

	setCurrentMino(ctx, &board)

	err = r.Status().Update(ctx, &board)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func isCollision(board tetrisv1.Board, coords []tetrisv1.Coord) (bool) {
	for _, coord := range(coords) {
		if coord.X < 0 || coord.X >= int(board.Spec.Width) {return true}
		if coord.Y < 0 || coord.Y >= int(board.Spec.Height) {return true}
		if board.Status.Data[coord.Y][coord.X] != 0 {return true}
	}
	return false
}

func calcAbsoluteCoords(mino *tetrisv1.Mino) {
	var coords []tetrisv1.Coord
	for _, coord := range(mino.RelativeCoords) {
		c := tetrisv1.Coord{
			X: mino.Center.X + coord.X,
			Y: mino.Center.Y - coord.Y,
		}
		coords = append(coords, c)
	}
	mino.AbsoluteCoords = coords
}

func newMino(ctx context.Context, board *tetrisv1.Board) bool {
	logger := log.FromContext(ctx)
	logger.Info("newMino")

	mino := tetrisv1.Mino{
		MinoId: 1,
		Center: tetrisv1.Coord{X: 4, Y: 2},
		RelativeCoords: []tetrisv1.Coord{
			tetrisv1.Coord{X: -1, Y: 0},
			tetrisv1.Coord{X: 0, Y: 0},
			tetrisv1.Coord{X: 0, Y: 1},
			tetrisv1.Coord{X: 1, Y: 0},
		},
	}
	calcAbsoluteCoords(&mino)

	if isCollision(*board, mino.AbsoluteCoords) {
		return false
	}

	board.Status.CurrentMino = []tetrisv1.Mino{}
	board.Status.CurrentMino = append(board.Status.CurrentMino, mino)

	for _, coord := range(mino.AbsoluteCoords) {
		board.Status.Data[coord.Y][coord.X] = mino.MinoId
	}

	return true
}

func setCurrentMino(ctx context.Context, board *tetrisv1.Board) {
	logger := log.FromContext(ctx)
	logger.Info("set current mino")

	if len(board.Status.CurrentMino) == 0 {
		logger.Info("No CurrentMino. Creating...")
		if ok := newMino(ctx, board); !ok {
			logger.Info("Failed to create new mino. GameOver")
			return
		} 
	}

	logger.Info("set CurrentMino successfully")
	return
}

// SetupWithManager sets up the controller with the Manager.
func (r *BoardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tetrisv1.Board{}).
		Complete(r)
}
