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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kantetaskv1 "codereliant.io/entrytask/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	// 使用选择器获取相关的 Pods
	selector, err := metav1.LabelSelectorAsSelector(entryTask.Spec.Selector)
	if err != nil {
		return ctrl.Result{}, err
	}

	// 创建一个 Pod 列表
	podList := &v1.PodList{}
	// 获取指定命名空间下的 Pods
	if err := r.List(ctx, podList, &client.ListOptions{
		Namespace:     req.Namespace, // 指定命名空间
		LabelSelector: selector,      // 使用选择器
	}); err != nil {
		return ctrl.Result{}, err
	}

	desiredPodCount := entryTask.Spec.DesiredReplicas
	currentPodCount := len(podList.Items)

	for currentPodCount < desiredPodCount {
		log.Info("currentPodCount < desiredPodCount, create new pod", "desiredPodCount", desiredPodCount, "currentPodCount", currentPodCount)
		// 创建新 Pod 的代码示例
		newPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-pod-%d", entryTask.Name, podCount), // 为新 Pod 生成唯一名称
				Namespace: req.Namespace,
				Labels:    entryTask.Spec.Selector.MatchLabels, // 设置 Pod 的标签
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  entryTask.Spec.Template.Spec.Containers[0].Name,  // 替换为实际的容器名称
						Image: entryTask.Spec.Template.Spec.Containers[0].Image, // 替换为实际的镜像
					},
				},
			},
		}

		// 创建 Pod
		if err := r.Create(ctx, newPod); err != nil {
			log.Error(err, "Failed to create Pod")
			return ctrl.Result{}, err
		}
		log.Info("Created new Pod", "Pod.Name", newPod.Name)
	}

	return ctrl.Result{}, nil
}

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
		Complete(r)
}
