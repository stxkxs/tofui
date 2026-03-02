import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/api/client";
import type { Workspace, CreateWorkspaceRequest } from "@/api/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import { CreateWorkspaceDialog } from "./CreateWorkspaceDialog";
import { formatRelativeTime, getEnvironmentColor } from "@/lib/utils";
import {
  Plus,
  GitBranch,
  FolderGit2,
  Clock,
  Lock,
  Zap,
  ShieldCheck,
  Webhook,
} from "lucide-react";

export function WorkspaceList() {
  const [showCreate, setShowCreate] = useState(false);
  const queryClient = useQueryClient();

  const { data, isLoading, isError } = useQuery({
    queryKey: ["workspaces"],
    queryFn: async () => {
      const { data, error } = await api.GET("/workspaces", {
        params: { query: { per_page: 50 } },
      });
      if (error) throw error;
      return data;
    },
  });

  const createMutation = useMutation({
    mutationFn: async (params: CreateWorkspaceRequest) => {
      const { data, error } = await api.POST("/workspaces", {
        body: params,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["workspaces"] });
      setShowCreate(false);
      toast.success("Workspace created");
    },
    onError: () => toast.error("Failed to create workspace"),
  });

  return (
    <div className="p-8 max-w-6xl">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Workspaces</h1>
          <p className="text-muted-foreground mt-1">
            Manage your OpenTofu workspaces
          </p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus className="w-4 h-4" />
          New workspace
        </Button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-20">
          <Spinner className="w-6 h-6" />
        </div>
      ) : isError ? (
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-10 text-center">
          <p className="text-sm text-destructive">Failed to load workspaces. Please try again.</p>
        </div>
      ) : !data?.data?.length ? (
        <div className="rounded-xl border border-dashed border-border p-12 text-center">
          <FolderGit2 className="w-12 h-12 text-muted-foreground mx-auto mb-4" />
          <h3 className="text-lg font-medium mb-2">No workspaces yet</h3>
          <p className="text-muted-foreground mb-6 max-w-sm mx-auto">
            Create your first workspace to start managing OpenTofu infrastructure.
          </p>
          <Button onClick={() => setShowCreate(true)}>
            <Plus className="w-4 h-4" />
            Create workspace
          </Button>
        </div>
      ) : (
        <div className="grid gap-3" role="list" aria-label="Workspaces">
          {data.data.map((workspace: Workspace) => (
            <a
              key={workspace.id}
              href={`/workspaces/${workspace.id}`}
              role="listitem"
              aria-label={`Workspace ${workspace.name}, ${workspace.environment}`}
              className="group block rounded-xl border border-border bg-card p-5 transition-all hover:border-primary/30 hover:shadow-lg hover:shadow-primary/5"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-3 mb-2">
                    <h3 className="font-semibold text-base group-hover:text-primary transition-colors">
                      {workspace.name}
                    </h3>
                    <Badge
                      className={getEnvironmentColor(workspace.environment)}
                      variant="outline"
                    >
                      {workspace.environment}
                    </Badge>
                    {workspace.auto_apply && (
                      <Badge variant="outline" className="text-xs py-0 px-1.5 text-blue-400 border-blue-400/30">
                        <Zap className="w-3 h-3 mr-0.5" />
                        auto
                      </Badge>
                    )}
                    {workspace.requires_approval && (
                      <Badge variant="outline" className="text-xs py-0 px-1.5 text-amber-400 border-amber-400/30">
                        <ShieldCheck className="w-3 h-3 mr-0.5" />
                        approval
                      </Badge>
                    )}
                    {workspace.vcs_trigger_enabled && (
                      <Badge variant="outline" className="text-xs py-0 px-1.5 text-violet-400 border-violet-400/30">
                        <Webhook className="w-3 h-3 mr-0.5" />
                        vcs
                      </Badge>
                    )}
                    {workspace.locked && (
                      <span aria-label="Locked">
                        <Lock className="w-3.5 h-3.5 text-warning" aria-hidden="true" />
                      </span>
                    )}
                  </div>
                  {workspace.description && (
                    <p className="text-sm text-muted-foreground mb-3 line-clamp-1">
                      {workspace.description}
                    </p>
                  )}
                  <div className="flex items-center gap-4 text-xs text-muted-foreground">
                    <span className="flex items-center gap-1.5">
                      <GitBranch className="w-3.5 h-3.5" />
                      {workspace.repo_branch}
                    </span>
                    <span className="flex items-center gap-1.5">
                      <span className="font-mono">tofu {workspace.tofu_version}</span>
                    </span>
                    <span className="flex items-center gap-1.5">
                      <Clock className="w-3.5 h-3.5" />
                      {formatRelativeTime(workspace.updated_at)}
                    </span>
                  </div>
                </div>
              </div>
            </a>
          ))}
        </div>
      )}

      <CreateWorkspaceDialog
        open={showCreate}
        onClose={() => setShowCreate(false)}
        onSubmit={(data) => createMutation.mutate(data)}
        isLoading={createMutation.isPending}
      />
    </div>
  );
}
