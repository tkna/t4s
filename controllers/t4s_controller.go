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
	"bytes"
	"context"
	"io"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
	rbacv1apply "k8s.io/client-go/applyconfigurations/rbac/v1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	t4sv1 "github.com/tkna/t4s/api/v1"
	"github.com/tkna/t4s/pkg/constants"
)

// T4sReconciler reconciles a T4s object.
type T4sReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=t4s.tkna.net,resources=t4s,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=t4s.tkna.net,resources=t4s/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=t4s.tkna.net,resources=t4s/finalizers,verbs=update
//+kubebuilder:rbac:groups=t4s.tkna.net,resources=boards,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=t4s.tkna.net,resources=actions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=t4s.tkna.net,resources=minoes,verbs=get;list;watch;create;update;patch;delete

func (r *T4sReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var t4s t4sv1.T4s
	err := r.Get(ctx, req.NamespacedName, &t4s)
	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		logger.Error(err, "unable to get T4s")
		return ctrl.Result{}, err
	}
	if !t4s.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	if err := r.reconcileMino(ctx, t4s); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileBoard(ctx, t4s); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileApp(ctx, t4s); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("reconcile T4s successfully")
	return ctrl.Result{}, nil
}

func (r *T4sReconciler) reconcileBoard(ctx context.Context, t4s t4sv1.T4s) error {
	logger := log.FromContext(ctx)

	board := &t4sv1.Board{}
	err := r.Get(ctx, client.ObjectKey{Namespace: t4s.Namespace, Name: constants.BoardName}, board)
	notFound := errors.IsNotFound(err)
	if err != nil && !notFound {
		logger.Error(err, "failed to get Board")
		return err
	}
	if !board.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.Info("Board is being deleted")
		return nil
	}

	needsRecreation := t4s.Spec.Width != board.Spec.Width || t4s.Spec.Height != board.Spec.Height
	needsUpdate := t4s.Spec.Wait != board.Spec.Wait

	if !notFound && needsRecreation {
		if err := r.Delete(ctx, board); err != nil {
			logger.Error(err, "failed to delete Board")
			return err
		}
	}
	if notFound || needsRecreation {
		board := &t4sv1.Board{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: t4s.Namespace,
				Name:      constants.BoardName,
			},
			Spec: t4sv1.BoardSpec{
				Width:  t4s.Spec.Width,
				Height: t4s.Spec.Height,
				Wait:   t4s.Spec.Wait,
			},
		}
		if err := ctrl.SetControllerReference(&t4s, board, r.Scheme); err != nil {
			logger.Error(err, "failed to set controller reference")
			return err
		}
		if err := r.Create(ctx, board); err != nil {
			logger.Error(err, "failed to create Board")
			return err
		}
	} else if needsUpdate {
		board.Spec.Wait = t4s.Spec.Wait
		if err := r.Update(ctx, board); err != nil {
			logger.Error(err, "failed to update Board")
			return err
		}
	}

	logger.Info("reconcile Board successfully")
	return nil
}

