package git

import (
	"context"
	"testing"
)

func TestGiteePipelineNotSupported(t *testing.T) {
	ctx := context.Background()
	provider, _ := NewGiteeProvider("", "test-token")

	t.Run("trigger pipeline not supported", func(t *testing.T) {
		_, err := provider.TriggerPipeline(ctx, "owner/repo", &TriggerPipelineRequest{Ref: "main"})
		if err != ErrGiteePipelineNotSupported {
			t.Errorf("expected ErrGiteePipelineNotSupported, got %v", err)
		}
	})

	t.Run("get pipeline not supported", func(t *testing.T) {
		_, err := provider.GetPipeline(ctx, "owner/repo", 1001)
		if err != ErrGiteePipelineNotSupported {
			t.Errorf("expected ErrGiteePipelineNotSupported, got %v", err)
		}
	})

	t.Run("list pipelines not supported", func(t *testing.T) {
		_, err := provider.ListPipelines(ctx, "owner/repo", "main", "success", 1, 20)
		if err != ErrGiteePipelineNotSupported {
			t.Errorf("expected ErrGiteePipelineNotSupported, got %v", err)
		}
	})

	t.Run("cancel pipeline not supported", func(t *testing.T) {
		_, err := provider.CancelPipeline(ctx, "owner/repo", 1001)
		if err != ErrGiteePipelineNotSupported {
			t.Errorf("expected ErrGiteePipelineNotSupported, got %v", err)
		}
	})

	t.Run("retry pipeline not supported", func(t *testing.T) {
		_, err := provider.RetryPipeline(ctx, "owner/repo", 1001)
		if err != ErrGiteePipelineNotSupported {
			t.Errorf("expected ErrGiteePipelineNotSupported, got %v", err)
		}
	})
}

func TestGiteeJobNotSupported(t *testing.T) {
	ctx := context.Background()
	provider, _ := NewGiteeProvider("", "test-token")

	t.Run("get job not supported", func(t *testing.T) {
		_, err := provider.GetJob(ctx, "owner/repo", 2001)
		if err != ErrGiteePipelineNotSupported {
			t.Errorf("expected ErrGiteePipelineNotSupported, got %v", err)
		}
	})

	t.Run("list pipeline jobs not supported", func(t *testing.T) {
		_, err := provider.ListPipelineJobs(ctx, "owner/repo", 1001)
		if err != ErrGiteePipelineNotSupported {
			t.Errorf("expected ErrGiteePipelineNotSupported, got %v", err)
		}
	})

	t.Run("retry job not supported", func(t *testing.T) {
		_, err := provider.RetryJob(ctx, "owner/repo", 2001)
		if err != ErrGiteePipelineNotSupported {
			t.Errorf("expected ErrGiteePipelineNotSupported, got %v", err)
		}
	})

	t.Run("cancel job not supported", func(t *testing.T) {
		_, err := provider.CancelJob(ctx, "owner/repo", 2001)
		if err != ErrGiteePipelineNotSupported {
			t.Errorf("expected ErrGiteePipelineNotSupported, got %v", err)
		}
	})

	t.Run("get job trace not supported", func(t *testing.T) {
		_, err := provider.GetJobTrace(ctx, "owner/repo", 2001)
		if err != ErrGiteePipelineNotSupported {
			t.Errorf("expected ErrGiteePipelineNotSupported, got %v", err)
		}
	})

	t.Run("get job artifact not supported", func(t *testing.T) {
		_, err := provider.GetJobArtifact(ctx, "owner/repo", 2001, "artifact.zip")
		if err != ErrGiteePipelineNotSupported {
			t.Errorf("expected ErrGiteePipelineNotSupported, got %v", err)
		}
	})

	t.Run("download job artifacts not supported", func(t *testing.T) {
		_, err := provider.DownloadJobArtifacts(ctx, "owner/repo", 2001)
		if err != ErrGiteePipelineNotSupported {
			t.Errorf("expected ErrGiteePipelineNotSupported, got %v", err)
		}
	})
}

func TestGiteeErrorVariableValue(t *testing.T) {
	if ErrGiteePipelineNotSupported.Error() != "gitee pipeline API not fully supported" {
		t.Errorf("ErrGiteePipelineNotSupported message = %s", ErrGiteePipelineNotSupported.Error())
	}
}
