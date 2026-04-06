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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	platformv1alpha1 "github.com/caferdg/talk-k8s-operators/api/v1alpha1"
)

// GitlabGroupReconciler reconciles a GitlabGroup object
type GitlabGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=platform.caferdg.io,resources=gitlabgroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.caferdg.io,resources=gitlabgroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.caferdg.io,resources=gitlabgroups/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *GitlabGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var group platformv1alpha1.GitlabGroup
	if err := r.Get(ctx, req.NamespacedName, &group); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	token, res := r.resolveToken(ctx, &group)
	if token == "" {
		return res, nil
	}

	result, res, err := r.reconcileGitlab(ctx, &group, token)
	if err != nil {
		return res, nil
	}

	log.Info("reconciled group", "fullPath", result.FullPath, "url", result.WebURL)
	return r.updateSuccessStatus(ctx, &group, result)
}

// SetupWithManager sets up the controller with the Manager.
func (r *GitlabGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.GitlabGroup{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.mapSecretToGroups),
		).
		Named("gitlabgroup").
		Complete(r)
}

// mapSecretToGroups enqueues all GitlabGroups that reference the changed Secret as their token source.
func (r *GitlabGroupReconciler) mapSecretToGroups(ctx context.Context, obj client.Object) []reconcile.Request {
	var groupList platformv1alpha1.GitlabGroupList
	if err := r.List(ctx, &groupList, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	var requests []reconcile.Request
	for _, group := range groupList.Items {
		if group.Spec.TokenSecretRef == obj.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      group.Name,
					Namespace: group.Namespace,
				},
			})
		}
	}
	return requests
}