func (r *T4sReconciler) reconcileApp(ctx context.Context, t4s t4sv1.T4s) error {
	logger := log.FromContext(ctx)
	owner, err := ownerRef(t4s, r.Scheme)
	if err != nil {
		return err
	}
	label := map[string]string{
		"tier": "app",
	}

	// ServiceAccount
	saName := "t4s-app-sa"
	sa := corev1apply.ServiceAccount(saName, t4s.Namespace).
		WithLabels(label).
		WithOwnerReferences(owner)

	if err := r.reconcileServiceAccount(ctx, t4s, sa); err != nil {
		return err
	}

	// Role
	t4sRoleName := "t4s-viewer-role"
	role := rbacv1apply.Role(t4sRoleName, t4s.Namespace).
		WithLabels(label).
		WithOwnerReferences(owner).
		WithRules(
			rbacv1apply.PolicyRule().
				WithAPIGroups(t4sv1.GroupVersion.Group).
				WithResources("t4s", "t4s").
				WithVerbs("get", "list", "watch"),
		)

	if err := r.reconcileRole(ctx, t4s, role); err != nil {
		return err
	}

	boardRoleName := "board-editor-role"
	role = rbacv1apply.Role(boardRoleName, t4s.Namespace).
		WithLabels(label).
		WithOwnerReferences(owner).
		WithRules(
			rbacv1apply.PolicyRule().
				WithAPIGroups(t4sv1.GroupVersion.Group).
				WithResources("boards", "boards").
				WithVerbs("get", "list", "watch", "create", "delete", "patch", "update"),
			rbacv1apply.PolicyRule().
				WithAPIGroups(t4sv1.GroupVersion.Group).
				WithResources("boards", "boards/status").
				WithVerbs("get"),
		)

	if err := r.reconcileRole(ctx, t4s, role); err != nil {
		return err
	}

	actionRoleName := "action-editor-role"
	role = rbacv1apply.Role(actionRoleName, t4s.Namespace).
		WithLabels(label).
		WithOwnerReferences(owner).
		WithRules(
			rbacv1apply.PolicyRule().
				WithAPIGroups(t4sv1.GroupVersion.Group).
				WithResources("actions", "actions").
				WithVerbs("get", "list", "watch", "create", "delete", "patch", "update"),
		)

	if err := r.reconcileRole(ctx, t4s, role); err != nil {
		return err
	}

	minoRoleName := "mino-viewer-role"
	role = rbacv1apply.Role(minoRoleName, t4s.Namespace).
		WithLabels(label).
		WithOwnerReferences(owner).
		WithRules(
			rbacv1apply.PolicyRule().
				WithAPIGroups(t4sv1.GroupVersion.Group).
				WithResources("minoes", "minoes").
				WithVerbs("get", "list", "watch"),
		)

	if err := r.reconcileRole(ctx, t4s, role); err != nil {
		return err
	}

	// RoleBinding
	rbName := "t4s-viewer-rb"
	rb := rbacv1apply.RoleBinding(rbName, t4s.Namespace).
		WithLabels(label).
		WithOwnerReferences(owner).
		WithRoleRef(rbacv1apply.RoleRef().
			WithAPIGroup(rbacv1.SchemeGroupVersion.Group).
			WithKind("Role").
			WithName(t4sRoleName)).
		WithSubjects(rbacv1apply.Subject().
			WithKind("ServiceAccount").
			WithName(saName).
			WithNamespace(t4s.Namespace))

	if err := r.reconcileRoleBinding(ctx, t4s, rb); err != nil {
		return err
	}

	rbName = "board-editor-rb"
	rb = rbacv1apply.RoleBinding(rbName, t4s.Namespace).
		WithLabels(label).
		WithOwnerReferences(owner).
		WithRoleRef(rbacv1apply.RoleRef().
			WithAPIGroup(rbacv1.SchemeGroupVersion.Group).
			WithKind("Role").
			WithName(boardRoleName)).
		WithSubjects(rbacv1apply.Subject().
			WithKind("ServiceAccount").
			WithName(saName).
			WithNamespace(t4s.Namespace))

	if err := r.reconcileRoleBinding(ctx, t4s, rb); err != nil {
		return err
	}

	rbName = "action-editor-rb"
	rb = rbacv1apply.RoleBinding(rbName, t4s.Namespace).
		WithLabels(label).
		WithOwnerReferences(owner).
		WithRoleRef(rbacv1apply.RoleRef().
			WithAPIGroup(rbacv1.SchemeGroupVersion.Group).
			WithKind("Role").
			WithName(actionRoleName)).
		WithSubjects(rbacv1apply.Subject().
			WithKind("ServiceAccount").
			WithName(saName).
			WithNamespace(t4s.Namespace))

	if err := r.reconcileRoleBinding(ctx, t4s, rb); err != nil {
		return err
	}

	rbName = "mino-viewer-rb"
	rb = rbacv1apply.RoleBinding(rbName, t4s.Namespace).
		WithLabels(label).
		WithOwnerReferences(owner).
		WithRoleRef(rbacv1apply.RoleRef().
			WithAPIGroup(rbacv1.SchemeGroupVersion.Group).
			WithKind("Role").
			WithName(minoRoleName)).
		WithSubjects(rbacv1apply.Subject().
			WithKind("ServiceAccount").
			WithName(saName).
			WithNamespace(t4s.Namespace))

	if err := r.reconcileRoleBinding(ctx, t4s, rb); err != nil {
		return err
	}

	// Deployment
	depName := "t4s-app"
	dep := appsv1apply.Deployment(depName, t4s.Namespace).
		WithLabels(label).
		WithOwnerReferences(owner).
		WithSpec(appsv1apply.DeploymentSpec().
			WithReplicas(1).
			WithSelector(metav1apply.LabelSelector().WithMatchLabels(label)).
			WithTemplate(corev1apply.PodTemplateSpec().
				WithLabels(label).
				WithSpec(corev1apply.PodSpec().
					WithServiceAccountName(saName).
					WithContainers(corev1apply.Container().
						WithName(constants.BoardName).
						WithImage(constants.AppImage).
						WithImagePullPolicy(corev1.PullIfNotPresent).
						WithPorts(corev1apply.ContainerPort().
							WithName("http").
							WithProtocol(corev1.ProtocolTCP).
							WithContainerPort(8000),
						).
						WithEnv(corev1apply.EnvVar().
							WithName("NAMESPACE").
							WithValue(t4s.Namespace),
						).
						WithEnv(corev1apply.EnvVar().
							WithName("T4S_NAME").
							WithValue(t4s.Name),
						).
						WithEnv(corev1apply.EnvVar().
							WithName("BOARD_NAME").
							WithValue(constants.BoardName),
						),
					),
				),
			),
		)

	if err := r.reconcileDeployment(ctx, t4s, dep); err != nil {
		return err
	}

	// Service
	svcName := "t4s-app"
	svcType := corev1.ServiceTypeNodePort
	if t4s.Spec.ServiceType == "LoadBalancer" {
		svcType = corev1.ServiceTypeLoadBalancer
	}
	port := corev1apply.ServicePort().
		WithProtocol(corev1.ProtocolTCP).
		WithPort(8000).
		WithTargetPort(intstr.FromInt(8000))
	if t4s.Spec.NodePort != 0 {
		port = port.WithNodePort(t4s.Spec.NodePort)
	}
	spec := corev1apply.ServiceSpec().
		WithSelector(label).
		WithType(svcType).
		WithPorts(port)
	if t4s.Spec.LoadBalancerIP != "" {
		spec = spec.WithLoadBalancerIP(t4s.Spec.LoadBalancerIP)
	}
	if len(t4s.Spec.LoadBalancerSourceRanges) != 0 {
		spec = spec.WithLoadBalancerSourceRanges(t4s.Spec.LoadBalancerSourceRanges...)
	}
	svc := corev1apply.Service(svcName, t4s.Namespace).
		WithLabels(label).
		WithOwnerReferences(owner).
		WithSpec(spec)

	if err := r.reconcileService(ctx, t4s, svc); err != nil {
		return err
	}

	logger.Info("reconcile App successfully")
	return nil
}

