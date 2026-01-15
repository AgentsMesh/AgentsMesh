package git

import (
	"testing"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name         string
		providerType string
		baseURL      string
		accessToken  string
		expectError  bool
		errorIs      error
	}{
		{
			name:         "github provider",
			providerType: ProviderTypeGitHub,
			baseURL:      "https://api.github.com",
			accessToken:  "test-token",
			expectError:  false,
		},
		{
			name:         "gitlab provider",
			providerType: ProviderTypeGitLab,
			baseURL:      "https://gitlab.com/api/v4",
			accessToken:  "test-token",
			expectError:  false,
		},
		{
			name:         "gitee provider",
			providerType: ProviderTypeGitee,
			baseURL:      "https://gitee.com/api/v5",
			accessToken:  "test-token",
			expectError:  false,
		},
		{
			name:         "github with empty base url uses default",
			providerType: ProviderTypeGitHub,
			baseURL:      "",
			accessToken:  "test-token",
			expectError:  false,
		},
		{
			name:         "gitlab with empty base url uses default",
			providerType: ProviderTypeGitLab,
			baseURL:      "",
			accessToken:  "test-token",
			expectError:  false,
		},
		{
			name:         "gitee with empty base url uses default",
			providerType: ProviderTypeGitee,
			baseURL:      "",
			accessToken:  "test-token",
			expectError:  false,
		},
		{
			name:         "unsupported provider",
			providerType: "bitbucket",
			baseURL:      "",
			accessToken:  "test-token",
			expectError:  true,
			errorIs:      ErrProviderNotSupported,
		},
		{
			name:         "empty provider type",
			providerType: "",
			baseURL:      "",
			accessToken:  "test-token",
			expectError:  true,
			errorIs:      ErrProviderNotSupported,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.providerType, tt.baseURL, tt.accessToken)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errorIs != nil && err != tt.errorIs {
					t.Errorf("expected error %v, got %v", tt.errorIs, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if provider == nil {
					t.Error("expected provider, got nil")
				}
			}
		})
	}
}

func TestProviderTypeConstants(t *testing.T) {
	if ProviderTypeGitHub != "github" {
		t.Errorf("ProviderTypeGitHub = %s, want github", ProviderTypeGitHub)
	}
	if ProviderTypeGitLab != "gitlab" {
		t.Errorf("ProviderTypeGitLab = %s, want gitlab", ProviderTypeGitLab)
	}
	if ProviderTypeGitee != "gitee" {
		t.Errorf("ProviderTypeGitee = %s, want gitee", ProviderTypeGitee)
	}
}

func TestPipelineStatusConstants(t *testing.T) {
	statuses := []string{
		PipelineStatusPending,
		PipelineStatusRunning,
		PipelineStatusSuccess,
		PipelineStatusFailed,
		PipelineStatusCanceled,
		PipelineStatusSkipped,
		PipelineStatusManual,
	}
	expected := []string{"pending", "running", "success", "failed", "canceled", "skipped", "manual"}

	for i, status := range statuses {
		if status != expected[i] {
			t.Errorf("status[%d] = %s, want %s", i, status, expected[i])
		}
	}
}

func TestJobStatusConstants(t *testing.T) {
	statuses := []string{
		JobStatusCreated,
		JobStatusPending,
		JobStatusRunning,
		JobStatusSuccess,
		JobStatusFailed,
		JobStatusCanceled,
		JobStatusSkipped,
		JobStatusManual,
	}
	expected := []string{"created", "pending", "running", "success", "failed", "canceled", "skipped", "manual"}

	for i, status := range statuses {
		if status != expected[i] {
			t.Errorf("status[%d] = %s, want %s", i, status, expected[i])
		}
	}
}

func TestErrorValues(t *testing.T) {
	if ErrProviderNotSupported.Error() != "git provider not supported" {
		t.Errorf("ErrProviderNotSupported = %s", ErrProviderNotSupported.Error())
	}
	if ErrUnauthorized.Error() != "unauthorized" {
		t.Errorf("ErrUnauthorized = %s", ErrUnauthorized.Error())
	}
	if ErrNotFound.Error() != "resource not found" {
		t.Errorf("ErrNotFound = %s", ErrNotFound.Error())
	}
	if ErrRateLimited.Error() != "rate limited" {
		t.Errorf("ErrRateLimited = %s", ErrRateLimited.Error())
	}
}

func TestUserStruct(t *testing.T) {
	user := &User{
		ID:        "123",
		Username:  "testuser",
		Name:      "Test User",
		Email:     "test@example.com",
		AvatarURL: "https://example.com/avatar.png",
	}

	if user.ID != "123" {
		t.Errorf("User.ID = %s, want 123", user.ID)
	}
	if user.Username != "testuser" {
		t.Errorf("User.Username = %s, want testuser", user.Username)
	}
}

