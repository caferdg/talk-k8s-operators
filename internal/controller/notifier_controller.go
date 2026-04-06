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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	platformv1alpha1 "github.com/caferdg/talk-k8s-operators/api/v1alpha1"
)

// NotifierReconciler reconciles a Notifier object
type NotifierReconciler struct {
	client.Client
	APIReader client.Reader
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
}

// +kubebuilder:rbac:groups=platform.caferdg.io,resources=notifiers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.caferdg.io,resources=notifiers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.caferdg.io,resources=notifiers/finalizers,verbs=update
// +kubebuilder:rbac:groups=platform.caferdg.io,resources=gitlabprojects,verbs=get;list;watch
// +kubebuilder:rbac:groups=platform.caferdg.io,resources=gitlabprojects/status,verbs=get
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *NotifierReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var notifier platformv1alpha1.Notifier
	if err := r.APIReader.Get(ctx, req.NamespacedName, &notifier); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	patch := client.MergeFrom(notifier.DeepCopy())

	notifiers, err := r.resolveNotifiers(ctx, &notifier)
	if err != nil {
		log.Error(err, "failed to resolve notification providers")
		r.setCondition(ctx, &notifier, "False", "ProviderError", err.Error())
		if statusErr := r.Status().Patch(ctx, &notifier, patch); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{RequeueAfter: requeueSlow}, nil
	}

	for _, n := range notifiers {
		if err := n.HealthCheck(ctx); err != nil {
			log.Error(err, "provider health check failed")
			r.setCondition(ctx, &notifier, "False", "HealthCheckFailed", err.Error())
			if statusErr := r.Status().Patch(ctx, &notifier, patch); statusErr != nil {
				return ctrl.Result{}, statusErr
			}
			return ctrl.Result{RequeueAfter: requeueSlow}, nil
		}
	}

	projects, err := r.findLinkedProjects(ctx, &notifier)
	if err != nil {
		log.Error(err, "failed to list GitlabProjects")
		return ctrl.Result{RequeueAfter: requeueSlow}, nil
	}

	if len(projects) == 0 {
		log.Info("no GitlabProjects reference this notifier")
		r.setCondition(ctx, &notifier, "True", "Ready", "No projects linked")
		if err := r.Status().Patch(ctx, &notifier, patch); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: requeueDefault}, nil
	}

	if notifier.Status.ProjectStates == nil {
		notifier.Status.ProjectStates = make(map[string]platformv1alpha1.ProjectNotificationState)
	}

	for i := range projects {
		project := &projects[i]
		if err := r.processProject(ctx, &notifier, project, notifiers); err != nil {
			log.Error(err, "failed to process project", "project", project.Name)
		}
	}

	r.setCondition(ctx, &notifier, "True", "Ready", "Notifier is active")

	if err := r.Status().Patch(ctx, &notifier, patch); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("reconciled notifier", "projects", len(projects))
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NotifierReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Notifier{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(
			&platformv1alpha1.GitlabProject{},
			handler.EnqueueRequestsFromMapFunc(r.mapProjectToNotifier),
		).
		Named("notifier").
		Complete(r)
}

// mapProjectToNotifier maps a GitlabProject change to the Notifier it references.
func (r *NotifierReconciler) mapProjectToNotifier(ctx context.Context, obj client.Object) []reconcile.Request {
	project, ok := obj.(*platformv1alpha1.GitlabProject)
	if !ok {
		return nil
	}
	if project.Spec.NotifierRef == "" {
		return nil
	}
	return []reconcile.Request{
		{NamespacedName: client.ObjectKey{
			Name:      project.Spec.NotifierRef,
			Namespace: project.Namespace,
		}},
	}
}
