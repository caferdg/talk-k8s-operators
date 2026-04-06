package services

import (
	"context"
	"fmt"

	"github.com/caferdg/talk-k8s-operators/pkg/domain/entity"
	"github.com/caferdg/talk-k8s-operators/pkg/ports/secondary"
)

type GroupServiceImpl struct {
	client secondary.SCMClient
}

func NewGroupService(client secondary.SCMClient) *GroupServiceImpl {
	return &GroupServiceImpl{client: client}
}

func (s *GroupServiceImpl) ReconcileGroup(ctx context.Context, groupID int) (*entity.Group, error) {
	group, err := s.client.GetGroup(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group %d: %w", groupID, err)
	}
	return group, nil
}
