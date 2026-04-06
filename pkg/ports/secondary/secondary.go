package secondary

import (
	"context"

	"github.com/caferdg/talk-k8s-operators/pkg/domain/entity"
)

type SCMClient interface {
	GetGroup(ctx context.Context, groupID int) (*entity.Group, error)
	GetProject(ctx context.Context, groupID int, projectName string) (*entity.Project, error)
	CreateProject(ctx context.Context, groupID int, name, description, visibility string) (*entity.Project, error)
	UpdateProject(ctx context.Context, projectID int, description, visibility string) (*entity.Project, error)
	GetBranchCommit(ctx context.Context, projectID int, branch string) (*entity.Commit, error)
	GetLastPipeline(ctx context.Context, projectID int, branch string) (*entity.Pipeline, error)
	Stamp(ctx context.Context, projectID int) error
	Unstamp(ctx context.Context, projectID int) error
}

type Notifier interface {
	Send(ctx context.Context, notification entity.Notification) error
	HealthCheck(ctx context.Context) error
}
