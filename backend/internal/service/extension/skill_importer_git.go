package extension

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	xssh "golang.org/x/crypto/ssh"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

func validateGitBranch(branch string) error {
	for _, c := range branch {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '/' {
			continue
		}
		return fmt.Errorf("invalid branch name character: %c", c)
	}
	return nil
}

func validateBranchIfSet(branch string) error {
	if branch == "" {
		return nil
	}
	if err := validateGitBranch(branch); err != nil {
		return fmt.Errorf("invalid branch: %w", err)
	}
	return nil
}

func gitCloneWithAuth(ctx context.Context, repoURL, branch, targetDir, authType, credential string) error {
	slog.InfoContext(ctx, "git clone with auth", "auth_type", authType, "branch", branch)
	switch authType {
	case extension.AuthTypeGitHubPAT:
		authedURL, err := injectPATIntoURL(repoURL, credential)
		if err != nil {
			return fmt.Errorf("failed to build authenticated URL: %w", err)
		}
		return gitClone(ctx, authedURL, branch, targetDir)

	case extension.AuthTypeGitLabPAT:
		authedURL, err := injectGitLabPATIntoURL(repoURL, credential)
		if err != nil {
			return fmt.Errorf("failed to build authenticated URL: %w", err)
		}
		return gitClone(ctx, authedURL, branch, targetDir)

	case extension.AuthTypeSSHKey:
		return gitCloneWithSSHKey(ctx, repoURL, branch, targetDir, credential)

	default:
		return gitClone(ctx, repoURL, branch, targetDir)
	}
}

func injectPATIntoURL(repoURL, token string) (string, error) {
	if !strings.HasPrefix(repoURL, "https://") {
		return "", fmt.Errorf("PAT auth requires https:// URL, got: %s", repoURL)
	}
	rest := strings.TrimPrefix(repoURL, "https://")
	return fmt.Sprintf("https://%s@%s", token, rest), nil
}

// injectGitLabPATIntoURL uses the oauth2 username form GitLab requires.
func injectGitLabPATIntoURL(repoURL, token string) (string, error) {
	if !strings.HasPrefix(repoURL, "https://") {
		return "", fmt.Errorf("PAT auth requires https:// URL, got: %s", repoURL)
	}
	rest := strings.TrimPrefix(repoURL, "https://")
	return fmt.Sprintf("https://oauth2:%s@%s", token, rest), nil
}

// splitBasicAuth pulls userinfo out of an https URL into an explicit
// go-git BasicAuth, returning a credential-free URL. Keeping creds off the
// clone URL means they can never leak into go-git error messages.
func splitBasicAuth(rawURL string) (string, transport.AuthMethod, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", nil, fmt.Errorf("invalid repository URL: %w", err)
	}
	if u.User == nil {
		return rawURL, nil, nil
	}
	password, _ := u.User.Password()
	auth := &githttp.BasicAuth{Username: u.User.Username(), Password: password}
	u.User = nil
	return u.String(), auth, nil
}

func cloneOptions(cloneURL, branch string, auth transport.AuthMethod, shallow bool) *git.CloneOptions {
	opts := &git.CloneOptions{URL: cloneURL, Auth: auth}
	if shallow {
		opts.Depth = 1
	}
	if branch != "" {
		opts.SingleBranch = true
		opts.ReferenceName = plumbing.NewBranchReferenceName(branch)
	}
	return opts
}

func runClone(ctx context.Context, targetDir string, opts *git.CloneOptions, errPrefix string) error {
	if _, err := git.PlainCloneContext(ctx, targetDir, false, opts); err != nil {
		return fmt.Errorf("%s: %w", errPrefix, err)
	}
	return nil
}

func gitCloneWithSSHKey(ctx context.Context, repoURL, branch, targetDir, sshKey string) error {
	isGitSSH := strings.HasPrefix(repoURL, "git@")
	isLocalPath := strings.HasPrefix(repoURL, "/") || strings.HasPrefix(repoURL, ".")
	if !isGitSSH && !isLocalPath {
		return fmt.Errorf("SSH key auth requires git@ URL, got: %s", repoURL)
	}
	if err := validateBranchIfSet(branch); err != nil {
		return err
	}

	// Local-path clones (used in tests and file:// sources) never touch SSH,
	// so an unparseable key must not fail them — only remote git@ uses auth.
	var auth transport.AuthMethod
	if isGitSSH {
		keys, err := gitssh.NewPublicKeys("git", []byte(sshKey), "")
		if err != nil {
			return fmt.Errorf("git clone with SSH key failed: %w", err)
		}
		keys.HostKeyCallback = xssh.InsecureIgnoreHostKey()
		auth = keys
	}

	return runClone(ctx, targetDir, cloneOptions(repoURL, branch, auth, !isLocalPath), "git clone with SSH key failed")
}

func gitClone(ctx context.Context, rawURL, branch, targetDir string) error {
	if !strings.HasPrefix(rawURL, "https://") {
		return fmt.Errorf("only https:// URLs are allowed for git clone, got: %s", rawURL)
	}
	if err := validateBranchIfSet(branch); err != nil {
		return err
	}
	cloneURL, auth, err := splitBasicAuth(rawURL)
	if err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}
	return runClone(ctx, targetDir, cloneOptions(cloneURL, branch, auth, true), "git clone failed")
}

func gitHead(_ context.Context, repoDir string) (string, error) {
	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		return "", err
	}
	head, err := repo.Head()
	if err != nil {
		return "", err
	}
	return head.Hash().String(), nil
}
