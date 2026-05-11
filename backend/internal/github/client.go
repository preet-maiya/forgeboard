package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const apiBase = "https://api.github.com"

// Client wraps GitHub REST API calls using a personal access token.
// Stdlib only — no external dependencies.
type Client struct {
	owner   string
	repo    string
	token   string
	botUser string
	hc      *http.Client
}

// NewClient creates a Client from "owner/repo" and a PAT.
// It verifies the token by fetching the authenticated user.
func NewClient(ownerRepo, token string) (*Client, error) {
	parts := strings.SplitN(ownerRepo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("GITHUB_REPO must be owner/repo, got %q", ownerRepo)
	}
	c := &Client{
		owner: parts[0],
		repo:  parts[1],
		token: token,
		hc:    &http.Client{},
	}
	user, err := c.getAuthenticatedUser()
	if err != nil {
		return nil, fmt.Errorf("github auth: %w", err)
	}
	c.botUser = user
	return c, nil
}

// BotUser returns the GitHub login of the PAT owner (used as review bot identity).
func (c *Client) BotUser() string { return c.botUser }

// ========== HTTP helpers ==========

func (c *Client) do(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, apiBase+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.hc.Do(req)
}

func (c *Client) doJSON(method, path string, body interface{}, out interface{}) error {
	resp, err := c.do(method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		return &RateLimitError{StatusCode: resp.StatusCode, Body: string(data)}
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("github %s %s: %d %s", method, path, resp.StatusCode, string(data))
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

// RateLimitError is returned when GitHub responds with 403 or 429.
type RateLimitError struct {
	StatusCode int
	Body       string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("github rate limit %d: %s", e.StatusCode, e.Body)
}

// IsRateLimit returns true if err is a GitHub rate-limit response.
func IsRateLimit(err error) bool {
	_, ok := err.(*RateLimitError)
	return ok
}

// ========== User ==========

func (c *Client) getAuthenticatedUser() (string, error) {
	var u struct {
		Login string `json:"login"`
	}
	if err := c.doJSON("GET", "/user", nil, &u); err != nil {
		return "", err
	}
	return u.Login, nil
}

// ========== Branch ==========

// BranchExists returns true if branch exists in the repo.
func (c *Client) BranchExists(branch string) (bool, error) {
	resp, err := c.do("GET",
		fmt.Sprintf("/repos/%s/%s/branches/%s", c.owner, c.repo, encodeBranch(branch)), nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body) // drain
	switch resp.StatusCode {
	case 200:
		return true, nil
	case 404:
		return false, nil
	default:
		return false, fmt.Errorf("BranchExists %s: %d", branch, resp.StatusCode)
	}
}

// GetBranchCommitSHA returns the latest commit SHA for branch.
func (c *Client) GetBranchCommitSHA(branch string) (string, error) {
	var b struct {
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if err := c.doJSON("GET",
		fmt.Sprintf("/repos/%s/%s/branches/%s", c.owner, c.repo, encodeBranch(branch)),
		nil, &b); err != nil {
		return "", err
	}
	return b.Commit.SHA, nil
}

// CreateBranch creates a new branch off base.
func (c *Client) CreateBranch(branch, base string) error {
	sha, err := c.GetBranchCommitSHA(base)
	if err != nil {
		return fmt.Errorf("get base branch SHA: %w", err)
	}
	return c.doJSON("POST",
		fmt.Sprintf("/repos/%s/%s/git/refs", c.owner, c.repo),
		map[string]string{"ref": "refs/heads/" + branch, "sha": sha},
		nil)
}

// ========== Push (Git Data API) ==========

type treeEntry struct {
	Path    string `json:"path"`
	Mode    string `json:"mode"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

// PushFiles pushes files (repo-relative-path → content) to branch.
func (c *Client) PushFiles(branch string, files map[string]string) error {
	// Get current branch commit SHA
	commitSHA, err := c.GetBranchCommitSHA(branch)
	if err != nil {
		return fmt.Errorf("get branch commit SHA: %w", err)
	}

	// Get tree SHA of that commit
	var commit struct {
		Tree struct {
			SHA string `json:"sha"`
		} `json:"tree"`
	}
	if err := c.doJSON("GET",
		fmt.Sprintf("/repos/%s/%s/git/commits/%s", c.owner, c.repo, commitSHA),
		nil, &commit); err != nil {
		return fmt.Errorf("get commit tree SHA: %w", err)
	}

	// Build tree entries (inline content for text files)
	entries := make([]treeEntry, 0, len(files))
	for path, content := range files {
		entries = append(entries, treeEntry{
			Path:    path,
			Mode:    "100644",
			Type:    "blob",
			Content: content,
		})
	}

	// Create tree
	var treeResp struct {
		SHA string `json:"sha"`
	}
	if err := c.doJSON("POST",
		fmt.Sprintf("/repos/%s/%s/git/trees", c.owner, c.repo),
		map[string]interface{}{"base_tree": commit.Tree.SHA, "tree": entries},
		&treeResp); err != nil {
		return fmt.Errorf("create tree: %w", err)
	}

	// Create commit
	var newCommit struct {
		SHA string `json:"sha"`
	}
	if err := c.doJSON("POST",
		fmt.Sprintf("/repos/%s/%s/git/commits", c.owner, c.repo),
		map[string]interface{}{
			"message": "chore: push task artifacts [forgeboard]",
			"tree":    treeResp.SHA,
			"parents": []string{commitSHA},
		},
		&newCommit); err != nil {
		return fmt.Errorf("create commit: %w", err)
	}

	// Update branch ref
	if err := c.doJSON("PATCH",
		fmt.Sprintf("/repos/%s/%s/git/refs/heads/%s", c.owner, c.repo, encodeBranch(branch)),
		map[string]interface{}{"sha": newCommit.SHA, "force": false},
		nil); err != nil {
		return fmt.Errorf("update ref: %w", err)
	}

	return nil
}

// ========== PR ==========

// FindOpenPR returns the PR number for an open PR on branch, or 0 if none.
func (c *Client) FindOpenPR(branch string) (int, error) {
	var prs []struct {
		Number int `json:"number"`
	}
	if err := c.doJSON("GET",
		fmt.Sprintf("/repos/%s/%s/pulls?state=open&head=%s:%s", c.owner, c.repo, c.owner, encodeBranch(branch)),
		nil, &prs); err != nil {
		return 0, err
	}
	if len(prs) == 0 {
		return 0, nil
	}
	return prs[0].Number, nil
}

// CreatePR opens a new pull request and returns its number.
func (c *Client) CreatePR(title, body, branch, base string) (int, error) {
	var pr struct {
		Number int `json:"number"`
	}
	if err := c.doJSON("POST",
		fmt.Sprintf("/repos/%s/%s/pulls", c.owner, c.repo),
		map[string]string{"title": title, "body": body, "head": branch, "base": base},
		&pr); err != nil {
		return 0, err
	}
	return pr.Number, nil
}

// GetPRDiff returns the unified diff for a PR.
func (c *Client) GetPRDiff(prNumber int) (string, error) {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/repos/%s/%s/pulls/%d", apiBase, c.owner, c.repo, prNumber), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3.diff")
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GetPRDiff: %d %s", resp.StatusCode, string(data))
	}
	return string(data), nil
}

// IsPROpen returns true if the PR is in "open" state.
func (c *Client) IsPROpen(prNumber int) (bool, error) {
	var pr struct {
		State string `json:"state"`
	}
	if err := c.doJSON("GET",
		fmt.Sprintf("/repos/%s/%s/pulls/%d", c.owner, c.repo, prNumber),
		nil, &pr); err != nil {
		return false, err
	}
	return pr.State == "open", nil
}

// ========== Reviews ==========

// Review represents a single PR review.
type Review struct {
	ID    int    `json:"id"`
	State string `json:"state"` // APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED
	Body  string `json:"body"`
	User  struct {
		Login string `json:"login"`
	} `json:"user"`
}

// GetPRReviews returns all reviews for a PR.
func (c *Client) GetPRReviews(prNumber int) ([]Review, error) {
	var reviews []Review
	if err := c.doJSON("GET",
		fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", c.owner, c.repo, prNumber),
		nil, &reviews); err != nil {
		return nil, err
	}
	return reviews, nil
}

// PostPRReview submits a PR review.
// event must be "APPROVE", "REQUEST_CHANGES", or "COMMENT".
func (c *Client) PostPRReview(prNumber int, event, body string) error {
	return c.doJSON("POST",
		fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", c.owner, c.repo, prNumber),
		map[string]string{"event": event, "body": body},
		nil)
}

// ========== File helpers ==========

// CollectTaskFiles walks taskDir and returns repo-relative-path → content.
// repoRelDir is the repo-relative directory prefix (e.g. "tasks/task-0001").
func CollectTaskFiles(taskDir, repoRelDir string) (map[string]string, error) {
	files := make(map[string]string)
	err := filepath.Walk(taskDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(taskDir, path)
		if err != nil {
			return err
		}
		repoPath := repoRelDir + "/" + filepath.ToSlash(rel)
		files[repoPath] = string(data)
		return nil
	})
	return files, err
}

// ========== Helpers ==========

// encodeBranch replaces "/" with "%2F" for use in URL path segments.
func encodeBranch(branch string) string {
	return strings.ReplaceAll(branch, "/", "%2F")
}
