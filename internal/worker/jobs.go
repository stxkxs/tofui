package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
	"github.com/riverqueue/river"

	"github.com/stxkxs/tofui/internal/logstream"
	"github.com/stxkxs/tofui/internal/repository"
	"github.com/stxkxs/tofui/internal/secrets"
	"github.com/stxkxs/tofui/internal/storage"
	"github.com/stxkxs/tofui/internal/worker/executor"
)

type RunJobArgs struct {
	RunID       string `json:"run_id"`
	WorkspaceID string `json:"workspace_id"`
	OrgID       string `json:"org_id"`
	Operation   string `json:"operation"`
}

func (RunJobArgs) Kind() string { return "run" }

func (RunJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:    "default",
		Priority: 1,
	}
}

type RunJobWorker struct {
	river.WorkerDefaults[RunJobArgs]
	queries     *repository.Queries
	executor    executor.Executor
	streamer    logstream.Streamer
	storage     *storage.S3Storage    // nil in dev without MinIO
	encryptor   *secrets.Encryptor    // nil if encryption not configured
	riverClient *river.Client[pgx.Tx]
	db          *pgxpool.Pool
}

// Timeout returns the maximum duration a run job can execute before River cancels it.
func (w *RunJobWorker) Timeout(*river.Job[RunJobArgs]) time.Duration {
	return 2 * time.Hour
}

func NewRunJobWorker(queries *repository.Queries, exec executor.Executor, streamer logstream.Streamer, store *storage.S3Storage, encryptor *secrets.Encryptor) *RunJobWorker {
	return &RunJobWorker{
		queries:   queries,
		executor:  exec,
		streamer:  streamer,
		storage:   store,
		encryptor: encryptor,
	}
}

func (w *RunJobWorker) SetRiverClient(client *river.Client[pgx.Tx], db *pgxpool.Pool) {
	w.riverClient = client
	w.db = db
}

