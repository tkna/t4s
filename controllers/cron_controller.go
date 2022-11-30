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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	t4sv1 "github.com/tkna/t4s/api/v1"
)

// CronReconciler reconciles a Cron object.
type CronReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=t4s.tkna.net,resources=crons,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=t4s.tkna.net,resources=crons/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=t4s.tkna.net,resources=crons/finalizers,verbs=update
//+kubebuilder:rbac:groups=t4s.tkna.net,resources=actions,verbs=get;list;watch;create;update;patch;delete

func (r *CronReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconcile Cron")

	var cron t4sv1.Cron
	err := r.Get(ctx, req.NamespacedName, &cron)
	if errors.IsNotFound(err) {
		logger.Info("Cron not found")
		return ctrl.Result{}, nil
	}
	if err != nil {
		logger.Error(err, "unable to get Cron", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}
	if !cron.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.Info("DeletionTimestamp is not zero", cron.ObjectMeta.DeletionTimestamp)
		return ctrl.Result{}, nil
	}

	action := t4sv1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    cron.Namespace,
			GenerateName: "action-",
		},
		Spec: t4sv1.ActionSpec{
			Op: "down",
		},
	}
	action.SetOwnerReferences(cron.GetOwnerReferences())

	logger.Info("creating Action")
	err = r.Create(ctx, &action)
	if err != nil {
		logger.Error(err, "failed to create Action")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Millisecond * time.Duration(cron.Spec.Period)}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CronReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&t4sv1.Cron{}).
		Complete(r)
}
