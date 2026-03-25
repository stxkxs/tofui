import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/api/client";
import type { PipelineRunStage } from "@/api/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import { Link } from "@/components/ui/link";
import { formatDuration } from "@/lib/utils";
import {
  ArrowLeft,
  CheckCircle2,
  XCircle,
  Clock,
  Ban,
  Loader2,
  Import,
  Pause,
  SkipForward,
  ExternalLink,
} from "lucide-react";

function stageStatusIcon(status: string) {
  const base = "w-[18px] h-[18px]";
  switch (status) {
    case "completed":
      return <CheckCircle2 className={`${base} text-success`} />;
    case "errored":
      return <XCircle className={`${base} text-destructive`} />;
    case "running":
      return <Loader2 className={`${base} text-primary animate-spin`} />;
    case "importing_outputs":
      return <Import className={`${base} text-primary animate-pulse`} />;
    case "awaiting_approval":
      return <Pause className={`${base} text-warning`} />;
    case "cancelled":
      return <Ban className={`${base} text-muted-foreground/60`} />;
    case "skipped":
      return <SkipForward className={`${base} text-muted-foreground/60`} />;
    default:
      return <Clock className={`${base} text-muted-foreground/40`} />;
  }
}

function stageStatusBadge(status: string) {
  switch (status) {
    case "completed":
      return <Badge variant="success">Completed</Badge>;
    case "errored":
      return <Badge variant="destructive">Errored</Badge>;
    case "running":
      return <Badge variant="default">Running</Badge>;
    case "importing_outputs":
      return <Badge variant="default">Importing</Badge>;
    case "awaiting_approval":
      return <Badge variant="warning">Awaiting Approval</Badge>;
    case "cancelled":
      return <Badge variant="secondary">Cancelled</Badge>;
    case "skipped":
      return <Badge variant="secondary">Skipped</Badge>;
    default:
      return <Badge variant="secondary">Pending</Badge>;
  }
}

function pipelineRunStatusBadge(status: string) {
  switch (status) {
    case "running":
      return <Badge variant="default">Running</Badge>;
    case "completed":
      return <Badge variant="success">Completed</Badge>;
    case "errored":
      return <Badge variant="destructive">Errored</Badge>;
    case "cancelled":
      return <Badge variant="warning">Cancelled</Badge>;
    default:
      return <Badge variant="secondary">{status}</Badge>;
  }
}

function progressBarColor(status: string) {
  switch (status) {
    case "completed":
      return "bg-success";
    case "running":
    case "importing_outputs":
      return "bg-primary animate-shimmer";
    case "errored":
      return "bg-destructive";
    case "awaiting_approval":
      return "bg-warning";
    case "cancelled":
      return "bg-muted-foreground/20";
    default:
      return "bg-border/40";
  }
}

function stageBorderStyle(status: string) {
  switch (status) {
    case "running":
    case "importing_outputs":
      return "border-primary/30 bg-primary/[0.03]";
    case "completed":
      return "border-success/20";
    case "errored":
      return "border-destructive/20";
    case "awaiting_approval":
      return "border-warning/20";
    default:
      return "border-border/50";
  }
}