func (w *RunJobWorker) Work(ctx context.Context, job *river.Job[RunJobArgs]) error {
	args := job.Args
	logger := slog.With("run_id", args.RunID, "workspace_id", args.WorkspaceID, "operation", args.Operation)
	logger.Info("starting run job")

	// Lock workspace
	if err := w.queries.SetWorkspaceCurrentRun(ctx, repository.SetWorkspaceCurrentRunParams{
		ID: args.WorkspaceID, OrgID: args.OrgID, CurrentRunID: &args.RunID,
	}); err != nil {
		return fmt.Errorf("failed to lock workspace: %w", err)
	}

	// Update run status
	status := "planning"
	if args.Operation == "apply" || args.Operation == "destroy" {
		status = "applying"
	}
	if _, err := w.queries.UpdateRunStarted(ctx, repository.UpdateRunStartedParams{ID: args.RunID, Status: status}); err != nil {
		return fmt.Errorf("failed to update run started: %w", err)
	}

	// Get workspace
	workspace, err := w.queries.GetWorkspace(ctx, repository.GetWorkspaceParams{ID: args.WorkspaceID, OrgID: args.OrgID})
	if err != nil {
		return w.failRun(ctx, args, logger, fmt.Errorf("failed to get workspace: %w", err), "")
	}

	// Load variables and decrypt sensitive ones
	vars, err := w.queries.ListWorkspaceVariables(ctx, repository.ListWorkspaceVariablesParams{
		WorkspaceID: args.WorkspaceID, OrgID: args.OrgID,
	})
	if err != nil {
		return w.failRun(ctx, args, logger, fmt.Errorf("failed to load variables: %w", err), "")
	}
	execVars := make([]executor.Variable, len(vars))
	for i, v := range vars {
		value := v.Value
		if v.Sensitive && w.encryptor != nil {
			decrypted, err := w.encryptor.Decrypt(v.Value)
			if err != nil {
				return w.failRun(ctx, args, logger, fmt.Errorf("failed to decrypt variable %q: %w", v.Key, err), "")
			}
			value = decrypted
		}
		execVars[i] = executor.Variable{Key: v.Key, Value: value, Category: v.Category}
	}

	// Fetch previous state from S3 for continuity
	var previousState []byte
	if w.storage != nil {
		latestSV, err := w.queries.GetLatestStateVersion(ctx, repository.GetLatestStateVersionParams{
			WorkspaceID: args.WorkspaceID, OrgID: args.OrgID,
		})
		if err == nil && latestSV.StateURL != "" {
			if stateData, err := w.storage.GetState(ctx, latestSV.StateURL); err == nil {
				previousState = stateData
				logger.Info("fetched previous state", "serial", latestSV.Serial, "size", len(stateData))
			} else {
				logger.Warn("failed to fetch previous state", "error", err)
			}
		}
	}

	// Derive state encryption passphrase if encryption is configured
	var stateEncPassphrase string
	if w.encryptor != nil {
		stateEncPassphrase = w.encryptor.DerivePassphrase("state:" + args.WorkspaceID)
	}

	// Collect log output for storage
	var logBuf strings.Builder
	logCallback := func(line []byte) {
		logBuf.Write(line)
		w.streamer.Publish(args.RunID, line)
	}

	// Execute
	result, err := w.executor.Execute(ctx, executor.ExecuteParams{
		RunID:                     args.RunID,
		WorkspaceID:               args.WorkspaceID,
		Operation:                 args.Operation,
		RepoURL:                   workspace.RepoURL,
		RepoBranch:                workspace.RepoBranch,
		WorkingDir:                workspace.WorkingDir,
		TofuVersion:               workspace.TofuVersion,
		Variables:                 execVars,
		LogCallback:               logCallback,
		PreviousState:             previousState,
		StateEncryptionPassphrase: stateEncPassphrase,
	})

	if err != nil {
		return w.failRun(ctx, args, logger, err, logBuf.String())
	}

	// Determine final status
	finalStatus := "planned"
	if args.Operation == "apply" || args.Operation == "destroy" {
		finalStatus = "applied"
	} else if args.Operation == "plan" {
		finalStatus = postPlanAction(workspace.AutoApply, workspace.RequiresApproval)
	}

	// Upload logs to S3
	if w.storage != nil {
		phase := args.Operation
		logURL, err := w.storage.PutLog(ctx, args.RunID, phase, []byte(logBuf.String()))
		if err != nil {
			logger.Error("failed to upload logs", "error", err)
		} else {
			planLog := &logURL
			var applyLog *string
			if args.Operation != "plan" {
				applyLog = planLog
				planLog = nil
			}
			if _, err := w.queries.UpdateRunLogURLs(ctx, repository.UpdateRunLogURLsParams{
				ID: args.RunID, PlanLogURL: planLog, ApplyLogURL: applyLog,
			}); err != nil {
				logger.Error("failed to update run log URLs", "error", err)
			}
		}
	}

	// Upload state file to S3 after apply/destroy
	if result.StateFile != nil && w.storage != nil {
		latestSV, _ := w.queries.GetLatestStateVersion(ctx, repository.GetLatestStateVersionParams{
			WorkspaceID: args.WorkspaceID, OrgID: args.OrgID,
		})
		nextSerial := latestSV.Serial + 1

		stateURL, err := w.storage.PutState(ctx, args.WorkspaceID, int(nextSerial), result.StateFile)
		if err != nil {
			logger.Error("failed to upload state", "error", err)
		} else {
			if _, err := w.queries.CreateStateVersion(ctx, repository.CreateStateVersionParams{
				ID:              ulid.Make().String(),
				WorkspaceID:     args.WorkspaceID,
				OrgID:           args.OrgID,
				RunID:           args.RunID,
				Serial:          nextSerial,
				StateURL:        stateURL,
				ResourceCount:   result.ResourcesAdded + result.ResourcesChanged,
				ResourceSummary: fmt.Sprintf("+%d ~%d -%d", result.ResourcesAdded, result.ResourcesChanged, result.ResourcesDeleted),
			}); err != nil {
				logger.Error("failed to create state version", "error", err)
			}
		}
	}

	// Check if run was cancelled while we were executing
	if w.isRunCancelled(ctx, args.RunID, args.OrgID) {
		logger.Info("run was cancelled during execution, skipping status update")
		w.streamer.Publish(args.RunID, []byte("\r\n\033[33mRun was cancelled\033[0m\r\n"))
		w.streamer.Close(args.RunID)
		if err := w.queries.SetWorkspaceCurrentRun(ctx, repository.SetWorkspaceCurrentRunParams{
			ID: args.WorkspaceID, OrgID: args.OrgID, CurrentRunID: nil,
		}); err != nil {
			logger.Error("failed to unlock workspace after cancel", "error", err)
		}
		w.enqueueNextPendingRun(ctx, args.WorkspaceID, logger)
		return nil
	}

	// Update run as finished — return the error so River can retry if DB fails
	if _, err := w.queries.UpdateRunFinished(ctx, repository.UpdateRunFinishedParams{
		ID:               args.RunID,
		Status:           finalStatus,
		PlanOutput:       &result.Output,
		ResourcesAdded:   &result.ResourcesAdded,
		ResourcesChanged: &result.ResourcesChanged,
		ResourcesDeleted: &result.ResourcesDeleted,
	}); err != nil {
		return fmt.Errorf("failed to update run finished: %w", err)
	}

	w.streamer.Publish(args.RunID, []byte(fmt.Sprintf("\r\n\033[32mRun completed successfully at %s\033[0m\r\n", time.Now().Format(time.RFC3339))))
	w.streamer.Close(args.RunID)

	// Auto-apply: enqueue apply job immediately instead of unlocking
	if finalStatus == "queued" && w.riverClient != nil && w.db != nil {
		tx, txErr := w.db.Begin(ctx)
		if txErr == nil {
			_, insErr := w.riverClient.InsertTx(ctx, tx, RunJobArgs{
				RunID:       args.RunID,
				WorkspaceID: args.WorkspaceID,
				OrgID:       args.OrgID,
				Operation:   "apply",
			}, nil)
			if insErr == nil {
				tx.Commit(ctx)
				logger.Info("auto-apply enqueued", "run_id", args.RunID)
				return nil
			}
			tx.Rollback(ctx)
		}
	}

	// Unlock workspace and pick up next queued run
	if err := w.queries.SetWorkspaceCurrentRun(ctx, repository.SetWorkspaceCurrentRunParams{
		ID: args.WorkspaceID, OrgID: args.OrgID, CurrentRunID: nil,
	}); err != nil {
		logger.Error("failed to unlock workspace", "error", err)
	}

	w.enqueueNextPendingRun(ctx, args.WorkspaceID, logger)

	logger.Info("run completed", "status", finalStatus)
	return nil
}

