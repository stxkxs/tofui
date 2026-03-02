package vcs

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// PushEvent represents the relevant fields from a GitHub push webhook event.
type PushEvent struct {
	Ref        string `json:"ref"`         // e.g. "refs/heads/main"
	RepoURL    string `json:"repo_url"`    // normalized clone URL
	CommitSHA  string `json:"commit_sha"`  // head commit SHA
	CommitMsg  string `json:"commit_msg"`  // head commit message
	SenderName string `json:"sender_name"` // who pushed
}

// Branch extracts the branch name from the ref (e.g. "refs/heads/main" → "main").
func (e PushEvent) Branch() string {
	return strings.TrimPrefix(e.Ref, "refs/heads/")
}

// githubPushPayload is the subset of GitHub's push webhook JSON we care about.
type githubPushPayload struct {
	Ref        string `json:"ref"`
	Repository struct {
		CloneURL string `json:"clone_url"`
		HTMLURL  string `json:"html_url"`
	} `json:"repository"`
	HeadCommit *struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	} `json:"head_commit"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
}

// ParsePushEvent parses a GitHub push webhook payload into a PushEvent.
// Returns an error if the JSON is invalid or it's not a branch push.
func ParsePushEvent(body []byte) (PushEvent, error) {
	var p githubPushPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return PushEvent{}, fmt.Errorf("invalid push event JSON: %w", err)
	}

	if p.Ref == "" || p.Repository.CloneURL == "" {
		return PushEvent{}, fmt.Errorf("missing required fields (ref or repository)")
	}

	// Only handle branch pushes, not tag pushes
	if !strings.HasPrefix(p.Ref, "refs/heads/") {
		return PushEvent{}, fmt.Errorf("not a branch push: %s", p.Ref)
	}

	// Normalize repo URL: use clone_url as canonical, strip .git suffix for matching
	repoURL := normalizeRepoURL(p.Repository.CloneURL)

	event := PushEvent{
		Ref:        p.Ref,
		RepoURL:    repoURL,
		SenderName: p.Sender.Login,
	}

	if p.HeadCommit != nil {
		event.CommitSHA = p.HeadCommit.ID
		event.CommitMsg = p.HeadCommit.Message
	}

	return event, nil
}

// VerifySignature validates the GitHub webhook HMAC-SHA256 signature.
// signatureHeader is the value of the X-Hub-Signature-256 header (e.g. "sha256=abc123...").
func VerifySignature(body []byte, signatureHeader, secret string) bool {
	if secret == "" || signatureHeader == "" {
		return false
	}

	// GitHub sends "sha256=<hex>"
	if !strings.HasPrefix(signatureHeader, "sha256=") {
		return false
	}
	sigHex := strings.TrimPrefix(signatureHeader, "sha256=")

	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := mac.Sum(nil)

	return hmac.Equal(sig, expected)
}

// normalizeRepoURL lowercases and strips the .git suffix for consistent matching.
func normalizeRepoURL(url string) string {
	url = strings.ToLower(url)
	url = strings.TrimSuffix(url, ".git")
	return url
}

// NormalizeRepoURL is the exported version for use in workspace matching.
func NormalizeRepoURL(url string) string {
	return normalizeRepoURL(url)
}
