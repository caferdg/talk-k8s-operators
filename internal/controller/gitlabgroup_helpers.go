package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/caferdg/talk-k8s-operators/api/v1alpha1"
	"github.com/caferdg/talk-k8s-operators/pkg/adapters/secondary/gitlab"
	"github.com/caferdg/talk-k8s-operators/pkg/domain/entity"
	"github.com/caferdg/talk-k8s-operators/pkg/domain/services"
)

func (r *GitlabGroupReconciler) resolveToken(ctx context.Context, group *platformv1alpha1.GitlabGroup) (string, ctrl.Result) {
	var secret corev1.Secret
	secretKey := types.NamespacedName{
		Name:      group.Spec.TokenSecretRef,
		Namespace: group.Namespace,
	}
	if err := r.Get(ctx, secretKey, &secret); err != nil {
		r.setCondition(ctx, group, metav1.ConditionFalse, "SecretNotFound", "Token secret not found: "+err.Error())
		return "", ctrl.Result{RequeueAfter: requeueFast}
	}

	token := string(secret.Data["token"])
	if token == "" {
		r.setCondition(ctx, group, metav1.ConditionFalse, "TokenEmpty", "Secret does not contain a 'token' key")
		return "", ctrl.Result{RequeueAfter: requeueFast}
	}

	return token, ctrl.Result{}
}

func (r *GitlabGroupReconciler) reconcileGitlab(ctx context.Context, group *platformv1alpha1.GitlabGroup, token string) (*entity.Group, ctrl.Result, error) {
	log := logf.FromContext(ctx)

	gitlabClient := gitlab.NewClient(token)
	groupService := services.NewGroupService(gitlabClient)

	result, err := groupService.ReconcileGroup(ctx, group.Spec.GroupID)
	if err != nil {
		log.Error(err, "failed to reconcile group")
		r.setCondition(ctx, group, metav1.ConditionFalse, "GitLabAPIError", err.Error())
		return nil, ctrl.Result{RequeueAfter: requeueSlow}, err
	}

	return result, ctrl.Result{}, nil
}

func (r *GitlabGroupReconciler) updateSuccessStatus(ctx context.Context, group *platformv1alpha1.GitlabGroup, result *entity.Group) (ctrl.Result, error) {
	patch := client.MergeFrom(group.DeepCopy())

	group.Status.FullPath = result.FullPath
	group.Status.WebURL = result.WebURL
	group.Status.Description = result.Description
	group.Status.Visibility = result.Visibility
	group.Status.Projects = result.Projects
	group.Status.Subgroups = result.Subgroups
	group.Status.MemberCount = result.MemberCount
	meta.SetStatusCondition(&group.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "GroupFound",
		Message:            "Group is accessible",
		ObservedGeneration: group.Generation,
	})

	if err := r.Status().Patch(ctx, group, patch); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: requeueDefault}, nil
}

func (r *GitlabGroupReconciler) setCondition(ctx context.Context, group *platformv1alpha1.GitlabGroup, status metav1.ConditionStatus, reason, message string) {
	patch := client.MergeFrom(group.DeepCopy())
	meta.SetStatusCondition(&group.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: group.Generation,
	})
	_ = r.Status().Patch(ctx, group, patch)
}
