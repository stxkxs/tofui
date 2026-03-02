package vcs

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifySignature(t *testing.T) {
	secret := "test-secret-key"
	body := []byte(`{"ref":"refs/heads/main"}`)

	// Compute valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name      string
		body      []byte
		signature string
		secret    string
		want      bool
	}{
		{"valid signature", body, validSig, secret, true},
		{"wrong secret", body, validSig, "wrong-secret", false},
		{"empty secret", body, validSig, "", false},
		{"empty signature", body, "", secret, false},
		{"tampered body", []byte(`{"ref":"refs/heads/evil"}`), validSig, secret, false},
		{"missing sha256 prefix", body, "abc123", secret, false},
		{"invalid hex", body, "sha256=zzzz", secret, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifySignature(tt.body, tt.signature, tt.secret)
			if got != tt.want {
				t.Errorf("VerifySignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParsePushEvent(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantErr    bool
		wantBranch string
		wantRepo   string
		wantSender string
	}{
		{
			name: "valid push event",
			body: `{
				"ref": "refs/heads/main",
				"repository": {
					"clone_url": "https://github.com/org/repo.git",
					"html_url": "https://github.com/org/repo"
				},
				"head_commit": {
					"id": "abc123",
					"message": "fix: something"
				},
				"sender": {
					"login": "octocat"
				}
			}`,
			wantErr:    false,
			wantBranch: "main",
			wantRepo:   "https://github.com/org/repo",
			wantSender: "octocat",
		},
		{
			name: "feature branch",
			body: `{
				"ref": "refs/heads/feature/my-feature",
				"repository": {
					"clone_url": "https://github.com/org/repo.git",
					"html_url": "https://github.com/org/repo"
				},
				"sender": {"login": "dev"}
			}`,
			wantErr:    false,
			wantBranch: "feature/my-feature",
			wantRepo:   "https://github.com/org/repo",
			wantSender: "dev",
		},
		{
			name: "tag push rejected",
			body: `{
				"ref": "refs/tags/v1.0.0",
				"repository": {
					"clone_url": "https://github.com/org/repo.git",
					"html_url": "https://github.com/org/repo"
				},
				"sender": {"login": "dev"}
			}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			body:    `{invalid`,
			wantErr: true,
		},
		{
			name: "missing ref",
			body: `{
				"repository": {
					"clone_url": "https://github.com/org/repo.git"
				}
			}`,
			wantErr: true,
		},
		{
			name: "missing repository",
			body: `{
				"ref": "refs/heads/main"
			}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParsePushEvent([]byte(tt.body))
			if tt.wantErr {
				if err == nil {
					t.Error("ParsePushEvent() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePushEvent() unexpected error: %v", err)
			}
			if event.Branch() != tt.wantBranch {
				t.Errorf("Branch() = %q, want %q", event.Branch(), tt.wantBranch)
			}
			if event.RepoURL != tt.wantRepo {
				t.Errorf("RepoURL = %q, want %q", event.RepoURL, tt.wantRepo)
			}
			if event.SenderName != tt.wantSender {
				t.Errorf("SenderName = %q, want %q", event.SenderName, tt.wantSender)
			}
		})
	}
}

func TestNormalizeRepoURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/Org/Repo.git", "https://github.com/org/repo"},
		{"https://github.com/org/repo", "https://github.com/org/repo"},
		{"HTTPS://GITHUB.COM/ORG/REPO.GIT", "https://github.com/org/repo"},
	}

	for _, tt := range tests {
		got := NormalizeRepoURL(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeRepoURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
