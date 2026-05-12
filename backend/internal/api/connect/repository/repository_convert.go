package repositoryconnect

import (
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
	repositoryv1 "github.com/anthropics/agentsmesh/proto/gen/go/repository/v1"
)

// toProtoRepository converts the GORM-backed domain model into the
// protobuf wire shape. Fields kept in lockstep with the .proto definition
// — every reviewer's first check is the field-count + name diff (watch
// list §6 / §8).
//
// Timestamp policy (conventions §6): time.Time → RFC 3339 string.
// optional fields are encoded by leaving the *string nil when source is
// the zero value or pointer is nil.
func toProtoRepository(r *gitprovider.Repository) *repositoryv1.Repository {
	if r == nil {
		return nil
	}
	out := &repositoryv1.Repository{
		Id:              r.ID,
		OrganizationId:  r.OrganizationID,
		ProviderType:    r.ProviderType,
		ProviderBaseUrl: r.ProviderBaseURL,
		HttpCloneUrl:    r.HttpCloneURL,
		SshCloneUrl:     r.SshCloneURL,
		ExternalId:      r.ExternalID,
		Name:            r.Name,
		Slug:            r.Slug,
		DefaultBranch:   r.DefaultBranch,
		Visibility:      r.Visibility,
		IsActive:        r.IsActive,
		CreatedAt:       r.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       r.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if r.TicketPrefix != nil {
		out.TicketPrefix = r.TicketPrefix
	}
	if r.ImportedByUserID != nil {
		out.ImportedByUserId = r.ImportedByUserID
	}
	if r.PreparationScript != nil {
		out.PreparationScript = r.PreparationScript
	}
	if r.PreparationTimeout != nil {
		t := int32(*r.PreparationTimeout)
		out.PreparationTimeout = &t
	}
	if r.WebhookConfig != nil {
		out.WebhookConfig = toProtoWebhookConfig(r.WebhookConfig)
	}
	return out
}

func toProtoWebhookConfig(c *gitprovider.WebhookConfig) *repositoryv1.RepositoryWebhookConfig {
	if c == nil {
		return nil
	}
	out := &repositoryv1.RepositoryWebhookConfig{
		Id:               c.ID,
		Url:              c.URL,
		Events:           c.Events,
		IsActive:         c.IsActive,
		NeedsManualSetup: c.NeedsManualSetup,
	}
	if c.LastError != "" {
		s := c.LastError
		out.LastError = &s
	}
	if c.CreatedAt != "" {
		s := c.CreatedAt
		out.CreatedAt = &s
	}
	return out
}

func toProtoWebhookStatus(s *gitprovider.WebhookStatus) *repositoryv1.WebhookStatus {
	if s == nil {
		return &repositoryv1.WebhookStatus{}
	}
	out := &repositoryv1.WebhookStatus{
		Registered:       s.Registered,
		Events:           s.Events,
		IsActive:         s.IsActive,
		NeedsManualSetup: s.NeedsManualSetup,
	}
	if s.WebhookID != "" {
		v := s.WebhookID
		out.WebhookId = &v
	}
	if s.WebhookURL != "" {
		v := s.WebhookURL
		out.WebhookUrl = &v
	}
	if s.LastError != "" {
		v := s.LastError
		out.LastError = &v
	}
	if s.RegisteredAt != "" {
		v := s.RegisteredAt
		out.RegisteredAt = &v
	}
	return out
}

func toProtoWebhookResult(r *repository.WebhookResult) *repositoryv1.WebhookResult {
	if r == nil {
		return nil
	}
	out := &repositoryv1.WebhookResult{
		RepoId:           r.RepoID,
		Registered:       r.Registered,
		NeedsManualSetup: r.NeedsManualSetup,
	}
	if r.WebhookID != "" {
		v := r.WebhookID
		out.WebhookId = &v
	}
	if r.ManualWebhookURL != "" {
		v := r.ManualWebhookURL
		out.ManualWebhookUrl = &v
	}
	if r.ManualWebhookSecret != "" {
		v := r.ManualWebhookSecret
		out.ManualWebhookSecret = &v
	}
	if r.Error != "" {
		v := r.Error
		out.ErrorMessage = &v
	}
	return out
}

func toProtoMergeRequest(mr *repository.MergeRequestInfo) *repositoryv1.MergeRequest {
	if mr == nil {
		return nil
	}
	return &repositoryv1.MergeRequest{
		Id:             mr.ID,
		MrIid:          int32(mr.MRIID),
		Title:          mr.Title,
		State:          mr.State,
		MrUrl:          mr.MRURL,
		SourceBranch:   mr.SourceBranch,
		TargetBranch:   mr.TargetBranch,
		PipelineStatus: mr.PipelineStatus,
		PipelineId:     mr.PipelineID,
		PipelineUrl:    mr.PipelineURL,
		TicketId:       mr.TicketID,
		PodId:          mr.PodID,
	}
}
