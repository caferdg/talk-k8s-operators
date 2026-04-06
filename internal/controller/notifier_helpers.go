package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	platformv1alpha1 "github.com/caferdg/talk-k8s-operators/api/v1alpha1"
	"github.com/caferdg/talk-k8s-operators/pkg/domain/entity"
	"github.com/caferdg/talk-k8s-operators/pkg/domain/services"
	"github.com/caferdg/talk-k8s-operators/pkg/ports/secondary"
)

func (r *NotifierReconciler) findLinkedProjects(ctx context.Context, notifier *platformv1alpha1.Notifier) ([]platformv1alpha1.GitlabProject, error) {
	var projectList platformv1alpha1.GitlabProjectList
	if err := r.List(ctx, &projectList, client.InNamespace(notifier.Namespace)); err != nil {
		return nil, err
	}

	var linked []platformv1alpha1.GitlabProject
	for _, p := range projectList.Items {
		if p.Spec.NotifierRef == notifier.Name {
			linked = append(linked, p)
		}
	}
	return linked, nil
}

func (r *NotifierReconciler) resolveNotifiers(ctx context.Context, notifier *platformv1alpha1.Notifier) ([]secondary.Notifier, error) {
	return resolveNotifierAdapters(ctx, r.Client, notifier)
}

func (r *NotifierReconciler) processProject(ctx context.Context, notifier *platformv1alpha1.Notifier, project *platformv1alpha1.GitlabProject, notifiers []secondary.Notifier) error {
	// Skip projects that haven't been reconciled yet (no status data)
	if project.Status.LastPipeline.ID == 0 && project.Status.LastCommit.SHA == "" {
		return nil
	}

	crState, exists := notifier.Status.ProjectStates[project.Name]
	if !exists {
		// For existing projects (commit predates the CR), seed state without notifying.
		// For new projects (commit arrived after CR creation), start with empty state so the first commit triggers a notification.
		commitTime, err := time.Parse(time.RFC3339, project.Status.LastCommit.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to parse commit timestamp %q for project %s: %w", project.Status.LastCommit.CreatedAt, project.Name, err)
		}
		if commitTime.Before(project.CreationTimestamp.Time) {
			notifier.Status.ProjectStates[project.Name] = platformv1alpha1.ProjectNotificationState{
				LastNotifiedPipelineID:     project.Status.LastPipeline.ID,
				LastNotifiedPipelineStatus: project.Status.LastPipeline.Status,
				LastNotifiedCommitSHA:      project.Status.LastCommit.SHA,
			}
			return nil
		}
		crState = platformv1alpha1.ProjectNotificationState{}
	}

	state := toDomainState(&crState)

	subscribedEvents := toDomainEvents(notifier.Spec.Events)
	pipeline := toDomainPipeline(&project.Status.LastPipeline)
	commit := toDomainCommit(&project.Status.LastCommit)

	// Use any notifier to detect events (detection logic is the same regardless of provider)
	notificationService := services.NewNotificationService(notifiers[0])
	events := notificationService.DetectEvents(subscribedEvents, pipeline, commit, &state)

	if len(events) == 0 {
		return nil
	}

	for _, event := range events {
		notification := buildNotification(event, project)

		for _, n := range notifiers {
			svc := services.NewNotificationService(n)
			if err := svc.Notify(ctx, notification); err != nil {
				r.Recorder.Event(notifier, corev1.EventTypeWarning, "NotificationError", err.Error())
				return err
			}
		}

		r.Recorder.Event(notifier, corev1.EventTypeNormal, "NotificationSent",
			fmt.Sprintf("Notification sent for %s on '%s'", event, project.Name))
	}

	notifier.Status.ProjectStates[project.Name] = toCRDState(&state)
	return nil
}

