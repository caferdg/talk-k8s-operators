/*
Copyright 2026.

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

package v1alpha1

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	platformv1alpha1 "github.com/caferdg/talk-k8s-operators/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var notifierlog = logf.Log.WithName("notifier-resource")

var validEvents = map[platformv1alpha1.NotifierEvent]bool{
	platformv1alpha1.EventPipelineRunning:  true,
	platformv1alpha1.EventPipelineSuccess:  true,
	platformv1alpha1.EventPipelineFailed:   true,
	platformv1alpha1.EventPipelineCanceled: true,
	platformv1alpha1.EventNewCommit:        true,
}

// SetupNotifierWebhookWithManager registers the webhook for Notifier in the manager.
func SetupNotifierWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&platformv1alpha1.Notifier{}).
		WithValidator(&NotifierCustomValidator{}).
		WithDefaulter(&NotifierCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-platform-caferdg-io-v1alpha1-notifier,mutating=true,failurePolicy=fail,sideEffects=None,groups=platform.caferdg.io,resources=notifiers,verbs=create;update,versions=v1alpha1,name=mnotifier-v1alpha1.kb.io,admissionReviewVersions=v1

// NotifierCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Notifier when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type NotifierCustomDefaulter struct{}

var _ webhook.CustomDefaulter = &NotifierCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Notifier.
func (d *NotifierCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	notifier, ok := obj.(*platformv1alpha1.Notifier)
	if !ok {
		return fmt.Errorf("expected a Notifier object but got %T", obj)
	}
	notifierlog.Info("Defaulting for Notifier", "name", notifier.GetName())

	return nil
}

// +kubebuilder:webhook:path=/validate-platform-caferdg-io-v1alpha1-notifier,mutating=false,failurePolicy=fail,sideEffects=None,groups=platform.caferdg.io,resources=notifiers,verbs=create;update,versions=v1alpha1,name=vnotifier-v1alpha1.kb.io,admissionReviewVersions=v1

// NotifierCustomValidator struct is responsible for validating the Notifier resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type NotifierCustomValidator struct{}

var _ webhook.CustomValidator = &NotifierCustomValidator{}

func validateNotifier(notifier *platformv1alpha1.Notifier) (admission.Warnings, error) {
	var errs []string

	// At least one provider must be configured
	if notifier.Spec.Discord == nil && notifier.Spec.Slack == nil && notifier.Spec.Telegram == nil {
		errs = append(errs, "at least one provider must be configured (discord, slack, or telegram)")
	}

	// Events must not be empty
	if len(notifier.Spec.Events) == 0 {
		errs = append(errs, "at least one event must be specified")
	}

	// All events must be valid
	for _, event := range notifier.Spec.Events {
		if !validEvents[event] {
			errs = append(errs, fmt.Sprintf("invalid event %q, must be one of: pipeline_running, pipeline_success, pipeline_failed, pipeline_canceled, new_commit", event))
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("validation failed: %s", strings.Join(errs, "; "))
	}

	return nil, nil
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Notifier.
func (v *NotifierCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	notifier, ok := obj.(*platformv1alpha1.Notifier)
	if !ok {
		return nil, fmt.Errorf("expected a Notifier object but got %T", obj)
	}
	notifierlog.Info("Validation for Notifier upon creation", "name", notifier.GetName())

	return validateNotifier(notifier)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Notifier.
func (v *NotifierCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	notifier, ok := newObj.(*platformv1alpha1.Notifier)
	if !ok {
		return nil, fmt.Errorf("expected a Notifier object for the newObj but got %T", newObj)
	}
	notifierlog.Info("Validation for Notifier upon update", "name", notifier.GetName())

	return validateNotifier(notifier)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Notifier.
func (v *NotifierCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
