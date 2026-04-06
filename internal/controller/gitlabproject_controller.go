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

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	platformv1alpha1 "github.com/caferdg/talk-k8s-operators/api/v1alpha1"
)

const projectFinalizer = "platform.caferdg.io/project-cleanup"

// GitlabProjectReconciler reconciles a GitlabProject object
type GitlabProjectReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=platform.caferdg.io,resources=gitlabprojects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.caferdg.io,resources=gitlabprojects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.caferdg.io,resources=gitlabprojects/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *GitlabProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Fetch the GitlabProject CR
	var project platformv1alpha1.GitlabProject
	if err := r.Get(ctx, req.NamespacedName, &project); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Handle deletion, unstamp project (= remove Gitlab tag)
	if !project.DeletionTimestamp.IsZero() {
		var token string
		var groupID int
		if parentGroup, _ := r.resolveParentGroup(ctx, &project); parentGroup != nil {
			token, _ = r.resolveToken(ctx, &project, parentGroup)
			groupID = parentGroup.Spec.GroupID
		}
		return r.handleDeletion(ctx, &project, token, groupID)
	}

	// 3. Resolve dependencies
	parentGroup, res := r.resolveParentGroup(ctx, &project)
	if parentGroup == nil {
		return res, nil
	}
	token, res := r.resolveToken(ctx, &project, parentGroup)
	if token == "" {
		return res, nil
	}

	// 4. Ensure finalizer is registered
	if !controllerutil.ContainsFinalizer(&project, projectFinalizer) {
		patch := client.MergeFrom(project.DeepCopy())
		controllerutil.AddFinalizer(&project, projectFinalizer)
		if err := r.Patch(ctx, &project, patch); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 5. Reconcile the project on GitLab
	result, res, err := r.reconcileGitlab(ctx, &project, token, parentGroup.Spec.GroupID)
	if err != nil {
		return res, nil
	}

	log.Info("reconciled project", "name", result.Project.Name, "url", result.Project.WebURL)
	return r.updateSuccessStatus(ctx, &project, result)
}

// SetupWithManager sets up the controller with the Manager.
func (r *GitlabProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.GitlabProject{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.mapSecretToProjects),
		).
		Named("gitlabproject").
		Complete(r)
}

// mapSecretToProjects enqueues GitlabProjects whose parent group references the changed Secret.
func (r *GitlabProjectReconciler) mapSecretToProjects(ctx context.Context, obj client.Object) []reconcile.Request {
	var groupList platformv1alpha1.GitlabGroupList
	if err := r.List(ctx, &groupList, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	var requests []reconcile.Request
	for _, group := range groupList.Items {
		if group.Spec.TokenSecretRef != obj.GetName() {
			continue
		}
		var projectList platformv1alpha1.GitlabProjectList
		if err := r.List(ctx, &projectList, client.InNamespace(obj.GetNamespace())); err != nil {
			return nil
		}
		for _, project := range projectList.Items {
			if project.Spec.ParentGroupRef == group.Name {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      project.Name,
						Namespace: project.Namespace,
					},
				})
			}
		}
	}
	return requests
}
