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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	platformv1alpha1 "github.com/caferdg/talk-k8s-operators/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var gitlabprojectlog = logf.Log.WithName("gitlabproject-resource")

// SetupGitlabProjectWebhookWithManager registers the webhook for GitlabProject in the manager.
func SetupGitlabProjectWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&platformv1alpha1.GitlabProject{}).
		WithValidator(&GitlabProjectCustomValidator{}).
		WithDefaulter(&GitlabProjectCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-platform-caferdg-io-v1alpha1-gitlabproject,mutating=true,failurePolicy=fail,sideEffects=None,groups=platform.caferdg.io,resources=gitlabprojects,verbs=create;update,versions=v1alpha1,name=mgitlabproject-v1alpha1.kb.io,admissionReviewVersions=v1

// GitlabProjectCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind GitlabProject when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type GitlabProjectCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &GitlabProjectCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind GitlabProject.
func (d *GitlabProjectCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	project, ok := obj.(*platformv1alpha1.GitlabProject)
	if !ok {
		return fmt.Errorf("expected a GitlabProject object but got %T", obj)
	}
	gitlabprojectlog.Info("Defaulting for GitlabProject", "name", project.GetName())

	if project.Spec.Properties.Visibility == "" {
		project.Spec.Properties.Visibility = "private"
	}

	return nil
}

// +kubebuilder:webhook:path=/validate-platform-caferdg-io-v1alpha1-gitlabproject,mutating=false,failurePolicy=fail,sideEffects=None,groups=platform.caferdg.io,resources=gitlabprojects,verbs=create;update,versions=v1alpha1,name=vgitlabproject-v1alpha1.kb.io,admissionReviewVersions=v1

// GitlabProjectCustomValidator struct is responsible for validating the GitlabProject resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type GitlabProjectCustomValidator struct{}

var _ webhook.CustomValidator = &GitlabProjectCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type GitlabProject.
func (v *GitlabProjectCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type GitlabProject.
func (v *GitlabProjectCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type GitlabProject.
func (v *GitlabProjectCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
