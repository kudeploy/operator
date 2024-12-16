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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kudeploycomv1 "github.com/kudeploy/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kudeploy.com,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kudeploy.com,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kudeploy.com,resources=services/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Service object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// 获取 Service CR
	var service kudeploycomv1.Service
	if err := r.Get(ctx, req.NamespacedName, &service); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	labels := map[string]string{
		"kudeploy.com/service": service.Name,
	}

	// 构造期望的 Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      service.Name,
			Namespace: service.Namespace,
			Labels:    labels,
		},
	}

	// 创建或更新 Deployment
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		// 设置 deployment spec
		replicas := int32(1)

		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            service.Name,
							Image:           service.Spec.Image,
							ImagePullPolicy: corev1.PullPolicy(service.Spec.ImagePullPolicy),
							Ports:           convertToContainerPorts(service.Spec.Ports),
							Env:             convertToContainerEnv(service.Spec.Env),
							Resources:       *service.Spec.Resources,
						},
					},
				},
			},
		}

		// 设置所有者引用
		return ctrl.SetControllerReference(&service, deployment, r.Scheme)
	})

	if err != nil {
		log.Error(err, "Failed to create or update Deployment")
		return ctrl.Result{}, err
	}

	log.Info("Deployment reconciled", "operation", op)

	// 创建 Kubernetes Service
	k8sService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      service.Name,
			Namespace: service.Namespace,
			Labels:    labels,
		},
	}

	// 创建或更新 Kubernetes Service
	svcOp, err := ctrl.CreateOrUpdate(ctx, r.Client, k8sService, func() error {
		// 设置标签
		k8sService.Labels = map[string]string{
			"kudeploy.com/service": service.Name,
		}

		// 设置 Service spec
		k8sService.Spec = corev1.ServiceSpec{
			Selector: labels,
			Ports:    convertToServicePorts(service.Spec.Ports),
			Type:     corev1.ServiceTypeClusterIP,
		}

		// 设置所有者引用
		return ctrl.SetControllerReference(&service, k8sService, r.Scheme)
	})

	if err != nil {
		log.Error(err, "Failed to create or update Kubernetes Service")
		return ctrl.Result{}, err
	}

	log.Info("Kubernetes Service reconciled", "operation", svcOp)
	return ctrl.Result{}, nil
}

func convertToContainerEnv(envs []kudeploycomv1.EnvVar) []corev1.EnvVar {
	var containerEnvs []corev1.EnvVar
	for _, env := range envs {
		containerEnvs = append(containerEnvs, corev1.EnvVar{
			Name:  env.Name,
			Value: env.Value,
		})
	}
	return containerEnvs
}

// 转换 ServicePort 到 corev1.ContainerPort
func convertToContainerPorts(servicePorts []kudeploycomv1.ServicePort) []corev1.ContainerPort {
	var containerPorts []corev1.ContainerPort
	for _, port := range servicePorts {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name:          fmt.Sprintf("port-%d", port.ContainerPort),
			Protocol:      corev1.Protocol(port.Protocol),
			ContainerPort: port.ContainerPort,
		})
	}
	return containerPorts
}

// 转换 ServicePort 到 corev1.ServicePort
func convertToServicePorts(servicePorts []kudeploycomv1.ServicePort) []corev1.ServicePort {
	var k8sServicePorts []corev1.ServicePort
	for _, port := range servicePorts {
		k8sServicePorts = append(k8sServicePorts, corev1.ServicePort{
			Name:       fmt.Sprintf("port-%d", port.ContainerPort),
			Protocol:   corev1.Protocol(port.Protocol),
			Port:       port.ServicePort,
			TargetPort: intstr.FromInt(int(port.ContainerPort)),
		})
	}
	return k8sServicePorts
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kudeploycomv1.Service{}).
		Named("service").
		Complete(r)
}
