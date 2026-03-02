package executor

import "context"

// Variable represents an OpenTofu or environment variable.
type Variable struct {
	Key      string
	Value    string
	Category string // "terraform" or "env"
}

// ExecuteParams holds the parameters for running OpenTofu.
type ExecuteParams struct {
	RunID       string
	WorkspaceID string
	Operation   string // "plan", "apply", "destroy"
	RepoURL     string
	RepoBranch  string
	WorkingDir  string
	TofuVersion string
	Variables   []Variable
	LogCallback func([]byte)

	// PreviousState is the state file from the last successful apply.
	// If non-nil, it is restored as terraform.tfstate before execution.
	PreviousState []byte

	// StateEncryptionPassphrase enables OpenTofu 1.7+ native state encryption.
	// When set, an encryption override file is written with PBKDF2+AES-GCM.
	StateEncryptionPassphrase string
}

// ExecuteResult holds the outcome of an OpenTofu execution.
type ExecuteResult struct {
	Output           string
	ResourcesAdded   int32
	ResourcesChanged int32
	ResourcesDeleted int32
	StateFile        []byte
	PlanJSON         []byte
}

// Executor runs OpenTofu commands in an isolated environment.
type Executor interface {
	Execute(ctx context.Context, params ExecuteParams) (*ExecuteResult, error)
}
