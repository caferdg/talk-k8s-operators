package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/caferdg/talk-k8s-operators/api/v1alpha1"
	"github.com/caferdg/talk-k8s-operators/pkg/adapters/secondary/gitlab"
	"github.com/caferdg/talk-k8s-operators/pkg/domain/entity"
	"github.com/caferdg/talk-k8s-operators/pkg/domain/services"
	"github.com/caferdg/talk-k8s-operators/pkg/ports/secondary"
)

func (r *GitlabProjectReconciler) resolveParentGroup(ctx context.Context, project *platformv1alpha1.GitlabProject) (*platformv1alpha1.GitlabGroup, ctrl.Result) {
	var parentGroup platformv1alpha1.GitlabGroup
	key := types.NamespacedName{
		Name:      project.Spec.ParentGroupRef,
		Namespace: project.Namespace,
	}
	if err := r.Get(ctx, key, &parentGroup); err != nil {
		r.setCondition(ctx, project, metav1.ConditionFalse, "ParentGroupNotFound", "GitlabGroup CR not found: "+err.Error())
		return nil, ctrl.Result{RequeueAfter: requeueFast}
	}
	return &parentGroup, ctrl.Result{}
}

func (r *GitlabProjectReconciler) resolveToken(ctx context.Context, project *platformv1alpha1.GitlabProject, parentGroup *platformv1alpha1.GitlabGroup) (string, ctrl.Result) {
	var secret corev1.Secret
	secretKey := types.NamespacedName{
		Name:      parentGroup.Spec.TokenSecretRef,
		Namespace: project.Namespace,
	}
	if err := r.Get(ctx, secretKey, &secret); err != nil {
		r.setCondition(ctx, project, metav1.ConditionFalse, "SecretNotFound", "Token secret not found: "+err.Error())
		return "", ctrl.Result{RequeueAfter: requeueFast}
	}

	token := string(secret.Data["token"])
	if token == "" {
		r.setCondition(ctx, project, metav1.ConditionFalse, "TokenEmpty", "Secret does not contain a 'token' key")
		return "", ctrl.Result{RequeueAfter: requeueFast}
	}

	return token, ctrl.Result{}
}

func (r *GitlabProjectReconciler) reconcileGitlab(ctx context.Context, project *platformv1alpha1.GitlabProject, token string, groupID int) (*entity.ProjectResult, ctrl.Result, error) {
	log := logf.FromContext(ctx)

	gitlabClient := gitlab.NewClient(token)
	notifier, err := r.resolveNotifier(ctx, project)
	if err != nil {
		log.Error(err, "failed to resolve notifier")
		r.Recorder.Event(project, corev1.EventTypeWarning, "NotifierError", err.Error())
		r.setCondition(ctx, project, metav1.ConditionFalse, "NotifierError", err.Error())
		return nil, ctrl.Result{RequeueAfter: requeueFast}, err
	}
	projectService := services.NewProjectService(gitlabClient, notifier)

	result, err := projectService.ReconcileProject(ctx, groupID, project.Spec.Properties.Name, project.Spec.Properties.Description, project.Spec.Properties.Visibility)
	if err != nil {
		log.Error(err, "failed to reconcile project")
		r.Recorder.Event(project, corev1.EventTypeWarning, "GitLabAPIError", err.Error())
		r.setCondition(ctx, project, metav1.ConditionFalse, "GitLabAPIError", err.Error())
		return nil, ctrl.Result{RequeueAfter: requeueSlow}, err
	}

	if !project.Status.Stamped {
		if err := projectService.StampProject(ctx, groupID, project.Spec.Properties.Name); err != nil {
			log.Error(err, "failed to stamp project")
		} else {
			project.Status.Stamped = true
		}
	}

	commit, err := projectService.GetBranchCommit(ctx, result.Project.ID, project.Spec.TrackedBranch)
	if err != nil {
		log.Error(err, "failed to get branch commit")
		r.Recorder.Event(project, corev1.EventTypeWarning, "BranchError", err.Error())
		r.setCondition(ctx, project, metav1.ConditionFalse, "BranchError", err.Error())
		return nil, ctrl.Result{RequeueAfter: requeueSlow}, err
	}
	result.Project.LastCommit = commit

	pipeline, err := projectService.GetLastPipeline(ctx, result.Project.ID, project.Spec.TrackedBranch)
	if err != nil {
		log.Error(err, "failed to get last pipeline")
		r.Recorder.Event(project, corev1.EventTypeWarning, "PipelineError", err.Error())
		r.setCondition(ctx, project, metav1.ConditionFalse, "PipelineError", err.Error())
		return nil, ctrl.Result{RequeueAfter: requeueSlow}, err
	}
	result.Project.LastPipeline = pipeline

	return result, ctrl.Result{}, nil
}

