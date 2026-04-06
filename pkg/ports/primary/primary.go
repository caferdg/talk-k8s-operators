package primary

import (
	"context"

	"github.com/caferdg/talk-k8s-operators/pkg/domain/entity"
)

type GroupService interface {
	ReconcileGroup(ctx context.Context, groupID int) (*entity.Group, error)
}

type ProjectService interface {
	ReconcileProject(ctx context.Context, groupID int, name, visibility string) (*entity.ProjectResult, error)
	GetBranchCommit(ctx context.Context, projectID int, branch string) (*entity.Commit, error)
	GetLastPipeline(ctx context.Context, projectID int, branch string) (*entity.Pipeline, error)
}

type NotificationService interface {
	Notify(ctx context.Context, notification entity.Notification) error
	DetectEvents(
		subscribedEvents []entity.NotifierEvent,
		pipeline *entity.Pipeline,
		commit *entity.Commit,
		state *entity.ProjectNotificationState,
	) []entity.NotifierEvent
}
