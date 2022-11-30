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
	"fmt"
	"math/rand"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	t4sv1 "github.com/tkna/t4s/api/v1"
)

// BoardReconciler reconciles a Board object.
type BoardReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=t4s.tkna.net,resources=boards,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=t4s.tkna.net,resources=boards/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=t4s.tkna.net,resources=boards/finalizers,verbs=update
//+kubebuilder:rbac:groups=t4s.tkna.net,resources=actions,verbs=get;list;watch;create;update;patch;delete

func (r *BoardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconcile Board")

	var board t4sv1.Board
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

	// Init board.Status
	if board.Status.Data == nil {
		board.Status.Data = make([][]int, board.Spec.Height)
		for i := 0; i < board.Spec.Height; i++ {
			board.Status.Data[i] = make([]int, board.Spec.Width)
		}
		board.Status.State = board.Spec.State
	}

	if err := r.reconcileCurrentMino(ctx, &board); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileAction(ctx, &board); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileCron(ctx, &board); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Status().Update(ctx, &board); err != nil {
		logger.Error(err, "failed to update board")
		return ctrl.Result{}, err
	}

	logger.Info("reconcile Board successfully")
	return ctrl.Result{}, nil
}

func isCollision(board t4sv1.Board, coords []t4sv1.Coord) bool {
	for _, coord := range coords {
		if coord.X < 0 || coord.X >= board.Spec.Width {
			return true
		}
		if coord.Y < 0 || coord.Y >= board.Spec.Height {
			return true
		}
		if board.Status.Data[coord.Y][coord.X] != 0 {
			return true
		}
	}
	return false
}

func setAbsoluteCoords(mino *t4sv1.CurrentMino) {
	var coords []t4sv1.Coord
	for _, coord := range mino.RelativeCoords {
		c := t4sv1.Coord{
			X: mino.Center.X + coord.X,
			Y: mino.Center.Y - coord.Y,
		}
		coords = append(coords, c)
	}
	mino.AbsoluteCoords = coords
}

func (r *BoardReconciler) newMino(ctx context.Context, board *t4sv1.Board) (bool, error) {
	logger := log.FromContext(ctx)
	logger.Info("newMino")

	logger.Info("list Minoes")
	minoes := t4sv1.MinoList{}
	err := r.List(ctx, &minoes, &client.ListOptions{
		Namespace: board.Namespace,
	})
	if err != nil {
		logger.Error(err, "failed to list Minoes")
		return false, err
	}
	if len(minoes.Items) == 0 {
		logger.Info("no minoes found")
		return false, fmt.Errorf("no minoes found")
	}

	selectedMino := minoes.Items[rand.Intn(len(minoes.Items))]
	mino := t4sv1.CurrentMino{
		MinoID:         selectedMino.Spec.MinoID,
		Center:         t4sv1.Coord{X: (board.Spec.Width - 1) / 2, Y: 2},
		RelativeCoords: append([]t4sv1.Coord{}, selectedMino.Spec.Coords...),
	}
	setAbsoluteCoords(&mino)
	if isCollision(*board, mino.AbsoluteCoords) {
		return false, nil
	}
	board.Status.CurrentMino = []t4sv1.CurrentMino{}
	board.Status.CurrentMino = append(board.Status.CurrentMino, mino)

	return true, nil
}

func (r *BoardReconciler) reconcileCurrentMino(ctx context.Context, board *t4sv1.Board) error {
	logger := log.FromContext(ctx)
	logger.Info("reconcile CurrentMino")

	if board.Status.State == t4sv1.GameOver {
		logger.Info("State == GameOver")
		return nil
	}

	if len(board.Status.CurrentMino) == 0 {
		logger.Info("no current mino. creating")
		ok, err := r.newMino(ctx, board)
		if err != nil {
			return err
		}
		if !ok {
			logger.Info("failed to create a new mino. game over")
			board.Status.State = t4sv1.GameOver
		}
	}

	logger.Info("reconcile CurrentMino successfully")
	return nil
}

