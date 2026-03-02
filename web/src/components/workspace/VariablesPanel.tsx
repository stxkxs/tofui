import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/api/client";
import type { WorkspaceVariable, DiscoveredVariable } from "@/api/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Plus, Trash2, Lock, Search, X, Check } from "lucide-react";

interface Props {
  workspaceId: string;
}

export function VariablesPanel({ workspaceId }: Props) {
  const queryClient = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");
  const [newSensitive, setNewSensitive] = useState(false);
  const [newCategory, setNewCategory] = useState<"terraform" | "env">(
    "terraform"
  );
  const [deleteTarget, setDeleteTarget] = useState<WorkspaceVariable | null>(null);
  const [discoveredVars, setDiscoveredVars] = useState<DiscoveredVariable[] | null>(null);

  const { data: variables, isLoading, isError } = useQuery({
    queryKey: ["variables", workspaceId],
    queryFn: async () => {
      const { data, error } = await api.GET(
        "/workspaces/{workspaceId}/variables",
        {
          params: { path: { workspaceId } },
        }
      );
      if (error) throw error;
      return data;
    },
  });

  const createMutation = useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST(
        "/workspaces/{workspaceId}/variables",
        {
          params: { path: { workspaceId } },
          body: {
            key: newKey,
            value: newValue,
            sensitive: newSensitive,
            category: newCategory,
          },
        }
      );
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["variables", workspaceId] });
      setNewKey("");
      setNewValue("");
      setNewSensitive(false);
      setShowForm(false);
      toast.success("Variable created");
      // Re-run discover to update configured status
      if (discoveredVars) {
        discoverMutation.mutate();
      }
    },
    onError: () => toast.error("Failed to create variable"),
  });

  const deleteMutation = useMutation({
    mutationFn: async (variableId: string) => {
      const { error } = await api.DELETE(
        "/workspaces/{workspaceId}/variables/{variableId}",
        {
          params: { path: { workspaceId, variableId } },
        }
      );
      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["variables", workspaceId] });
      setDeleteTarget(null);
      toast.success("Variable deleted");
    },
    onError: () => toast.error("Failed to delete variable"),
  });

  const discoverMutation = useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST(
        "/workspaces/{workspaceId}/variables/discover",
        {
          params: { path: { workspaceId } },
        }
      );
      if (error) throw error;
      return data;
    },
    onSuccess: (data) => {
      setDiscoveredVars(data ?? []);
    },
    onError: () => toast.error("Failed to discover variables"),
  });

  const handleAddDiscovered = (v: DiscoveredVariable) => {
    setNewKey(v.name);
    setNewValue(v.default ?? "");
    setNewCategory("terraform");
    setNewSensitive(false);
    setShowForm(true);
  };

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
        <p className="text-sm text-destructive">Failed to load variables.</p>
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-base font-semibold">Variables</h3>
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            variant="outline"
            onClick={() => discoverMutation.mutate()}
            disabled={discoverMutation.isPending}
          >
            {discoverMutation.isPending ? (
              <Spinner className="w-3.5 h-3.5" />
            ) : (
              <Search className="w-3.5 h-3.5" />
            )}
            Discover Variables
          </Button>
          <Button size="sm" variant="outline" onClick={() => setShowForm(!showForm)}>
            <Plus className="w-3.5 h-3.5" />
            Add variable
          </Button>
        </div>
      </div>

      {discoveredVars && (
        <div className="mb-4 rounded-lg border border-border bg-card">
          <div className="flex items-center justify-between px-4 py-3 border-b border-border">
            <div className="flex items-center gap-2">
              <Search className="w-4 h-4 text-muted-foreground" />
              <span className="text-sm font-medium">
                Discovered Variables ({discoveredVars.length})
              </span>
            </div>
            <button
              onClick={() => setDiscoveredVars(null)}
              className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
          {discoveredVars.length === 0 ? (
            <div className="px-4 py-6 text-center">
              <p className="text-sm text-muted-foreground">
                No variable blocks found in the repository.
              </p>
            </div>
          ) : (
            <div className="divide-y divide-border">
              {discoveredVars.map((v) => (
                <div
                  key={v.name}
                  className="flex items-center justify-between px-4 py-3"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <code className="text-sm font-mono font-medium">{v.name}</code>
                    {v.type && (
                      <Badge variant="outline" className="text-xs shrink-0">
                        {v.type}
                      </Badge>
                    )}
                    {v.configured ? (
                      <Badge className="text-xs bg-emerald-500/10 text-emerald-600 border-emerald-500/20 shrink-0">
                        <Check className="w-3 h-3 mr-1" />
                        configured
                      </Badge>
                    ) : v.required ? (
                      <Badge className="text-xs bg-red-500/10 text-red-600 border-red-500/20 shrink-0">
                        required
                      </Badge>
                    ) : (
                      <Badge className="text-xs bg-amber-500/10 text-amber-600 border-amber-500/20 shrink-0">
                        optional
                      </Badge>
                    )}
                  </div>
                  <div className="flex items-center gap-3 shrink-0">
                    {v.description && (
                      <span className="text-xs text-muted-foreground max-w-[200px] truncate hidden sm:inline">
                        {v.description}
                      </span>
                    )}
                    {v.default !== undefined && (
                      <span className="text-xs font-mono text-muted-foreground max-w-[120px] truncate">
                        ={v.default}
                      </span>
                    )}
                    {!v.configured && (
                      <Button
                        size="sm"
                        variant="outline"
                        className="text-xs h-7"
                        onClick={() => handleAddDiscovered(v)}
                      >
                        <Plus className="w-3 h-3" />
                        Add
                      </Button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {showForm && (
        <div className="mb-4 p-4 rounded-lg border border-border bg-card space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <Input
              placeholder="Variable key"
              value={newKey}
              onChange={(e) => setNewKey(e.target.value)}
            />
            <Input
              placeholder="Value"
              type={newSensitive ? "password" : "text"}
              value={newValue}
              onChange={(e) => setNewValue(e.target.value)}
            />
          </div>
          <div className="flex items-center gap-4">
            <Select
              value={newCategory}
              onChange={(e) =>
                setNewCategory(e.target.value as "terraform" | "env")
              }
            >
              <option value="terraform">OpenTofu</option>
              <option value="env">Environment</option>
            </Select>
            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={newSensitive}
                onChange={(e) => setNewSensitive(e.target.checked)}
                className="rounded border-border"
              />
              Sensitive
            </label>
            <div className="flex-1" />
            <Button
              size="sm"
              variant="ghost"
              onClick={() => setShowForm(false)}
            >
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={() => createMutation.mutate()}
              disabled={!newKey || createMutation.isPending}
            >
              {createMutation.isPending ? <Spinner /> : "Save"}
            </Button>
          </div>
        </div>
      )}

      {!variables?.length ? (
        <div className="rounded-lg border border-dashed border-border p-8 text-center">
          <p className="text-sm text-muted-foreground">
            No variables configured yet.
          </p>
        </div>
      ) : (
        <div className="rounded-lg border border-border divide-y divide-border">
          {variables.map((v: WorkspaceVariable) => (
            <div
              key={v.id}
              className="flex items-center justify-between px-4 py-3"
            >
              <div className="flex items-center gap-3">
                <code className="text-sm font-mono font-medium">{v.key}</code>
                <Badge variant="outline" className="text-xs">
                  {v.category}
                </Badge>
                {v.sensitive && (
                  <Lock className="w-3.5 h-3.5 text-muted-foreground" />
                )}
              </div>
              <div className="flex items-center gap-2">
                <span className="text-sm font-mono text-muted-foreground max-w-[200px] truncate">
                  {v.sensitive ? "***" : v.value}
                </span>
                <button
                  onClick={() => setDeleteTarget(v)}
                  className="p-1 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors cursor-pointer"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete variable</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete{" "}
              <code className="font-mono font-medium">{deleteTarget?.key}</code>?
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <div className="flex justify-end gap-2 mt-4">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setDeleteTarget(null)}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => {
                if (deleteTarget) deleteMutation.mutate(deleteTarget.id);
              }}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? <Spinner /> : "Delete"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
