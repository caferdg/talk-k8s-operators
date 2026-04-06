package services

import (
	"context"
	"fmt"

	"github.com/caferdg/talk-k8s-operators/pkg/domain/entity"
	"github.com/caferdg/talk-k8s-operators/pkg/ports/secondary"
)

type ProjectServiceImpl struct {
	scm      secondary.SCMClient
	notifier secondary.Notifier
}

func NewProjectService(scm secondary.SCMClient, notifier secondary.Notifier) *ProjectServiceImpl {
	return &ProjectServiceImpl{scm: scm, notifier: notifier}
}

func (s *ProjectServiceImpl) ReconcileProject(
	ctx context.Context, groupID int, name, description, visibility string,
) (*entity.ProjectResult, error) {
	// 1. Validate parent group exists
	_, err := s.scm.GetGroup(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("parent group %d not found: %w", groupID, err)
	}

	// 2. Check if project already exists
	project, err := s.scm.GetProject(ctx, groupID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to check project %s: %w", name, err)
	}

	// 3. Project doesn't exist — create it and notify
	if project == nil {
		project, err = s.scm.CreateProject(ctx, groupID, name, description, visibility)
		if err != nil {
			return nil, fmt.Errorf("failed to create project %s: %w", name, err)
		}

		if s.notifier != nil {
			_ = s.notifier.Send(ctx, entity.Notification{
				Title:   "🆕 Project created",
				Message: fmt.Sprintf("**%s** has been created\n<%s>", project.Name, project.WebURL),
			})
		}

		return &entity.ProjectResult{Project: project, Action: entity.ProjectCreated}, nil
	}

	// 4. Project exists — check for drift and update if needed
	if project.Visibility != visibility || project.Description != description {
		project, err = s.scm.UpdateProject(ctx, project.ID, description, visibility)
		if err != nil {
			return nil, fmt.Errorf("failed to update project %s: %w", name, err)
		}

		if s.notifier != nil {
			_ = s.notifier.Send(ctx, entity.Notification{
				Title:   "🔧 Project updated",
				Message: fmt.Sprintf("**%s** has been updated\n<%s>", project.Name, project.WebURL),
			})
		}

		return &entity.ProjectResult{Project: project, Action: entity.ProjectUpdated}, nil
	}

	return &entity.ProjectResult{Project: project, Action: entity.ProjectUnchanged}, nil
}

func (s *ProjectServiceImpl) StampProject(ctx context.Context, groupID int, name string) error {
	project, err := s.scm.GetProject(ctx, groupID, name)
	if err != nil || project == nil {
		return err
	}
	return s.scm.Stamp(ctx, project.ID)
}

func (s *ProjectServiceImpl) FinalizeProject(ctx context.Context, groupID int, name string) error {
	project, err := s.scm.GetProject(ctx, groupID, name)
	if err != nil || project == nil {
		return err
	}
	return s.scm.Unstamp(ctx, project.ID)
}

func (s *ProjectServiceImpl) GetBranchCommit(
	ctx context.Context, projectID int, branch string,
) (*entity.Commit, error) {
	commit, err := s.scm.GetBranchCommit(ctx, projectID, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch commit for project %d: %w", projectID, err)
	}
	return commit, nil
}

func (s *ProjectServiceImpl) GetLastPipeline(
	ctx context.Context, projectID int, branch string,
) (*entity.Pipeline, error) {
	pipeline, err := s.scm.GetLastPipeline(ctx, projectID, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to get last pipeline for project %d: %w", projectID, err)
	}
	return pipeline, nil
}