func (r *T4sReconciler) reconcileDeployment(ctx context.Context, t4s t4sv1.T4s, dep *appsv1apply.DeploymentApplyConfiguration) error {
	logger := log.FromContext(ctx)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(dep)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current appsv1.Deployment
	err = r.Get(ctx, client.ObjectKey{Namespace: t4s.Namespace, Name: *dep.Name}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	currApplyConfig, err := appsv1apply.ExtractDeployment(&current, "t4s-controller")
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(dep, currApplyConfig) {
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: "t4s-controller",
		Force:        pointer.Bool(true),
	})
	if err != nil {
		logger.Error(err, "unable to create or update Deployment")
		return err
	}

	logger.Info("reconcile Deployment successfully", "name", *dep.Name)
	return nil
}

func (r *T4sReconciler) reconcileService(ctx context.Context, t4s t4sv1.T4s, svc *corev1apply.ServiceApplyConfiguration) error {
	logger := log.FromContext(ctx)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(svc)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current corev1.Service
	err = r.Get(ctx, client.ObjectKey{Namespace: t4s.Namespace, Name: *svc.Name}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	currApplyConfig, err := corev1apply.ExtractService(&current, "t4s-controller")
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(svc, currApplyConfig) {
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: "t4s-controller",
		Force:        pointer.Bool(true),
	})
	if err != nil {
		logger.Error(err, "unable to create or update Service")
		return err
	}

	logger.Info("reconcile Service successfully", "name", *svc.Name)
	return nil
}

