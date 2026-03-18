package executor

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// LocalExecutor runs OpenTofu commands on the local machine (development only).
type LocalExecutor struct{}

func NewLocalExecutor() *LocalExecutor {
	return &LocalExecutor{}
}

var planSummaryRegex = regexp.MustCompile(`Plan: (\d+) to add, (\d+) to change, (\d+) to destroy`)

func (e *LocalExecutor) Execute(ctx context.Context, params ExecuteParams) (*ExecuteResult, error) {
	logger := slog.With("run_id", params.RunID, "operation", params.Operation)

	workDir, err := os.MkdirTemp("", fmt.Sprintf("tofui-run-%s-", params.RunID))
	if err != nil {
		return nil, fmt.Errorf("failed to create work dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	params.LogCallback([]byte(fmt.Sprintf("Preparing workspace for run %s...\r\n", params.RunID)))

	// Get source code: clone repo or extract uploaded archive
	if params.Source == "upload" {
		params.LogCallback([]byte("Extracting uploaded configuration...\r\n"))
		if err := extractArchive(params.ArchiveData, workDir); err != nil {
			params.LogCallback([]byte(fmt.Sprintf("\033[31mArchive extraction failed: %s\033[0m\r\n", err)))
			return nil, fmt.Errorf("archive extraction failed: %w", err)
		}
		params.LogCallback([]byte("Configuration extracted successfully.\r\n\r\n"))
	} else {
		params.LogCallback([]byte(fmt.Sprintf("Cloning %s (branch: %s)...\r\n", params.RepoURL, params.RepoBranch)))
		cloneCmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", params.RepoBranch, params.RepoURL, workDir)
		cloneOutput, err := cloneCmd.CombinedOutput()
		if err != nil {
			params.LogCallback([]byte(fmt.Sprintf("\033[31mGit clone failed: %s\033[0m\r\n", string(cloneOutput))))
			return nil, fmt.Errorf("git clone failed: %w", err)
		}
		params.LogCallback([]byte("Repository cloned successfully.\r\n\r\n"))
	}

	tfDir := filepath.Join(workDir, params.WorkingDir)

	// Restore previous state if available
	if len(params.PreviousState) > 0 {
		statePath := filepath.Join(tfDir, "terraform.tfstate")
		if err := os.WriteFile(statePath, params.PreviousState, 0600); err != nil {
			return nil, fmt.Errorf("failed to restore state: %w", err)
		}
		params.LogCallback([]byte("Restored previous state file.\r\n"))
		logger.Info("restored previous state", "size", len(params.PreviousState))
	}

	// Write encryption override if state encryption is enabled
	if params.StateEncryptionPassphrase != "" {
		overridePath := filepath.Join(tfDir, "tofui_encryption_override.tf")
		content := GenerateEncryptionOverride(params.StateEncryptionPassphrase)
		if err := os.WriteFile(overridePath, []byte(content), 0600); err != nil {
			return nil, fmt.Errorf("failed to write encryption override: %w", err)
		}
		params.LogCallback([]byte("State encryption enabled (AES-GCM).\r\n"))
	}

	// Write variables file if any
	if err := e.writeVariables(tfDir, params.Variables); err != nil {
		return nil, fmt.Errorf("failed to write variables: %w", err)
	}

	// Build environment with env variables
	env := append(os.Environ(), "TF_IN_AUTOMATION=true", "TF_INPUT=false")

	// Use plugin cache to avoid re-downloading providers every run
	if os.Getenv("TF_PLUGIN_CACHE_DIR") == "" {
		cacheDir := filepath.Join(os.TempDir(), "tofui-plugin-cache")
		os.MkdirAll(cacheDir, 0755)
		env = append(env, "TF_PLUGIN_CACHE_DIR="+cacheDir)
	}
	for _, v := range params.Variables {
		if v.Category == "env" {
			env = append(env, fmt.Sprintf("%s=%s", v.Key, v.Value))
		}
	}

	// tofu init
	params.LogCallback([]byte("\033[1m$ tofu init\033[0m\r\n"))
	if err := e.runTofu(ctx, tfDir, []string{"init", "-no-color"}, env, params.LogCallback); err != nil {
		return nil, fmt.Errorf("tofu init failed: %w", err)
	}
	params.LogCallback([]byte("\r\n"))

	// tofu validate
	params.LogCallback([]byte("\033[1m$ tofu validate\033[0m\r\n"))
	if err := e.runTofu(ctx, tfDir, []string{"validate", "-no-color"}, env, params.LogCallback); err != nil {
		return nil, fmt.Errorf("tofu validate failed: %w", err)
	}
	params.LogCallback([]byte("\r\n"))

	// Execute operation
	result := &ExecuteResult{}
	var tfArgs []string

	switch params.Operation {
	case "plan":
		tfArgs = []string{"plan", "-no-color", "-detailed-exitcode", "-out=planfile"}
		if e.hasVarFile(tfDir) {
			tfArgs = append(tfArgs, "-var-file=tofui.auto.tfvars")
		}
		params.LogCallback([]byte("\033[1m$ tofu plan\033[0m\r\n"))
	case "apply":
		tfArgs = []string{"apply", "-no-color", "-auto-approve"}
		if e.hasVarFile(tfDir) {
			tfArgs = append(tfArgs, "-var-file=tofui.auto.tfvars")
		}
		params.LogCallback([]byte("\033[1m$ tofu apply\033[0m\r\n"))
	case "destroy":
		tfArgs = []string{"destroy", "-no-color", "-auto-approve"}
		if e.hasVarFile(tfDir) {
			tfArgs = append(tfArgs, "-var-file=tofui.auto.tfvars")
		}
		params.LogCallback([]byte("\033[1m$ tofu destroy\033[0m\r\n"))
	default:
		return nil, fmt.Errorf("unknown operation: %s", params.Operation)
	}

	output, err := e.runTofuCapture(ctx, tfDir, tfArgs, env, params.LogCallback)
	if err != nil {
		if params.Operation == "plan" && strings.Contains(err.Error(), "exit status 2") {
			logger.Info("plan detected changes")
		} else {
			return nil, fmt.Errorf("tofu %s failed: %w", params.Operation, err)
		}
	}

	result.Output = output

	// Generate JSON plan from planfile (plan operation only)
	if params.Operation == "plan" {
		planfilePath := filepath.Join(tfDir, "planfile")
		if _, statErr := os.Stat(planfilePath); statErr == nil {
			jsonCmd := exec.CommandContext(ctx, "tofu", "show", "-json", "planfile")
			jsonCmd.Dir = tfDir
			jsonCmd.Env = env
			if jsonOut, jsonErr := jsonCmd.Output(); jsonErr == nil {
				result.PlanJSON = jsonOut
			} else {
				logger.Warn("failed to generate JSON plan", "error", jsonErr)
			}
		}
	}

	// Parse plan summary
	matches := planSummaryRegex.FindStringSubmatch(output)
	if len(matches) == 4 {
		added, _ := strconv.Atoi(matches[1])
		changed, _ := strconv.Atoi(matches[2])
		deleted, _ := strconv.Atoi(matches[3])
		result.ResourcesAdded = int32(added)
		result.ResourcesChanged = int32(changed)
		result.ResourcesDeleted = int32(deleted)
	}

	// Capture state after apply/destroy
	if params.Operation == "apply" || params.Operation == "destroy" {
		// Raw state file (may be encrypted) — used for restoration on next run
		statePath := filepath.Join(tfDir, "terraform.tfstate")
		if stateData, err := os.ReadFile(statePath); err == nil && len(stateData) > 0 {
			result.StateFile = stateData
			logger.Info("captured state file", "size", len(stateData))
		}

		// Decrypted state via "tofu state pull" — used for resource browsing
		pullCmd := exec.CommandContext(ctx, "tofu", "state", "pull")
		pullCmd.Dir = tfDir
		pullCmd.Env = env
		if jsonData, err := pullCmd.Output(); err == nil && len(jsonData) > 0 {
			result.StateJSON = jsonData
		}
	}

	return result, nil
}

// isHCLLiteral returns true if the value looks like an HCL literal
// (map, list, number, bool) that should not be quoted in tfvars.
func isHCLLiteral(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" {
		return false
	}
	// Maps and objects: { ... }
	if strings.HasPrefix(v, "{") && strings.HasSuffix(v, "}") {
		return true
	}
	// Lists and tuples: [ ... ]
	if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
		return true
	}
	// Booleans
	if v == "true" || v == "false" {
		return true
	}
	// Numbers — must consume the entire string
	var f float64
	var trailing string
	if n, _ := fmt.Sscanf(v, "%f%s", &f, &trailing); n == 1 {
		return true
	}
	return false
}