func moveCurrentMino(ctx context.Context, board *t4sv1.Board, op string) {
	logger := log.FromContext(ctx)
	logger.Info("move current mino", "op", op)

	mino := board.Status.CurrentMino[0].DeepCopy()

	switch op {
	case "down":
		mino.Center.Y++
		setAbsoluteCoords(&mino)
		if isCollision(*board, mino.AbsoluteCoords) {
			for _, coord := range board.Status.CurrentMino[0].AbsoluteCoords {
				board.Status.Data[coord.Y][coord.X] = board.Status.CurrentMino[0].MinoID
			}
			checkRemoveLines(ctx, board)
			board.Status.CurrentMino = nil
			logger.Info("CurrentMino landed successfully")
		} else {
			board.Status.CurrentMino[0] = mino
		}

	case "left":
		mino.Center.X--
		setAbsoluteCoords(&mino)
		if isCollision(*board, mino.AbsoluteCoords) {
			return
		}
		board.Status.CurrentMino[0] = mino

	case "right":
		mino.Center.X++
		setAbsoluteCoords(&mino)
		if isCollision(*board, mino.AbsoluteCoords) {
			return
		}
		board.Status.CurrentMino[0] = mino

	case "rotate":
		coords := []t4sv1.Coord{}
		for _, coord := range mino.RelativeCoords {
			newCoord := t4sv1.Coord{X: coord.Y, Y: -coord.X}
			coords = append(coords, newCoord)
		}
		mino.RelativeCoords = coords
		setAbsoluteCoords(&mino)
		if isCollision(*board, mino.AbsoluteCoords) {
			return
		}
		board.Status.CurrentMino[0] = mino

	case "drop":
		var minoFrom t4sv1.CurrentMino
		for {
			minoFrom = mino.DeepCopy()
			mino.Center.Y++
			setAbsoluteCoords(&mino)
			if isCollision(*board, mino.AbsoluteCoords) {
				break
			}
		}
		for _, coord := range minoFrom.AbsoluteCoords {
			board.Status.Data[coord.Y][coord.X] = board.Status.CurrentMino[0].MinoID
		}
		board.Status.CurrentMino[0] = minoFrom
		// "drop" does not fix the current mino and delegate it to "down" to get time to render when the line is full
	}

	logger.Info("move CurrentMino successfully")
}

func checkRemoveLines(ctx context.Context, board *t4sv1.Board) {
	logger := log.FromContext(ctx)
	logger.Info("check and remove lines")

	// Calc Ys of the lines to be removed
	removeYs := make(map[int]bool)
	for _, coord := range board.Status.CurrentMino[0].AbsoluteCoords {
		y := coord.Y
		if !removeYs[y] {
			full := true
			for x := 0; x < board.Spec.Width; x++ {
				if board.Status.Data[y][x] == 0 {
					full = false
					break
				}
			}
			if full {
				removeYs[y] = true
			}
		}
	}

	if len(removeYs) == 0 {
		logger.Info("no lines to remove")
		return
	}

	// Drop lines except the ones to be removed
	newY := board.Spec.Height - 1
	for y := board.Spec.Height - 2; y >= 0; y-- {
		if removeYs[y] {
			continue
		}
		copy(board.Status.Data[newY], board.Status.Data[y])
		newY--
	}

	logger.Info("check and remove lines successfully", "removed lines", len(removeYs))
}

func (r *BoardReconciler) reconcileAction(ctx context.Context, board *t4sv1.Board) error {
	logger := log.FromContext(ctx)
	logger.Info("reconcile Action")

	logger.Info("list Actions")
	actions := t4sv1.ActionList{}
	err := r.List(ctx, &actions, &client.ListOptions{
		Namespace: board.Namespace,
	})
	if err != nil {
		logger.Error(err, "failed to list Actions")
		return err
	}

	if len(actions.Items) != 0 {
		// Process the first Action only
		action := actions.Items[0]
		logger.Info("Action found", "name", action.GetName())
		if board.Status.State != t4sv1.GameOver {
			moveCurrentMino(ctx, board, action.Spec.Op)
		}
		for _, action := range actions.Items {
			logger.Info("delete Action", "name", action.GetName())
			err = r.Delete(ctx, &action)
			if err != nil {
				logger.Error(err, "failed to delete action")
				return err
			}
		}
	}

	logger.Info("reconcile Action successfully")
	return nil
}

func (r *BoardReconciler) reconcileCron(ctx context.Context, board *t4sv1.Board) error {
	logger := log.FromContext(ctx)
	logger.Info("reconcile Cron")

	if board.Status.State == t4sv1.Playing {
		cron := &t4sv1.Cron{}
		cron.SetNamespace(board.Namespace)
		cron.SetName("cron")
		op, err := ctrl.CreateOrUpdate(ctx, r.Client, cron, func() error {
			cron.Spec.Period = board.Spec.Wait
			return ctrl.SetControllerReference(board, cron, r.Scheme)
		})
		if err != nil {
			logger.Error(err, "unable to create or update Cron")
			return err
		}
		if op != controllerutil.OperationResultNone {
			logger.Info("reconcile Cron successfully", "op", op)
		}
	} else {
		// when board.Status.State == GameOver
		var cron t4sv1.Cron
		err := r.Get(ctx, client.ObjectKey{
			Namespace: board.Namespace,
			Name:      "cron",
		}, &cron)
		if errors.IsNotFound(err) {
			logger.Info("Cron not found")
			return nil
		}
		if err != nil {
			logger.Error(err, "unable to get Cron")
			return err
		}
		if !cron.ObjectMeta.DeletionTimestamp.IsZero() {
			logger.Info("DeletionTimestamp is not zero", cron.ObjectMeta.DeletionTimestamp)
			return nil
		}
		if err := r.Delete(ctx, &cron); err != nil {
			logger.Error(err, "failed to delete cron")
			return err
		}
	}

	logger.Info("reconcile Cron successfully")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BoardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&t4sv1.Board{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Owns(&t4sv1.Action{}, builder.WithPredicates(
			// ignore deletion of Action
			predicate.Funcs{
				DeleteFunc: func(e event.DeleteEvent) bool {
					return false
				},
			})).
		Complete(r)
}