func (r *T4sReconciler) reconcileServiceAccount(ctx context.Context, t4s t4sv1.T4s, sa *corev1apply.ServiceAccountApplyConfiguration) error {
	logger := log.FromContext(ctx)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(sa)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current corev1.ServiceAccount
	err = r.Get(ctx, client.ObjectKey{Namespace: t4s.Namespace, Name: *sa.Name}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	currApplyConfig, err := corev1apply.ExtractServiceAccount(&current, "t4s-controller")
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(sa, currApplyConfig) {
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: "t4s-controller",
		Force:        pointer.Bool(true),
	})
	if err != nil {
		logger.Error(err, "unable to create or update ServiceAccount")
		return err
	}

	logger.Info("reconcile ServiceAccount successfully", "name", *sa.Name)
	return nil
}

func (r *T4sReconciler) reconcileRole(ctx context.Context, t4s t4sv1.T4s, role *rbacv1apply.RoleApplyConfiguration) error {
	logger := log.FromContext(ctx)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(role)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current rbacv1.Role
	err = r.Get(ctx, client.ObjectKey{Namespace: t4s.Namespace, Name: *role.Name}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	currApplyConfig, err := rbacv1apply.ExtractRole(&current, "t4s-controller")
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(role, currApplyConfig) {
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: "t4s-controller",
		Force:        pointer.Bool(true),
	})
	if err != nil {
		logger.Error(err, "unable to create or update Role")
		return err
	}

	logger.Info("reconcile Role successfully", "name", *role.Name)
	return nil
}

func (r *T4sReconciler) reconcileRoleBinding(ctx context.Context, t4s t4sv1.T4s, rb *rbacv1apply.RoleBindingApplyConfiguration) error {
	logger := log.FromContext(ctx)

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(rb)
	if err != nil {
		return err
	}
	patch := &unstructured.Unstructured{
		Object: obj,
	}

	var current rbacv1.RoleBinding
	err = r.Get(ctx, client.ObjectKey{Namespace: t4s.Namespace, Name: *rb.Name}, &current)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	currApplyConfig, err := rbacv1apply.ExtractRoleBinding(&current, "t4s-controller")
	if err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(rb, currApplyConfig) {
		return nil
	}

	err = r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: "t4s-controller",
		Force:        pointer.Bool(true),
	})
	if err != nil {
		logger.Error(err, "unable to create or update RoleBinding")
		return err
	}

	logger.Info("reconcile RoleBinding successfully", "name", *rb.Name)
	return nil
}

func (r *T4sReconciler) reconcileMino(ctx context.Context, t4s t4sv1.T4s) error {
	logger := log.FromContext(ctx)

	minoConf := constants.DefaultMinoConf
	if constants.MinoConf != "" {
		minoConf = constants.MinoConf
	}

	yml, err := os.ReadFile(minoConf)
	if err != nil {
		logger.Error(err, "unable to read mino yaml")
		return err
	}

	reader := bytes.NewReader(yml)
	decoder := yamlutil.NewYAMLToJSONDecoder(reader)
	for {
		mino := &t4sv1.Mino{}
		if err = decoder.Decode(mino); err != nil {
			break
		}
		logger.Info("Reading mino from yaml file", "mino.GetName()", mino.GetName())

		m := &t4sv1.Mino{}
		m.SetNamespace(t4s.Namespace)
		m.SetName(mino.GetName())

		op, err := ctrl.CreateOrUpdate(ctx, r.Client, m, func() error {
			m.Spec.MinoID = mino.Spec.MinoID
			m.Spec.Coords = mino.Spec.Coords
			m.Spec.Color = mino.Spec.Color
			return ctrl.SetControllerReference(&t4s, m, r.Scheme)
		})
		if err != nil {
			logger.Error(err, "unable to create or update Mino")
			return err
		}
		if op != controllerutil.OperationResultNone {
			logger.Info("reconcile Mino successfully", "op", op)
		}
	}
	if err != io.EOF {
		logger.Error(err, "an error occured while reading mino yaml")
		return err
	}

	logger.Info("reconcile All Minoes successfully")
	return nil
}

func ownerRef(t4s t4sv1.T4s, scheme *runtime.Scheme) (*metav1apply.OwnerReferenceApplyConfiguration, error) {
	gvk, err := apiutil.GVKForObject(&t4s, scheme)
	if err != nil {
		return nil, err
	}
	ref := metav1apply.OwnerReference().
		WithAPIVersion(gvk.GroupVersion().String()).
		WithKind(gvk.Kind).
		WithName(t4s.Name).
		WithUID(t4s.GetUID()).
		WithBlockOwnerDeletion(true).
		WithController(true)
	return ref, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *T4sReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&t4sv1.T4s{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}