func (e *LocalExecutor) writeVariables(tfDir string, vars []Variable) error {
	var tfVars []string
	for _, v := range vars {
		if v.Category == "terraform" {
			if isHCLLiteral(v.Value) {
				tfVars = append(tfVars, fmt.Sprintf("%s = %s", v.Key, v.Value))
			} else {
				tfVars = append(tfVars, fmt.Sprintf("%s = %q", v.Key, v.Value))
			}
		}
	}
	if len(tfVars) == 0 {
		return nil
	}
	content := strings.Join(tfVars, "\n") + "\n"
	return os.WriteFile(filepath.Join(tfDir, "tofui.auto.tfvars"), []byte(content), 0600)
}

func (e *LocalExecutor) hasVarFile(tfDir string) bool {
	_, err := os.Stat(filepath.Join(tfDir, "tofui.auto.tfvars"))
	return err == nil
}

func (e *LocalExecutor) runTofu(ctx context.Context, dir string, args, env []string, logCallback func([]byte)) error {
	_, err := e.runTofuCapture(ctx, dir, args, env, logCallback)
	return err
}

func extractArchive(data []byte, destDir string) error {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("invalid gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Prevent path traversal
		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "..") {
			return fmt.Errorf("invalid path in archive: %s", hdr.Name)
		}

		target := filepath.Join(destDir, cleanName)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}

func (e *LocalExecutor) runTofuCapture(ctx context.Context, dir string, args, env []string, logCallback func([]byte)) (string, error) {
	cmd := exec.CommandContext(ctx, "tofu", args...)
	cmd.Dir = dir
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return "", err
	}

	var output strings.Builder
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(line)
		output.WriteString("\n")
		logCallback([]byte(line + "\r\n"))
	}

	if scanner.Err() != nil {
		remaining, _ := io.ReadAll(stdout)
		if len(remaining) > 0 {
			output.Write(remaining)
			logCallback(remaining)
		}
	}

	err = cmd.Wait()
	return output.String(), err
}
