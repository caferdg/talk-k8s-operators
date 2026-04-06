package services

import (
	"context"

	"github.com/caferdg/talk-k8s-operators/pkg/domain/entity"
	"github.com/caferdg/talk-k8s-operators/pkg/ports/secondary"
)

type NotificationServiceImpl struct {
	notifier secondary.Notifier
}

func NewNotificationService(notifier secondary.Notifier) *NotificationServiceImpl {
	return &NotificationServiceImpl{notifier: notifier}
}

func (s *NotificationServiceImpl) Notify(ctx context.Context, notification entity.Notification) error {
	return s.notifier.Send(ctx, notification)
}

// DetectEvents compares a project's current state against the last notified state
// and returns the list of events that should trigger a notification.
// It also updates the state in-place with the new values.
func (s *NotificationServiceImpl) DetectEvents(
	subscribedEvents []entity.NotifierEvent,
	pipeline *entity.Pipeline,
	commit *entity.Commit,
	state *entity.ProjectNotificationState,
) []entity.NotifierEvent {
	subscribed := make(map[entity.NotifierEvent]bool)
	for _, e := range subscribedEvents {
		subscribed[e] = true
	}

	var detected []entity.NotifierEvent

	if commit != nil && commit.SHA != "" && commit.SHA != state.LastNotifiedCommitSHA {
		if subscribed[entity.EventNewCommit] {
			detected = append(detected, entity.EventNewCommit)
			state.LastNotifiedCommitSHA = commit.SHA
		}
	}

	if pipeline != nil && pipeline.ID != 0 &&
		(pipeline.ID != state.LastNotifiedPipelineID || pipeline.Status != state.LastNotifiedPipelineStatus) {
		var event entity.NotifierEvent
		switch pipeline.Status {
		case "running":
			event = entity.EventPipelineRunning
		case "success":
			event = entity.EventPipelineSuccess
		case "failed":
			event = entity.EventPipelineFailed
		case "canceled":
			event = entity.EventPipelineCanceled
		}
		if event != "" && subscribed[event] {
			detected = append(detected, event)
			state.LastNotifiedPipelineID = pipeline.ID
			state.LastNotifiedPipelineStatus = pipeline.Status
		}
	}

	return detected
}