func TestProjectStruct(t *testing.T) {
	project := &Project{
		ID:            "456",
		Name:          "test-repo",
		FullPath:      "owner/test-repo",
		Description:   "Test repository",
		DefaultBranch: "main",
		WebURL:        "https://github.com/owner/test-repo",
		CloneURL:      "https://github.com/owner/test-repo.git",
		SSHCloneURL:   "git@github.com:owner/test-repo.git",
		Visibility:    "public",
	}

	if project.ID != "456" {
		t.Errorf("Project.ID = %s, want 456", project.ID)
	}
	if project.DefaultBranch != "main" {
		t.Errorf("Project.DefaultBranch = %s, want main", project.DefaultBranch)
	}
}

func TestBranchStruct(t *testing.T) {
	branch := &Branch{
		Name:      "feature-branch",
		CommitSHA: "abc123",
		Protected: true,
		Default:   false,
	}

	if branch.Name != "feature-branch" {
		t.Errorf("Branch.Name = %s, want feature-branch", branch.Name)
	}
	if !branch.Protected {
		t.Error("Branch.Protected should be true")
	}
}

func TestMergeRequestStruct(t *testing.T) {
	mr := &MergeRequest{
		ID:           1,
		IID:          10,
		Title:        "Test MR",
		Description:  "Test description",
		SourceBranch: "feature",
		TargetBranch: "main",
		State:        "opened",
		WebURL:       "https://github.com/owner/repo/pull/10",
		Author: &User{
			ID:       "123",
			Username: "testuser",
		},
	}

	if mr.IID != 10 {
		t.Errorf("MergeRequest.IID = %d, want 10", mr.IID)
	}
	if mr.Author.Username != "testuser" {
		t.Errorf("MergeRequest.Author.Username = %s, want testuser", mr.Author.Username)
	}
}

func TestCreateMRRequestStruct(t *testing.T) {
	req := &CreateMRRequest{
		ProjectID:    "owner/repo",
		Title:        "New Feature",
		Description:  "Description",
		SourceBranch: "feature",
		TargetBranch: "main",
	}

	if req.ProjectID != "owner/repo" {
		t.Errorf("CreateMRRequest.ProjectID = %s, want owner/repo", req.ProjectID)
	}
}

func TestCommitStruct(t *testing.T) {
	commit := &Commit{
		SHA:         "abc123def456",
		Message:     "Fix bug",
		Author:      "Test Author",
		AuthorEmail: "author@example.com",
	}

	if commit.SHA != "abc123def456" {
		t.Errorf("Commit.SHA = %s, want abc123def456", commit.SHA)
	}
}

func TestPipelineStruct(t *testing.T) {
	pipeline := &Pipeline{
		ID:        100,
		IID:       5,
		ProjectID: "owner/repo",
		Ref:       "main",
		SHA:       "abc123",
		Status:    PipelineStatusSuccess,
		Source:    "push",
		WebURL:    "https://gitlab.com/owner/repo/-/pipelines/100",
	}

	if pipeline.ID != 100 {
		t.Errorf("Pipeline.ID = %d, want 100", pipeline.ID)
	}
	if pipeline.Status != PipelineStatusSuccess {
		t.Errorf("Pipeline.Status = %s, want success", pipeline.Status)
	}
}

func TestJobStruct(t *testing.T) {
	job := &Job{
		ID:           200,
		Name:         "test",
		Stage:        "test",
		Status:       JobStatusSuccess,
		Ref:          "main",
		PipelineID:   100,
		WebURL:       "https://gitlab.com/owner/repo/-/jobs/200",
		AllowFailure: false,
		Duration:     120.5,
	}

	if job.ID != 200 {
		t.Errorf("Job.ID = %d, want 200", job.ID)
	}
	if job.Duration != 120.5 {
		t.Errorf("Job.Duration = %f, want 120.5", job.Duration)
	}
}

func TestTriggerPipelineRequestStruct(t *testing.T) {
	req := &TriggerPipelineRequest{
		Ref: "main",
		Variables: map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
		},
	}

	if req.Ref != "main" {
		t.Errorf("TriggerPipelineRequest.Ref = %s, want main", req.Ref)
	}
	if len(req.Variables) != 2 {
		t.Errorf("TriggerPipelineRequest.Variables len = %d, want 2", len(req.Variables))
	}
}

func TestWebhookConfigStruct(t *testing.T) {
	config := &WebhookConfig{
		URL:    "https://example.com/webhook",
		Secret: "webhook-secret",
		Events: []string{"push", "merge_request"},
	}

	if config.URL != "https://example.com/webhook" {
		t.Errorf("WebhookConfig.URL = %s", config.URL)
	}
	if len(config.Events) != 2 {
		t.Errorf("WebhookConfig.Events len = %d, want 2", len(config.Events))
	}
}
