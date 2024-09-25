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

package v1

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type EntryTaskValidator struct {
	client.Client
}

// log is for logging in this package.
var log = logf.Log.WithName("entrytask-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *EntryTask) SetupWebhookWithManager(mgr ctrl.Manager) error {
	// add validator
	validator := &EntryTaskValidator{
		mgr.GetClient(),
	}

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(validator).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.

// +kubebuilder:webhook:path=/validate-kantetask-codereliant-io-v1-entrytask,mutating=false,failurePolicy=fail,sideEffects=None,groups=kantetask.codereliant.io,resources=entrytasks,verbs=create;update,versions=v1,name=ventrytask.kb.io,admissionReviewVersions=v1

//var _ webhook.Validator = &EntryTask{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *EntryTaskValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	entrtask, ok := obj.(*EntryTask)

	if !ok {
		return nil, fmt.Errorf("unexpected object type, expected EntryTask")
	}

	// check entrytask service port range
	servicePort := entrtask.Spec.ServicePort
	if servicePort != 0 {
		//  8000-9999
		if servicePort < 8000 || servicePort > 9999 {
			return nil, fmt.Errorf("servicePort must be between 8000 and 9999")
		}
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *EntryTaskValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	entrtask, ok := newObj.(*EntryTask)

	if !ok {
		return nil, fmt.Errorf("unexpected object type, expected EntryTask")
	}

	// check entrytask service port range
	servicePort := entrtask.Spec.ServicePort
	if servicePort != 0 {
		//  8000-9999
		if servicePort < 8000 || servicePort > 9999 {
			return nil, fmt.Errorf("servicePort must be between 8000 and 9999")
		}
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *EntryTaskValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
