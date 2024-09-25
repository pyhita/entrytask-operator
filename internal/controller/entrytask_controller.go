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
	kantetaskv1 "codereliant.io/entrytask/api/v1"
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// EntryTaskReconciler reconciles a EntryTask object
type EntryTaskReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kantetask.codereliant.io,resources=entrytasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kantetask.codereliant.io,resources=entrytasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kantetask.codereliant.io,resources=entrytasks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EntryTask object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile

var podCount int = 0

const (
	containerPort = 80
	targetPort
	port
)

func (r *EntryTaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Reconciling EntryTask")
	// fetch entrytask
	entryTask := &kantetaskv1.EntryTask{}
	if err := r.Get(ctx, req.NamespacedName, entryTask); err != nil {
		// can not find entrytask
		log.Error(err, "unable to fetch EntryTask", "namespace", req.Namespace, "name", req.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ensure pods
	if err := r.ensurePods(ctx, entryTask, entryTask.Namespace); err != nil {
		log.Error(err, "unable to ensure Pods")
		return ctrl.Result{}, err
	}

	// ensure service
	if err := r.ensureService(ctx, entryTask, entryTask.Namespace); err != nil {
		log.Error(err, "unable to ensure Service")
		return ctrl.Result{}, err
	}

	// 维护 status 信息

	return ctrl.Result{}, nil
}

func (r *EntryTaskReconciler) ensurePods(ctx context.Context, entryTask *kantetaskv1.EntryTask, Namespace string) error {
	log := log.FromContext(ctx)

	// 使用选择器获取相关的 Pods
	selector, err := metav1.LabelSelectorAsSelector(entryTask.Spec.Selector)
	if err != nil {
		return err
	}

	// 创建一个 Pod 列表
	podList := &v1.PodList{}
	// 获取指定命名空间下的 Pods
	if err := r.List(ctx, podList, &client.ListOptions{
		Namespace:     Namespace, // 指定命名空间
		LabelSelector: selector,  // 使用选择器
	}); err != nil {
		return err
	}

	desiredPodCount := entryTask.Spec.DesiredReplicas
	currentPodCount := int32(len(podList.Items))

	log.Info("log pod count: ", "currentPodCount", currentPodCount)
	for currentPodCount < desiredPodCount {
		log.Info("currentPodCount < desiredPodCount, create new pod", "desiredPodCount", desiredPodCount, "currentPodCount", currentPodCount)
		// 创建新 Pod 的代码示例
		newPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-pod-%s", entryTask.Name, uuid.NewUUID()), // 为新 Pod 生成唯一名称
				Namespace: Namespace,
				Labels:    entryTask.Spec.Selector.MatchLabels, // 设置 Pod 的标签
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(entryTask, kantetaskv1.GroupVersion.WithKind("EntryTask")),
				},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  entryTask.Spec.Template.Spec.Containers[0].Name,
						Image: entryTask.Spec.Template.Spec.Containers[0].Image,
						Ports: []v1.ContainerPort{{ContainerPort: containerPort}},
					},
				},
			},
		}

		// 创建 Pod
		if err := r.Create(ctx, newPod); err != nil {
			log.Error(err, "Failed to create Pod")
			return err
		}
		podCount++
		currentPodCount++
		// add new pod to podList
		podList.Items = append(podList.Items, *newPod)
		// debug 延缓pod的创建速度
		//time.Sleep(1 * time.Second)
		log.Info("Created new Pod", "Pod.Name", newPod.Name)
	}

	return nil
}

func (r *EntryTaskReconciler) ensureService(ctx context.Context, entryTask *kantetaskv1.EntryTask, serviceName string) error {
	log := log.FromContext(ctx)
	service := &v1.Service{}

	err := r.Get(ctx, client.ObjectKey{Name: serviceName, Namespace: entryTask.Namespace}, service)
	if err != nil {
		if errors.IsNotFound(err) {
			// create a new service
			log.Info("Create new service ", "serviceName", serviceName)
			service := &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: entryTask.Namespace,
				},
				Spec: v1.ServiceSpec{
					Selector: entryTask.Spec.Selector.MatchLabels,
					Ports: []v1.ServicePort{{
						Protocol:   v1.ProtocolTCP,
						Port:       port,
						TargetPort: intstr.FromInt32(targetPort),
					}},
				},
			}

			if err := r.Create(ctx, service); err != nil {
				log.Error(err, "Failed to create new service", "serviceName", serviceName)
				return err
			}
		} else {
			return err
		}
	} else {
		// service already exists
		log.Info("Service already exists", "service", serviceName)
	}

	return nil
}

//func (r *EntryTaskReconciler) ensureNamespace(ctx context.Context, tenant *.Tenant, namespaceName string) error {
//	log := log.FromContext(ctx)
//
//	namespace := &v1.Namespace{}
//
//	err := r.Get(ctx, client.ObjectKey{Name: namespaceName}, namespace)
//	if err != nil {
//		// If the namespace doesn't exist, create it
//		if errors.IsNotFound(err) {
//			log.Info("Creating Namespace", "namespace", namespaceName)
//			namespace := &v1.Namespace{
//				ObjectMeta: metav1.ObjectMeta{
//					Name: namespaceName,
//					Annotations: map[string]string{
//						"adminEmail": tenant.Spec.AdminEmail,
//						"managed-by": tenantOperatorAnnotation,
//					},
//				},
//			}
//
//			// Attempt to create the namespace
//			if err = r.Create(ctx, namespace); err != nil {
//				return err
//			}
//		} else {
//			return err
//		}
//	} else {
//		// If the namespace already exists, check for required annotations
//		log.Info("Namespace already exists", "namespace", namespaceName)
//
//		// Logic for checking annotations
//	}
//
//	return nil
//}

// SetupWithManager sets up the controller with the Manager.
func (r *EntryTaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	//return ctrl.NewControllerManagedBy(mgr).
	//For(&kantetaskv1.EntryTask{}).
	//Complete(r)
	//
	//
	//// 如果集群中 有pod 发生了变化，触发 调谐逻辑
	////return ctrl.NewControllerManagedBy(mgr).
	////	For(&v1.Pod{}).
	////	Complete(r)

	// 被 selector 选中的pod发生变化时，可以收到通知
	return ctrl.NewControllerManagedBy(mgr).
		For(&kantetaskv1.EntryTask{}).
		Owns(&v1.Pod{}).
		Owns(&v1.Service{}).
		Complete(r)
}
