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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	defaultPort   = 8089
	finalizerName = "entrytask.coderliant.io/finalizer"
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

	if entryTask.DeletionTimestamp.IsZero() {
		// not deleted, add finalizer
		if !controllerutil.ContainsFinalizer(entryTask, finalizerName) {
			entryTask.ObjectMeta.Finalizers = append(entryTask.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(ctx, entryTask); err != nil {
				log.Error(err, "Failed to add finalizer")
				return ctrl.Result{}, err
			}
		}

		// ensure pods
		err, podList := r.ensurePods(ctx, entryTask, entryTask.Namespace)
		if err != nil {
			log.Error(err, "unable to ensure Pods")
			return ctrl.Result{}, err
		}

		// ensure service
		if err := r.ensureService(ctx, entryTask, entryTask.Name+"-service", podList.Items[0]); err != nil {
			log.Error(err, "unable to ensure Service")
			return ctrl.Result{}, err
		}

		// update entrytask status with current state
		entryTask.Status.ActualReplicas = int32(len(podList.Items))
		endpoints := make([]string, len(podList.Items))
		for _, pod := range podList.Items {
			endpoints = append(endpoints, fmt.Sprintf("%s:%d", pod.Status.PodIP, pod.Spec.Containers[0].Ports[0].ContainerPort))
		}
		entryTask.Status.Endpoints = endpoints
		if err := r.Status().Update(ctx, entryTask); err != nil {
			log.Error(err, "unable to update EntryTask status")
			return ctrl.Result{}, err
		}
		log.Info("Successfully reconciled EntryTask")
	} else {
		if controllerutil.ContainsFinalizer(entryTask, finalizerName) {
			// clean up resources
			log.Info("Finalizer found, clean up resources")
			if err := r.deleteResources(ctx, entryTask); err != nil {
				log.Error(err, "Failed to delete resources")
				return ctrl.Result{}, err
			}

			// remove finalizer
			controllerutil.RemoveFinalizer(entryTask, finalizerName)
			if err := r.Update(ctx, entryTask); err != nil {
				log.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *EntryTaskReconciler) ensurePods(ctx context.Context, entryTask *kantetaskv1.EntryTask, Namespace string) (error, *v1.PodList) {
	log := log.FromContext(ctx)

	// 使用选择器获取相关的 Pods
	selector, err := metav1.LabelSelectorAsSelector(entryTask.Spec.Selector)
	if err != nil {
		return err, nil
	}

	// 创建一个 Pod 列表
	podList := &v1.PodList{}
	// 获取指定命名空间下的 Pods
	if err := r.List(ctx, podList, &client.ListOptions{
		Namespace:     Namespace, // 指定命名空间
		LabelSelector: selector,  // 使用选择器
	}); err != nil {
		return err, nil
	}

	desiredPodCount := entryTask.Spec.DesiredReplicas
	currentPodCount := int32(len(podList.Items))

	//log.Info("log pod count: ", "currentPodCount", currentPodCount)
	//log.Info("container info: ", "containerName", entryTask.Spec.Name, "containerPort", entryTask.Spec.Port, "containerImage", entryTask.Spec.Image, "desiredReplicas", entryTask.Spec.DesiredReplicas)
	for currentPodCount < desiredPodCount {
		log.Info("currentPodCount < desiredPodCount, create new pod", "desiredPodCount", desiredPodCount, "currentPodCount", currentPodCount)
		newPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("entrytask-pod-%s", uuid.NewUUID()), // 为新 Pod 生成唯一名称
				Namespace: Namespace,
				Labels:    entryTask.Spec.Selector.MatchLabels, // 设置 Pod 的标签
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(entryTask, kantetaskv1.GroupVersion.WithKind("EntryTask")),
				},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  entryTask.Spec.Name,
						Image: entryTask.Spec.Image,
						Ports: []v1.ContainerPort{{ContainerPort: entryTask.Spec.Port}},
					},
				},
			},
		}

		// 创建 Pod
		if err := r.Create(ctx, newPod); err != nil {
			log.Error(err, "Failed to create Pod")
			return err, podList
		}
		podCount++
		currentPodCount++
		// add new pod to podList
		podList.Items = append(podList.Items, *newPod)
		// debug 延缓pod的创建速度
		//time.Sleep(1 * time.Second)
		log.Info("Created new Pod", "Pod.Name", newPod.Name)
	}

	return nil, podList
}

func (r *EntryTaskReconciler) ensureService(ctx context.Context, entryTask *kantetaskv1.EntryTask, serviceName string, pod v1.Pod) error {
	log := log.FromContext(ctx)
	service := &v1.Service{}

	err := r.Get(ctx, client.ObjectKey{Name: serviceName, Namespace: entryTask.Namespace}, service)
	if err != nil {
		if errors.IsNotFound(err) {
			// create a new service
			log.Info("Create new service ", "serviceName", serviceName)

			servicePort := func() int32 {
				if entryTask.Spec.ServicePort != 0 {
					return entryTask.Spec.ServicePort
				}
				return defaultPort
			}()

			service := &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: entryTask.Namespace,
					Labels:    entryTask.Spec.Selector.MatchLabels,
					OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(entryTask, kantetaskv1.GroupVersion.WithKind("EntryTask")),
					},
				},
				Spec: v1.ServiceSpec{
					Selector: entryTask.Spec.Selector.MatchLabels,
					Ports: []v1.ServicePort{{
						Protocol:   v1.ProtocolTCP,
						Port:       servicePort,
						TargetPort: intstr.FromInt32(pod.Spec.Containers[0].Ports[0].ContainerPort),
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

func (r *EntryTaskReconciler) deleteResources(ctx context.Context, entryTask *kantetaskv1.EntryTask) error {
	log := log.FromContext(ctx)

	deleteOptions := []client.DeleteAllOfOption{
		client.InNamespace(entryTask.Namespace),
		client.MatchingLabels(entryTask.Labels),
	}

	// delete all the pods
	pod := &v1.Pod{}
	if err := r.DeleteAllOf(ctx, pod, deleteOptions...); err != nil {
		log.Error(err, "Failed to delete pod")
		return err
	}
	log.Info("Deleted all the pods")

	service := &v1.Service{}
	if err := r.DeleteAllOf(ctx, service, deleteOptions...); err != nil {
		log.Error(err, "Failed to delete service")
		return err
	}
	log.Info("Deleted all the services")
	log.Info("Deleted all the resources")

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
