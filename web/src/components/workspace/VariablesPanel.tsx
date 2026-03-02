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
import { Plus, Trash2, Lock, Search, X, Check, Pencil, Eye, EyeOff, Upload } from "lucide-react";

interface Props {
  workspaceId: string;
}

function parseEnvFormat(text: string): { key: string; value: string }[] {
  const result: { key: string; value: string }[] = [];
  for (const line of text.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;
    const eqIdx = trimmed.indexOf("=");
    if (eqIdx <= 0) continue;
    const key = trimmed.slice(0, eqIdx).trim();
    let value = trimmed.slice(eqIdx + 1).trim();
    // Strip surrounding quotes
    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
      value = value.slice(1, -1);
    }
    result.push({ key, value });
  }
  return result;
}

export function VariablesPanel({ workspaceId }: Props) {
  const queryClient = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");
  const [newSensitive, setNewSensitive] = useState(false);
  const [newCategory, setNewCategory] = useState<"terraform" | "env">("terraform");
  const [newDescription, setNewDescription] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<WorkspaceVariable | null>(null);
  const [discoveredVars, setDiscoveredVars] = useState<DiscoveredVariable[] | null>(null);

  // Inline edit state
  const [editTarget, setEditTarget] = useState<string | null>(null);
  const [editValue, setEditValue] = useState("");
  const [editSensitive, setEditSensitive] = useState(false);
  const [editDescription, setEditDescription] = useState("");

  // Reveal state
  const [revealedValues, setRevealedValues] = useState<Record<string, string>>({});

  // Bulk import state
  const [showBulkImport, setShowBulkImport] = useState(false);
  const [bulkText, setBulkText] = useState("");
  const [bulkCategory, setBulkCategory] = useState<"terraform" | "env">("terraform");
  const [bulkSensitive, setBulkSensitive] = useState(false);
  const [bulkParsed, setBulkParsed] = useState<{ key: string; value: string }[] | null>(null);

  const { data: variables, isLoading, isError } = useQuery({
    queryKey: ["variables", workspaceId],
    queryFn: async () => {
      const { data, error } = await api.GET(
        "/workspaces/{workspaceId}/variables",
        { params: { path: { workspaceId } } }
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
          body: { key: newKey, value: newValue, sensitive: newSensitive, category: newCategory, description: newDescription },
        }
      );
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["variables", workspaceId] });
      setNewKey(""); setNewValue(""); setNewSensitive(false); setNewDescription("");
      setShowForm(false);
      toast.success("Variable created");
      if (discoveredVars) discoverMutation.mutate();
    },
    onError: () => toast.error("Failed to create variable"),
  });

  const updateMutation = useMutation({
    mutationFn: async (variableId: string) => {
      const { data, error } = await api.PUT(
        "/workspaces/{workspaceId}/variables/{variableId}",
        {
          params: { path: { workspaceId, variableId } },
          body: {
            key: variables?.find((v: WorkspaceVariable) => v.id === variableId)?.key ?? "",
            value: editValue,
            sensitive: editSensitive,
            category: variables?.find((v: WorkspaceVariable) => v.id === variableId)?.category ?? "terraform",
            description: editDescription,
          },
        }
      );
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["variables", workspaceId] });
      setEditTarget(null);
      toast.success("Variable updated");
    },
    onError: () => toast.error("Failed to update variable"),
  });

  const deleteMutation = useMutation({
    mutationFn: async (variableId: string) => {
      const { error } = await api.DELETE(
        "/workspaces/{workspaceId}/variables/{variableId}",
        { params: { path: { workspaceId, variableId } } }
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
        { params: { path: { workspaceId } } }
      );
      if (error) throw error;
      return data;
    },
    onSuccess: (data) => setDiscoveredVars(data ?? []),
    onError: () => toast.error("Failed to discover variables"),
  });

  const bulkCreateMutation = useMutation({
    mutationFn: async (vars: { key: string; value: string }[]) => {
      const { data, error } = await api.POST(
        "/workspaces/{workspaceId}/variables/bulk",
        {
          params: { path: { workspaceId } },
          body: {
            variables: vars.map((v) => ({
              key: v.key,
              value: v.value,
              sensitive: bulkSensitive,
              category: bulkCategory,
            })),
          },
        }
      );
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["variables", workspaceId] });
      setShowBulkImport(false);
      setBulkText(""); setBulkParsed(null);
      toast.success("Variables imported");
    },
    onError: () => toast.error("Failed to import variables"),
  });

  const handleAddDiscovered = (v: DiscoveredVariable) => {
    setNewKey(v.name); setNewValue(v.default ?? "");
    setNewCategory("terraform"); setNewSensitive(false);
    setNewDescription(v.description ?? "");
    setShowForm(true);
  };

  const startEdit = (v: WorkspaceVariable) => {
    setEditTarget(v.id);
    setEditValue(v.sensitive ? "" : v.value);
    setEditSensitive(v.sensitive);
    setEditDescription(v.description);
  };

  const toggleReveal = async (v: WorkspaceVariable) => {
    if (revealedValues[v.id]) {
      setRevealedValues((prev) => { const n = { ...prev }; delete n[v.id]; return n; });
      return;
    }
    try {
      const { data, error } = await api.GET(
        "/workspaces/{workspaceId}/variables/{variableId}/value",
        { params: { path: { workspaceId, variableId: v.id } } }
      );
      if (error) throw error;
      setRevealedValues((prev) => ({ ...prev, [v.id]: data.value }));
    } catch {
      toast.error("Failed to reveal variable value");
    }
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
          <Button size="sm" variant="outline" onClick={() => discoverMutation.mutate()} disabled={discoverMutation.isPending}>
            {discoverMutation.isPending ? <Spinner className="w-3.5 h-3.5" /> : <Search className="w-3.5 h-3.5" />}
            Discover
          </Button>
          <Button size="sm" variant="outline" onClick={() => setShowBulkImport(true)}>
            <Upload className="w-3.5 h-3.5" />
            Bulk Import
          </Button>
          <Button size="sm" variant="outline" onClick={() => setShowForm(!showForm)}>
            <Plus className="w-3.5 h-3.5" />
            Add variable
          </Button>
        </div>
      </div>

      {/* Discovered variables panel */}
      {discoveredVars && (
        <div className="mb-4 rounded-lg border border-border bg-card">
          <div className="flex items-center justify-between px-4 py-3 border-b border-border">
            <div className="flex items-center gap-2">
              <Search className="w-4 h-4 text-muted-foreground" />
              <span className="text-sm font-medium">Discovered Variables ({discoveredVars.length})</span>
            </div>
            <button onClick={() => setDiscoveredVars(null)} className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors cursor-pointer">
              <X className="w-4 h-4" />
            </button>
          </div>
          {discoveredVars.length === 0 ? (
            <div className="px-4 py-6 text-center">
              <p className="text-sm text-muted-foreground">No variable blocks found in the repository.</p>
            </div>
          ) : (
            <div className="divide-y divide-border">
              {discoveredVars.map((v) => (
                <div key={v.name} className="flex items-center justify-between px-4 py-3">
                  <div className="flex items-center gap-3 min-w-0">
                    <code className="text-sm font-mono font-medium">{v.name}</code>
                    {v.type && <Badge variant="outline" className="text-xs shrink-0">{v.type}</Badge>}
                    {v.configured ? (
                      <Badge className="text-xs bg-emerald-500/10 text-emerald-600 border-emerald-500/20 shrink-0"><Check className="w-3 h-3 mr-1" />configured</Badge>
                    ) : v.required ? (
                      <Badge className="text-xs bg-red-500/10 text-red-600 border-red-500/20 shrink-0">required</Badge>
                    ) : (
                      <Badge className="text-xs bg-amber-500/10 text-amber-600 border-amber-500/20 shrink-0">optional</Badge>
                    )}
                  </div>
                  <div className="flex items-center gap-3 shrink-0">
                    {v.description && <span className="text-xs text-muted-foreground max-w-[200px] truncate hidden sm:inline">{v.description}</span>}
                    {v.default !== undefined && <span className="text-xs font-mono text-muted-foreground max-w-[120px] truncate">={v.default}</span>}
                    {!v.configured && (
                      <Button size="sm" variant="outline" className="text-xs h-7" onClick={() => handleAddDiscovered(v)}>
                        <Plus className="w-3 h-3" />Add
                      </Button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Create form */}
      {showForm && (
        <div className="mb-4 p-4 rounded-lg border border-border bg-card space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <Input placeholder="Variable key" value={newKey} onChange={(e) => setNewKey(e.target.value)} />
            <Input placeholder="Value" type={newSensitive ? "password" : "text"} value={newValue} onChange={(e) => setNewValue(e.target.value)} />
          </div>
          <Input placeholder="Description (optional)" value={newDescription} onChange={(e) => setNewDescription(e.target.value)} />
          <div className="flex items-center gap-4">
            <Select value={newCategory} onChange={(e) => setNewCategory(e.target.value as "terraform" | "env")}>
              <option value="terraform">OpenTofu</option>
              <option value="env">Environment</option>
            </Select>
            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <input type="checkbox" checked={newSensitive} onChange={(e) => setNewSensitive(e.target.checked)} className="rounded border-border" />
              Sensitive
            </label>
            <div className="flex-1" />
            <Button size="sm" variant="ghost" onClick={() => setShowForm(false)}>Cancel</Button>
            <Button size="sm" onClick={() => createMutation.mutate()} disabled={!newKey || createMutation.isPending}>
              {createMutation.isPending ? <Spinner /> : "Save"}
            </Button>
          </div>
        </div>
      )}

      {/* Variable list */}
      {!variables?.length ? (
        <div className="rounded-lg border border-dashed border-border p-8 text-center">
          <p className="text-sm text-muted-foreground">No variables configured yet.</p>
        </div>
      ) : (
        <div className="rounded-lg border border-border divide-y divide-border">
          {variables.map((v: WorkspaceVariable) =>
            editTarget === v.id ? (
              <div key={v.id} className="px-4 py-3 space-y-2 bg-accent/20">
                <div className="flex items-center gap-2">
                  <code className="text-sm font-mono font-medium">{v.key}</code>
                  <Badge variant="outline" className="text-xs">{v.category}</Badge>
                </div>
                <Input
                  placeholder={v.sensitive ? "Enter new value" : "Value"}
                  type={editSensitive ? "password" : "text"}
                  value={editValue}
                  onChange={(e) => setEditValue(e.target.value)}
                />
                <Input
                  placeholder="Description (optional)"
                  value={editDescription}
                  onChange={(e) => setEditDescription(e.target.value)}
                />
                <div className="flex items-center gap-4">
                  <label className="flex items-center gap-2 text-sm cursor-pointer">
                    <input type="checkbox" checked={editSensitive} onChange={(e) => setEditSensitive(e.target.checked)} className="rounded border-border" />
                    Sensitive
                  </label>
                  <div className="flex-1" />
                  <Button size="sm" variant="ghost" onClick={() => setEditTarget(null)}>Cancel</Button>
                  <Button size="sm" onClick={() => updateMutation.mutate(v.id)} disabled={updateMutation.isPending}>
                    {updateMutation.isPending ? <Spinner /> : "Save"}
                  </Button>
                </div>
              </div>
            ) : (
              <div key={v.id} className="flex items-center justify-between px-4 py-3">
                <div>
                  <div className="flex items-center gap-3">
                    <code className="text-sm font-mono font-medium">{v.key}</code>
                    <Badge variant="outline" className="text-xs">{v.category}</Badge>
                    {v.sensitive && <Lock className="w-3.5 h-3.5 text-muted-foreground" />}
                  </div>
                  {v.description && <p className="text-xs text-muted-foreground mt-0.5">{v.description}</p>}
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-sm font-mono text-muted-foreground max-w-[200px] truncate">
                    {v.sensitive ? (revealedValues[v.id] ?? "***") : v.value}
                  </span>
                  {v.sensitive && (
                    <button
                      onClick={() => toggleReveal(v)}
                      className="p-1 rounded hover:bg-accent text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                      title={revealedValues[v.id] ? "Hide value" : "Reveal value"}
                    >
                      {revealedValues[v.id] ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                    </button>
                  )}
                  <button
                    onClick={() => startEdit(v)}
                    className="p-1 rounded hover:bg-accent text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                    title="Edit variable"
                  >
                    <Pencil className="w-3.5 h-3.5" />
                  </button>
                  <button
                    onClick={() => setDeleteTarget(v)}
                    className="p-1 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors cursor-pointer"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
            )
          )}
        </div>
      )}

      {/* Delete dialog */}
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
            <Button variant="ghost" size="sm" onClick={() => setDeleteTarget(null)}>Cancel</Button>
            <Button variant="destructive" size="sm" onClick={() => { if (deleteTarget) deleteMutation.mutate(deleteTarget.id); }} disabled={deleteMutation.isPending}>
              {deleteMutation.isPending ? <Spinner /> : "Delete"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Bulk import dialog */}
      <Dialog open={showBulkImport} onClose={() => { setShowBulkImport(false); setBulkParsed(null); setBulkText(""); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Bulk Import Variables</DialogTitle>
            <DialogDescription>
              Paste variables in .env format (KEY=value), one per line. Lines starting with # are ignored.
            </DialogDescription>
          </DialogHeader>
          {!bulkParsed ? (
            <div className="space-y-3 mt-2">
              <textarea
                className="w-full h-40 rounded-lg border border-border bg-background p-3 font-mono text-sm resize-none focus:outline-none focus:ring-2 focus:ring-primary/30"
                placeholder={"DB_HOST=localhost\nDB_PORT=5432\n# Comment lines are ignored\nAPI_KEY=secret123"}
                value={bulkText}
                onChange={(e) => setBulkText(e.target.value)}
              />
              <div className="flex items-center gap-4">
                <Select value={bulkCategory} onChange={(e) => setBulkCategory(e.target.value as "terraform" | "env")}>
                  <option value="terraform">OpenTofu</option>
                  <option value="env">Environment</option>
                </Select>
                <label className="flex items-center gap-2 text-sm cursor-pointer">
                  <input type="checkbox" checked={bulkSensitive} onChange={(e) => setBulkSensitive(e.target.checked)} className="rounded border-border" />
                  Sensitive
                </label>
              </div>
              <div className="flex justify-end gap-2">
                <Button variant="ghost" size="sm" onClick={() => { setShowBulkImport(false); setBulkText(""); }}>Cancel</Button>
                <Button size="sm" onClick={() => { const parsed = parseEnvFormat(bulkText); if (parsed.length === 0) { toast.error("No valid variables found"); return; } setBulkParsed(parsed); }} disabled={!bulkText.trim()}>
                  Preview
                </Button>
              </div>
            </div>
          ) : (
            <div className="space-y-3 mt-2">
              <div className="rounded-lg border border-border divide-y divide-border max-h-60 overflow-auto">
                {bulkParsed.map((v, i) => (
                  <div key={i} className="flex items-center justify-between px-3 py-2">
                    <code className="text-sm font-mono">{v.key}</code>
                    <span className="text-sm font-mono text-muted-foreground max-w-[200px] truncate">
                      {bulkSensitive ? "***" : v.value}
                    </span>
                  </div>
                ))}
              </div>
              <p className="text-xs text-muted-foreground">{bulkParsed.length} variable(s) will be created as {bulkCategory}{bulkSensitive ? " (sensitive)" : ""}</p>
              <div className="flex justify-end gap-2">
                <Button variant="ghost" size="sm" onClick={() => setBulkParsed(null)}>Back</Button>
                <Button size="sm" onClick={() => bulkCreateMutation.mutate(bulkParsed)} disabled={bulkCreateMutation.isPending}>
                  {bulkCreateMutation.isPending ? <Spinner /> : `Import ${bulkParsed.length} variables`}
                </Button>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