export function PipelineRunView({
  pipelineId,
  runId,
}: {
  pipelineId: string;
  runId: string;
}) {
  const queryClient = useQueryClient();

  const { data, isLoading, isError } = useQuery({
    queryKey: ["pipeline-run", pipelineId, runId],
    queryFn: async () => {
      const { data, error } = await api.GET(
        "/pipelines/{pipelineId}/runs/{runId}",
        { params: { path: { pipelineId, runId } } }
      );
      if (error) throw error;
      return data!;
    },
    refetchInterval: (query) => {
      const pr = query.state.data?.pipeline_run;
      if (pr && pr.status === "running") return 3000;
      return false;
    },
  });

  const cancelMutation = useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST(
        "/pipelines/{pipelineId}/runs/{runId}/cancel",
        { params: { path: { pipelineId, runId } } }
      );
      if (error) throw error;
      return data!;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["pipeline-run", pipelineId, runId],
      });
      toast.success("Pipeline run cancelled");
    },
    onError: () => toast.error("Failed to cancel pipeline run"),
  });

  if (isLoading) {
    return (
      <div className="flex justify-center py-20">
        <Spinner className="w-6 h-6 text-primary" />
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="p-6">
        <div className="bg-destructive/8 text-destructive border border-destructive/15 rounded-lg p-4 text-sm">
          Failed to load pipeline run.
        </div>
      </div>
    );
  }

  const { pipeline_run: pr, stages } = data;
  const isRunning = pr.status === "running";

  return (
    <div className="p-6 animate-fade-up">
      {/* Header */}
      <div className="mb-6">
        <Link
          href={`/pipelines/${pipelineId}`}
          className="text-xs text-muted-foreground hover:text-foreground inline-flex items-center gap-1 mb-3 transition-colors"
        >
          <ArrowLeft className="w-3 h-3" />
          Pipeline
        </Link>

        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <h1 className="text-lg font-semibold tracking-tight">
              Pipeline Run
            </h1>
            {pipelineRunStatusBadge(pr.status)}
          </div>
          <div className="flex items-center gap-3">
            <span className="text-xs text-muted-foreground font-mono">
              {formatDuration(pr.started_at, pr.finished_at)}
            </span>
            {isRunning && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => cancelMutation.mutate()}
                disabled={cancelMutation.isPending}
                className="text-destructive hover:text-destructive hover:bg-destructive/10 hover:border-destructive/30"
              >
                <Ban className="w-3 h-3" />
                Cancel
              </Button>
            )}
          </div>
        </div>

        <div className="text-[11px] text-muted-foreground/70 mt-1.5 flex items-center gap-1.5">
          <span>
            Stage {pr.current_stage + 1} of {pr.total_stages}
          </span>
          <span className="text-border">|</span>
          <span>Started {new Date(pr.started_at).toLocaleString()}</span>
          {pr.finished_at && (
            <>
              <span className="text-border">|</span>
              <span>
                Finished {new Date(pr.finished_at).toLocaleString()}
              </span>
            </>
          )}
        </div>
      </div>

      {/* Progress bar */}
      <div className="mb-8">
        <div className="flex gap-1 h-1.5 rounded-full overflow-hidden bg-border/20">
          {stages.map((stage: PipelineRunStage) => (
            <div
              key={stage.id}
              className={`flex-1 rounded-full transition-all duration-500 ${progressBarColor(stage.status)}`}
            />
          ))}
        </div>
      </div>

      {/* Stage cards */}
      <div className="relative">
        {/* Vertical connector */}
        {stages.length > 1 && (
          <div
            className="absolute left-[19px] top-[40px] w-px bg-border/40"
            style={{ height: `calc(100% - 56px)` }}
          />
        )}
        <div className="space-y-2">
          {stages.map((stage: PipelineRunStage, i: number) => {
            const isActive =
              stage.status === "running" ||
              stage.status === "importing_outputs";
            return (
              <div
                key={stage.id}
                className={`relative flex items-start gap-3 animate-fade-up ${isActive ? "animate-glow-pulse" : ""}`}
                style={{ animationDelay: `${i * 50}ms` }}
              >
                {/* Status dot */}
                <div
                  className={`w-10 h-10 rounded-full border-2 bg-card flex items-center justify-center z-10 shrink-0 transition-all duration-300 ${
                    stage.status === "completed"
                      ? "border-success/40"
                      : isActive
                      ? "border-primary/50"
                      : stage.status === "errored"
                      ? "border-destructive/40"
                      : "border-border/50"
                  }`}
                >
                  {stageStatusIcon(stage.status)}
                </div>

                {/* Card */}
                <div
                  className={`flex-1 border rounded-lg px-4 py-3 transition-all duration-200 ${stageBorderStyle(stage.status)}`}
                >
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-medium">
                          {stage.workspace_name}
                        </span>
                        {stageStatusBadge(stage.status)}
                      </div>
                      {stage.started_at && (
                        <div className="text-[11px] text-muted-foreground/60 mt-1 font-mono">
                          {formatDuration(stage.started_at, stage.finished_at)}
                        </div>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge
                        variant={stage.auto_apply ? "success" : "secondary"}
                      >
                        {stage.auto_apply ? "auto" : "manual"}
                      </Badge>
                      {stage.run_id && (
                        <Link
                          href={`/workspaces/${stage.workspace_id}/runs/${stage.run_id}`}
                          className="inline-flex items-center gap-1 text-[11px] text-primary/80 hover:text-primary transition-colors"
                        >
                          View Run
                          <ExternalLink className="w-2.5 h-2.5" />
                        </Link>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