func buildNotification(event entity.NotifierEvent, project *platformv1alpha1.GitlabProject) entity.Notification {
	var title, message string

	projectURL := "https://gitlab.com/" + project.Status.FullPath

	switch event {
	case entity.EventPipelineRunning:
		title = fmt.Sprintf("🔄 Pipeline running — [%s](%s)", project.Status.FullPath, projectURL)
		message = fmt.Sprintf("Pipeline [#%d](<%s>) is running", project.Status.LastPipeline.ID, project.Status.LastPipeline.WebURL)
	case entity.EventPipelineSuccess:
		title = fmt.Sprintf("✅ Pipeline succeeded — [%s](%s)", project.Status.FullPath, projectURL)
		message = fmt.Sprintf("Pipeline [#%d](<%s>) succeeded", project.Status.LastPipeline.ID, project.Status.LastPipeline.WebURL)
	case entity.EventPipelineFailed:
		title = fmt.Sprintf("❌ Pipeline failed — [%s](%s)", project.Status.FullPath, projectURL)
		message = fmt.Sprintf("Pipeline [#%d](<%s>) failed", project.Status.LastPipeline.ID, project.Status.LastPipeline.WebURL)
	case entity.EventPipelineCanceled:
		title = fmt.Sprintf("⛔ Pipeline canceled — [%s](%s)", project.Status.FullPath, projectURL)
		message = fmt.Sprintf("Pipeline [#%d](<%s>) was canceled", project.Status.LastPipeline.ID, project.Status.LastPipeline.WebURL)
	case entity.EventNewCommit:
		title = fmt.Sprintf("📦 New commit — [%s](%s)", project.Status.FullPath, projectURL)
		commitURL := fmt.Sprintf("%s/-/commit/%s", projectURL, project.Status.LastCommit.SHA)
		message = fmt.Sprintf("**%s** — by *%s*\n[`%s`](<%s>)", project.Status.LastCommit.Message, project.Status.LastCommit.Author, project.Status.LastCommit.SHA[:8], commitURL)
	}

	return entity.Notification{Title: title, Message: message}
}

// Mapping helpers between CRD types and domain types

func toDomainEvents(crEvents []platformv1alpha1.NotifierEvent) []entity.NotifierEvent {
	events := make([]entity.NotifierEvent, len(crEvents))
	for i, e := range crEvents {
		events[i] = entity.NotifierEvent(e)
	}
	return events
}

func toDomainPipeline(p *platformv1alpha1.GitlabPipelineStatus) *entity.Pipeline {
	if p.ID == 0 {
		return nil
	}
	return &entity.Pipeline{
		ID:        p.ID,
		Status:    p.Status,
		WebURL:    p.WebURL,
		CreatedAt: p.CreatedAt,
	}
}

func toDomainCommit(c *platformv1alpha1.GitlabCommitStatus) *entity.Commit {
	if c.SHA == "" {
		return nil
	}
	return &entity.Commit{
		SHA:       c.SHA,
		Message:   c.Message,
		Author:    c.Author,
		CreatedAt: c.CreatedAt,
	}
}

func toDomainState(s *platformv1alpha1.ProjectNotificationState) entity.ProjectNotificationState {
	return entity.ProjectNotificationState{
		LastNotifiedPipelineID:     s.LastNotifiedPipelineID,
		LastNotifiedPipelineStatus: s.LastNotifiedPipelineStatus,
		LastNotifiedCommitSHA:      s.LastNotifiedCommitSHA,
	}
}

func toCRDState(s *entity.ProjectNotificationState) platformv1alpha1.ProjectNotificationState {
	return platformv1alpha1.ProjectNotificationState{
		LastNotifiedPipelineID:     s.LastNotifiedPipelineID,
		LastNotifiedPipelineStatus: s.LastNotifiedPipelineStatus,
		LastNotifiedCommitSHA:      s.LastNotifiedCommitSHA,
	}
}

func (r *NotifierReconciler) setCondition(_ context.Context, notifier *platformv1alpha1.Notifier, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&notifier.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: notifier.Generation,
	})
}
