/*
Copyright 2023 kenny.

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
	"github.com/mk100120/app-controller/internal/controller/utils"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	v13 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	configurationv1 "github.com/mk100120/app-controller/api/v1"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=configuration.github.com,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=configuration.github.com,resources=applications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=configuration.github.com,resources=applications/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Application object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	application := &configurationv1.Application{}
	//从缓存中获取app
	err := r.Get(ctx, req.NamespacedName, application)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//1. deployment handler
	deployment := utils.NewDeployment(application)
	err = controllerutil.SetControllerReference(application, deployment, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}
	d := &v1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, d); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, deployment); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		//Bug: 这里会反复触发更新
		//原因：在148行SetupWithManager方法中，监听了Deployment，所以只要更新Deployment就会触发
		//     此处更新和controllerManager更新Deployment都会触发更新事件，导致循环触发
		//修复方法：
		//方式1. 注释掉在148行SetupWithManager方法中对Deployment，Ingress，Service等的监听，该处的处理只是为了
		//      手动删除Deployment等后能够自动重建，但正常不会出现这种情况，是否需要根据情况而定
		//方式2. 加上判断条件，仅在app.Spec.Replicas != deployment.Spec.Replicas &&
		//      app.Spec.Image != deployment.Spec.Template.Spec.Containers[0].Image时才更新deployment
		if application.Spec.Replicas != *d.Spec.Replicas || application.Spec.Image != d.Spec.Template.Spec.Containers[0].Image {
			if err := r.Update(ctx, deployment); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	//2 service handler
	service := utils.NewService(application)
	if err := controllerutil.SetControllerReference(application, service, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	s := &v12.Service{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: application.Namespace, Name: application.Name}, s); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, service); err != nil {
				return ctrl.Result{}, err
			}
		}
		if !errors.IsNotFound(err) && application.Spec.EnableService {
			return ctrl.Result{}, err
		}
	} else {
		if application.Spec.EnableService {
			logger.Info("service status is ok")
		} else {
			if err := r.Delete(ctx, s); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	//3 ingress Handler

	ingress := utils.NewIngress(application)
	if err := controllerutil.SetControllerReference(application, ingress, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	i := &v13.Ingress{}
	if err := r.Get(ctx, types.NamespacedName{Name: application.Name, Namespace: application.Namespace}, i); err != nil {
		if errors.IsNotFound(err) && application.Spec.EnableIngress {
			if err := r.Create(ctx, ingress); err != nil {
				return ctrl.Result{}, err
			}
		}
		if !errors.IsNotFound(err) && application.Spec.EnableIngress {
			return ctrl.Result{}, err
		}
	} else {
		if application.Spec.EnableIngress {
			logger.Info("skip Update")
		} else {
			if err := r.Delete(ctx, i); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configurationv1.Application{}).
		Owns(&v1.Deployment{}).
		Owns(&v13.Ingress{}).
		Owns(&v12.Service{}).
		Complete(r)
}