func (r *GitlabProjectReconciler) updateSuccessStatus(ctx context.Context, project *platformv1alpha1.GitlabProject, result *entity.ProjectResult) (ctrl.Result, error) {
	patch := client.MergeFrom(project.DeepCopy())

	project.Status.FullPath = result.Project.FullPath
	project.Status.Description = result.Project.Description
	project.Status.Visibility = result.Project.Visibility

	switch result.Action {
	case entity.ProjectCreated:
		r.Recorder.Event(project, corev1.EventTypeNormal, "ProjectCreated",
			fmt.Sprintf("Project '%s' created on GitLab", result.Project.Name))
	case entity.ProjectUpdated:
		r.Recorder.Event(project, corev1.EventTypeNormal, "ProjectUpdated",
			fmt.Sprintf("Project '%s' updated on GitLab", result.Project.Name))
	}

	if result.Project.LastCommit != nil {
		project.Status.LastCommit = platformv1alpha1.GitlabCommitStatus{
			SHA:       result.Project.LastCommit.SHA,
			Message:   result.Project.LastCommit.Message,
			Author:    result.Project.LastCommit.Author,
			CreatedAt: result.Project.LastCommit.CreatedAt,
		}
	}

	if result.Project.LastPipeline != nil {
		project.Status.LastPipeline = platformv1alpha1.GitlabPipelineStatus{
			ID:        result.Project.LastPipeline.ID,
			Status:    result.Project.LastPipeline.Status,
			WebURL:    result.Project.LastPipeline.WebURL,
			CreatedAt: result.Project.LastPipeline.CreatedAt,
		}
	}

	meta.SetStatusCondition(&project.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "ProjectReconciled",
		Message:            "Project is in sync",
		ObservedGeneration: project.Generation,
	})

	if err := r.Status().Patch(ctx, project, patch); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: requeueDefault}, nil
}

func (r *GitlabProjectReconciler) resolveNotifier(ctx context.Context, project *platformv1alpha1.GitlabProject) (secondary.Notifier, error) {
	if project.Spec.NotifierRef == "" {
		return nil, nil
	}

	var notifier platformv1alpha1.Notifier
	key := types.NamespacedName{Name: project.Spec.NotifierRef, Namespace: project.Namespace}
	if err := r.Get(ctx, key, &notifier); err != nil {
		return nil, fmt.Errorf("notifier %s not found: %w", project.Spec.NotifierRef, err)
	}

	adapters, err := resolveNotifierAdapters(ctx, r.Client, &notifier)
	if err != nil {
		return nil, err
	}
	if len(adapters) > 0 {
		return adapters[0], nil
	}
	return nil, nil
}

func (r *GitlabProjectReconciler) handleDeletion(ctx context.Context, project *platformv1alpha1.GitlabProject, token string, groupID int) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(project, projectFinalizer) {
		// Skip GitLab cleanup if dependencies (Secret/Group) are already gone
		if token != "" {
			r.finalizeProject(ctx, project, token, groupID)
		}
		patch := client.MergeFrom(project.DeepCopy())
		controllerutil.RemoveFinalizer(project, projectFinalizer)
		if err := r.Patch(ctx, project, patch); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *GitlabProjectReconciler) finalizeProject(ctx context.Context, project *platformv1alpha1.GitlabProject, token string, groupID int) {
	log := logf.FromContext(ctx)
	log.Info("finalizing project", "name", project.Spec.Properties.Name)

	gitlabClient := gitlab.NewClient(token)
	projectService := services.NewProjectService(gitlabClient, nil)
	if err := projectService.FinalizeProject(ctx, groupID, project.Spec.Properties.Name); err != nil {
		log.Error(err, "failed to unstamp project, proceeding with deletion")
	}
}

//nolint:unparam // status is always ConditionFalse for now but kept for symmetry
func (r *GitlabProjectReconciler) setCondition(ctx context.Context, project *platformv1alpha1.GitlabProject, status metav1.ConditionStatus, reason, message string) {
	patch := client.MergeFrom(project.DeepCopy())
	meta.SetStatusCondition(&project.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: project.Generation,
	})
	_ = r.Status().Patch(ctx, project, patch)
}
