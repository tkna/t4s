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
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&T4s{}).
		WithValidator(&t4sValidator{client: mgr.GetClient()}).
		Complete()
}

//+kubebuilder:webhook:path=/validate-t4s-tkna-net-v1-t4s,mutating=false,failurePolicy=fail,sideEffects=None,groups=t4s.tkna.net,resources=t4s,verbs=create;update,versions=v1,name=vt4s.kb.io,admissionReviewVersions=v1

type t4sValidator struct {
	client client.Client
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (v t4sValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	logger := log.FromContext(ctx)
	t4s := obj.(*T4s)
	logger.Info("validate create", "name", t4s.Name)

	t4sList := &T4sList{}
	if err := v.client.List(ctx, t4sList, &client.ListOptions{Namespace: t4s.Namespace}); err != nil {
		logger.Error(err, "failed to list t4s", "name", t4s.Name)
		return err
	}
	if len(t4sList.Items) > 0 {
		err := fmt.Errorf("T4s is not allowed to be created more than 2 in one namespace. namespace: %v", t4s.Namespace)
		logger.Error(err, "failed to create T4s", "name", t4s.Name)
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (v t4sValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (v t4sValidator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}
