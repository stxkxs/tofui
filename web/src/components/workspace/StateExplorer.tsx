import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/api/client";
import type { StateVersion } from "@/api/types";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import { formatRelativeTime } from "@/lib/utils";
import { Database, Download, Layers } from "lucide-react";

interface Props {
  workspaceId: string;
}

async function downloadState(workspaceId: string, stateId: string) {
  const token = localStorage.getItem("tofui_token");
  const res = await fetch(`/api/v1/workspaces/${workspaceId}/state/${stateId}/download`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) {
    toast.error("Failed to download state file");
    return;
  }
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `state-${stateId}.json`;
  a.click();
  URL.revokeObjectURL(url);
}

export function StateExplorer({ workspaceId }: Props) {
  const { data: versions, isLoading, isError } = useQuery({
    queryKey: ["state-versions", workspaceId],
    queryFn: async () => {
      const { data, error } = await api.GET(
        "/workspaces/{workspaceId}/state",
        {
          params: { path: { workspaceId } },
        }
      );
      if (error) throw error;
      return data;
    },
  });

  const { data: currentState } = useQuery({
    queryKey: ["state-current", workspaceId],
    queryFn: async () => {
      const { data, error } = await api.GET(
        "/workspaces/{workspaceId}/state/current",
        {
          params: { path: { workspaceId } },
        }
      );
      if (error) throw error;
      return data;
    },
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Spinner className="w-5 h-5" />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="rounded-lg border border-destructive/20 bg-destructive/5 p-8 text-center">
        <p className="text-sm text-destructive">Failed to load state versions.</p>
      </div>
    );
  }

  return (
    <div>
      <h3 className="text-base font-semibold mb-4">State Versions</h3>

      {/* Current state summary */}
      {currentState && (
        <div className="mb-4 p-4 rounded-lg border border-primary/20 bg-primary/5">
          <div className="flex items-center gap-3 mb-2">
            <Database className="w-4 h-4 text-primary" />
            <span className="text-sm font-medium">Current State</span>
            <Badge variant="outline">Serial #{currentState.serial}</Badge>
          </div>
          <div className="flex items-center gap-4 text-sm text-muted-foreground">
            <span className="flex items-center gap-1.5">
              <Layers className="w-3.5 h-3.5" />
              {currentState.resource_count} resources
            </span>
            <span className="font-mono text-xs">
              {currentState.resource_summary}
            </span>
            <span>{formatRelativeTime(currentState.created_at)}</span>
          </div>
        </div>
      )}

      {!versions?.length ? (
        <div className="rounded-lg border border-dashed border-border p-8 text-center">
          <Database className="w-8 h-8 text-muted-foreground mx-auto mb-2" />
          <p className="text-sm text-muted-foreground">
            No state versions yet. Run an apply to create the first state.
          </p>
        </div>
      ) : (
        <div className="rounded-lg border border-border divide-y divide-border">
          {(versions as StateVersion[]).map((sv) => (
            <div
              key={sv.id}
              className="flex items-center justify-between px-4 py-3 hover:bg-accent/50 transition-colors"
            >
              <div className="flex items-center gap-3">
                <span className="font-mono text-sm font-medium">
                  #{sv.serial}
                </span>
                <span className="text-sm text-muted-foreground">
                  {sv.resource_count} resources
                </span>
                <span className="font-mono text-xs text-muted-foreground">
                  {sv.resource_summary}
                </span>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-xs text-muted-foreground">
                  {formatRelativeTime(sv.created_at)}
                </span>
                <button
                  onClick={() => downloadState(workspaceId, sv.id)}
                  className="p-1.5 rounded hover:bg-accent text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                  title="Download state file"
                >
                  <Download className="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