func (w *RunJobWorker) failRun(ctx context.Context, args RunJobArgs, logger *slog.Logger, runErr error, logOutput string) error {
	logger.Error("run failed", "error", runErr)

	// Don't overwrite cancelled status
	if !w.isRunCancelled(ctx, args.RunID, args.OrgID) {
		errMsg := runErr.Error()
		var planOutput *string
		if logOutput != "" {
			planOutput = &logOutput
		}
		if _, dbErr := w.queries.UpdateRunFinished(ctx, repository.UpdateRunFinishedParams{
			ID: args.RunID, Status: "errored", ErrorMessage: &errMsg, PlanOutput: planOutput,
		}); dbErr != nil {
			// Return the DB error so River retries — the run would be stuck otherwise
			return fmt.Errorf("failed to mark run as errored (original error: %v): %w", runErr, dbErr)
		}
		w.streamer.Publish(args.RunID, []byte(fmt.Sprintf("\r\n\033[31mRun failed: %s\033[0m\r\n", runErr.Error())))
	} else {
		logger.Info("run was cancelled, not overwriting with errored status")
		w.streamer.Publish(args.RunID, []byte("\r\n\033[33mRun was cancelled\033[0m\r\n"))
	}
	w.streamer.Close(args.RunID)

	// Unlock workspace
	if err := w.queries.SetWorkspaceCurrentRun(ctx, repository.SetWorkspaceCurrentRunParams{
		ID: args.WorkspaceID, OrgID: args.OrgID, CurrentRunID: nil,
	}); err != nil {
		logger.Error("failed to unlock workspace after failure", "error", err)
	}

	w.enqueueNextPendingRun(ctx, args.WorkspaceID, logger)
	return nil
}

// isRunCancelled checks if the run status was set to cancelled (e.g. via the API)
// while the worker was executing. Returns true if the run should not have its status overwritten.
func (w *RunJobWorker) isRunCancelled(ctx context.Context, runID, orgID string) bool {
	currentRun, err := w.queries.GetRun(ctx, repository.GetRunParams{ID: runID, OrgID: orgID})
	if err != nil {
		return false
	}
	return currentRun.Status == "cancelled"
}

// postPlanAction determines the status after a plan completes.
// auto_apply wins over requires_approval. "queued" triggers auto-apply enqueue.
func postPlanAction(autoApply, requiresApproval bool) string {
	if autoApply {
		return "queued"
	}
	if requiresApproval {
		return "awaiting_approval"
	}
	return "planned"
}

func (w *RunJobWorker) enqueueNextPendingRun(ctx context.Context, workspaceID string, logger *slog.Logger) {
	if w.riverClient == nil || w.db == nil {
		return
	}

	nextRun, err := w.queries.GetNextPendingRun(ctx, workspaceID)
	if err != nil {
		return // no pending runs
	}

	tx, err := w.db.Begin(ctx)
	if err != nil {
		logger.Error("failed to begin tx for next pending run", "error", err)
		return
	}
	defer tx.Rollback(ctx)

	_, err = w.riverClient.InsertTx(ctx, tx, RunJobArgs{
		RunID:       nextRun.ID,
		WorkspaceID: nextRun.WorkspaceID,
		OrgID:       nextRun.OrgID,
		Operation:   nextRun.Operation,
	}, nil)
	if err != nil {
		logger.Error("failed to enqueue next pending run", "error", err, "run_id", nextRun.ID)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error("failed to commit next pending run", "error", err, "run_id", nextRun.ID)
		return
	}

	logger.Info("enqueued next pending run", "run_id", nextRun.ID)
}
