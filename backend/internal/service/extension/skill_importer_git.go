package extension

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
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
		return cloneHTTPS(ctx, repoURL, branch, targetDir, credential, "")
	case extension.AuthTypeGitLabPAT:
		return cloneHTTPS(ctx, repoURL, branch, targetDir, "oauth2", credential)
	case extension.AuthTypeSSHKey:
		return gitCloneWithSSHKey(ctx, repoURL, branch, targetDir, credential)
	default:
		return gitClone(ctx, repoURL, branch, targetDir)
	}
}

// httpsBasicAuth builds explicit BasicAuth for an https repo. Credentials go
// through go-git's Auth, never embedded in the clone URL, so they cannot leak
// into go-git errors, the persisted sync_error, or logs.
func httpsBasicAuth(repoURL, username, password string) (*githttp.BasicAuth, error) {
	if !strings.HasPrefix(repoURL, "https://") {
		return nil, fmt.Errorf("PAT auth requires https:// URL, got: %s", repoURL)
	}
	return &githttp.BasicAuth{Username: username, Password: password}, nil
}

func cloneHTTPS(ctx context.Context, repoURL, branch, targetDir, username, password string) error {
	auth, err := httpsBasicAuth(repoURL, username, password)
	if err != nil {
		return fmt.Errorf("failed to build authenticated URL: %w", err)
	}
	return cloneRef(ctx, targetDir, repoURL, branch, auth, true, "git clone failed")
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

	// Local-path clones (tests, file:// sources) never touch SSH, so an
	// unparseable key must not fail them — only remote git@ uses auth.
	var auth transport.AuthMethod
	if isGitSSH {
		keys, err := gitssh.NewPublicKeys("git", []byte(sshKey), "")
		if err != nil {
			return fmt.Errorf("git clone with SSH key failed: %w", err)
		}
		keys.HostKeyCallback = xssh.InsecureIgnoreHostKey()
		auth = keys
	}

	return cloneRef(ctx, targetDir, repoURL, branch, auth, !isLocalPath, "git clone with SSH key failed")
}

func gitClone(ctx context.Context, rawURL, branch, targetDir string) error {
	if !strings.HasPrefix(rawURL, "https://") {
		return fmt.Errorf("only https:// URLs are allowed for git clone, got: %s", rawURL)
	}
	return cloneRef(ctx, targetDir, rawURL, branch, nil, true, "git clone failed")
}

// cloneRef clones url into targetDir. A non-empty branch is resolved as a branch
// first and, if the remote has no such head, retried as a tag — matching
// `git clone --branch`, which accepts either (go-git needs the exact ref form).
func cloneRef(ctx context.Context, targetDir, url, branch string, auth transport.AuthMethod, shallow bool, errPrefix string) error {
	if err := validateBranchIfSet(branch); err != nil {
		return err
	}
	if branch == "" {
		return runClone(ctx, targetDir, cloneOptions(url, "", auth, shallow), errPrefix)
	}
	err := runClone(ctx, targetDir, cloneOptions(url, plumbing.NewBranchReferenceName(branch), auth, shallow), errPrefix)
	if errors.Is(err, git.NoMatchingRefSpecError{}) {
		_ = os.RemoveAll(targetDir) // go-git leaves a partial repo on the failed head fetch
		return runClone(ctx, targetDir, cloneOptions(url, plumbing.NewTagReferenceName(branch), auth, shallow), errPrefix)
	}
	return err
}

// cloneOptions mirrors `git clone --depth 1 --single-branch`. Tags:NoTags is
// explicit because go-git's Validate() otherwise defaults to AllTags, fetching
// every tag on each sync — the old CLI fetched none.
func cloneOptions(cloneURL string, ref plumbing.ReferenceName, auth transport.AuthMethod, shallow bool) *git.CloneOptions {
	opts := &git.CloneOptions{URL: cloneURL, Auth: auth, Tags: git.NoTags}
	if shallow {
		opts.Depth = 1
	}
	if ref != "" {
		opts.SingleBranch = true
		opts.ReferenceName = ref
	}
	return opts
}

// runClone treats an empty remote as success (0 skills), mirroring `git clone`
// exiting 0 on an empty repo. go-git checkout runs no Git-LFS smudge or
// autocrlf/.gitattributes filters, so skill sources must be plain-text repos.
func runClone(ctx context.Context, targetDir string, opts *git.CloneOptions, errPrefix string) error {
	_, err := git.PlainCloneContext(ctx, targetDir, false, opts)
	if errors.Is(err, transport.ErrEmptyRemoteRepository) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: %w", errPrefix, err)
	}
	return nil
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
