package repository

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/infra/git"
)

func (s *Service) SyncFromProvider(ctx context.Context, repoID int64, accessToken string) (*gitprovider.Repository, error) {
	repo, err := s.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}

	client, err := git.NewProvider(repo.ProviderType, repo.ProviderBaseURL, accessToken)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create git provider client for sync", "repo_id", repoID, "provider_type", repo.ProviderType, "error", err)
		return nil, err
	}

	project, err := client.GetProject(ctx, repo.ExternalID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch project from git provider", "repo_id", repoID, "external_id", repo.ExternalID, "error", err)
		return nil, err
	}

	updates := map[string]interface{}{
		"name":           project.Name,
		"slug":           project.Slug,
		"default_branch": project.DefaultBranch,
	}
	if project.HttpCloneURL != "" {
		updates["http_clone_url"] = project.HttpCloneURL
	}
	if project.SSHCloneURL != "" {
		updates["ssh_clone_url"] = project.SSHCloneURL
	}

	slog.InfoContext(ctx, "repository synced from provider", "repo_id", repoID, "slug", project.Slug)

	return s.Update(ctx, repoID, updates)
}

func (s *Service) ListBranches(ctx context.Context, repoID int64, accessToken string) ([]string, error) {
	repo, err := s.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}

	client, err := git.NewProvider(repo.ProviderType, repo.ProviderBaseURL, accessToken)
	if err != nil {
		return nil, err
	}

	branches, err := client.ListBranches(ctx, repo.ExternalID)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, b := range branches {
		names = append(names, b.Name)
	}
	return names, nil
}

func (s *Service) GetNextTicketNumber(ctx context.Context, repoID int64) (int, error) {
	maxNumber, err := s.repo.GetMaxTicketNumber(ctx, repoID)
	if err != nil {
		return 0, err
	}
	return maxNumber + 1, nil
}
