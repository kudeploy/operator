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

package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kudeploycomv1 "github.com/kudeploy/operator/api/v1"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kudeploy.com,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kudeploy.com,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kudeploy.com,resources=projects/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Project object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 获取 Project 资源
	project := &kudeploycomv1.Project{}
	if err := r.Get(ctx, req.NamespacedName, project); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "无法获取 Project")
		return ctrl.Result{}, err
	}

	// 检查并初始化状态
	if project.Status.Phase == "" {
		project.Status.Phase = "Pending"
		if err := r.Status().Update(ctx, project); err != nil {
			logger.Error(err, "初始化状态失败")
			return ctrl.Result{}, err
		}
	}

	// 处理删除逻辑
	if !project.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, project)
	}

	// 检查 namespace 是否存在
	namespace := &corev1.Namespace{}
	err := r.Get(ctx, client.ObjectKey{Name: project.Name}, namespace)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "检查 namespace 失败")
			return ctrl.Result{}, err
		}
		// namespace 不存在，创建新的
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: project.Name,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(project, kudeploycomv1.GroupVersion.WithKind("Project")),
				},
				Labels: map[string]string{
					"kudeploy.com/project": project.Name,
				},
			},
		}
		if err := r.Create(ctx, namespace); err != nil {
			logger.Error(err, "创建 namespace 失败")
			return ctrl.Result{}, err
		}
	} else {
		// namespace 存在，检查所有权
		if !metav1.IsControlledBy(namespace, project) {
			msg := fmt.Sprintf("Namespace %s 已存在且不属于此 project", project.Name)
			meta.SetStatusCondition(&project.Status.Conditions, metav1.Condition{
				Type:    "NamespaceCreated",
				Status:  metav1.ConditionFalse,
				Reason:  "NamespaceOwnedByOthers",
				Message: msg,
			})
			project.Status.Phase = "Failed"
			if err := r.Status().Update(ctx, project); err != nil {
				logger.Error(err, "更新状态失败")
			}
			return ctrl.Result{}, fmt.Errorf("%s", msg)
		}
	}

	// 更新状态为 Active
	meta.SetStatusCondition(&project.Status.Conditions, metav1.Condition{
		Type:    "NamespaceCreated",
		Status:  metav1.ConditionTrue,
		Reason:  "NamespaceReady",
		Message: fmt.Sprintf("Namespace %s is ready", project.Name),
	})
	project.Status.Phase = "Active"
	if err := r.Status().Update(ctx, project); err != nil {
		logger.Error(err, "更新状态失败")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// 添加删除处理函数
func (r *ProjectReconciler) reconcileDelete(ctx context.Context, project *kudeploycomv1.Project) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 更新状态为 Terminating
	project.Status.Phase = "Terminating"
	if err := r.Status().Update(ctx, project); err != nil {
		logger.Error(err, "更新删除状态失败")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kudeploycomv1.Project{}).
		Named("project").
		Complete(r)
}
