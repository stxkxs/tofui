import { useState } from "react";
import type { StateDiff, ResourceDiff } from "@/api/types";
import { ChevronDown, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

interface Props {
  diff: StateDiff;
}

const actionStyles: Record<
  ResourceDiff["action"],
  { border: string; bg: string; text: string; label: string }
> = {
  added: {
    border: "border-green-500/30",
    bg: "bg-green-500/10",
    text: "text-green-400",
    label: "+",
  },
  removed: {
    border: "border-red-500/30",
    bg: "bg-red-500/10",
    text: "text-red-400",
    label: "-",
  },
  changed: {
    border: "border-yellow-500/30",
    bg: "bg-yellow-500/10",
    text: "text-yellow-400",
    label: "~",
  },
  unchanged: {
    border: "border-border",
    bg: "bg-accent/10",
    text: "text-muted-foreground",
    label: "=",
  },
};

function formatValue(val: unknown): string {
  if (val === null || val === undefined) return "null";
  if (typeof val === "string") return `"${val}"`;
  if (typeof val === "object") return JSON.stringify(val, null, 2);
  return String(val);
}

function DiffRow({ diff }: { diff: ResourceDiff }) {
  const [expanded, setExpanded] = useState(false);
  const style = actionStyles[diff.action];
  const address = diff.module
    ? `${diff.module}.${diff.type}.${diff.name}`
    : `${diff.type}.${diff.name}`;

  return (
    <div className={cn("rounded-lg border overflow-hidden", style.border)}>
      <button
        onClick={() => setExpanded(!expanded)}
        className={cn(
          "w-full px-4 py-2 flex items-center gap-2 text-sm font-medium font-mono text-left cursor-pointer",
          style.bg,
          style.text
        )}
      >
        <span className="text-base leading-none">{style.label}</span>
        <span className="flex-1">{address}</span>
        {diff.changed_keys && diff.changed_keys.length > 0 && (
          <span className="text-xs opacity-70">
            {diff.changed_keys.length} attribute(s)
          </span>
        )}
        {expanded ? (
          <ChevronDown className="w-4 h-4" />
        ) : (
          <ChevronRight className="w-4 h-4" />
        )}
      </button>
      {expanded && diff.action === "changed" && diff.changed_keys && (
        <div className="divide-y divide-border/50">
          {diff.changed_keys.map((key) => {
            const before = diff.before?.[key];
            const after = diff.after?.[key];
            const isAdded = before === undefined;
            const isRemoved = after === undefined;

            return (
              <div key={key} className="px-4 py-2 text-xs font-mono">
                <span className="text-muted-foreground">{key}: </span>
                {isAdded ? (
                  <span className="text-green-400">{formatValue(after)}</span>
                ) : isRemoved ? (
                  <span className="text-red-400 line-through">
                    {formatValue(before)}
                  </span>
                ) : (
                  <>
                    <span className="text-red-400 line-through">
                      {formatValue(before)}
                    </span>
                    <span className="text-muted-foreground mx-1">{"->"}</span>
                    <span className="text-green-400">{formatValue(after)}</span>
                  </>
                )}
              </div>
            );
          })}
        </div>
      )}
      {expanded && diff.action === "added" && diff.after && (
        <div className="divide-y divide-border/50">
          {Object.entries(diff.after)
            .sort(([a], [b]) => a.localeCompare(b))
            .map(([key, value]) => (
              <div key={key} className="px-4 py-2 text-xs font-mono">
                <span className="text-muted-foreground">{key}: </span>
                <span className="text-green-400">{formatValue(value)}</span>
              </div>
            ))}
        </div>
      )}
      {expanded && diff.action === "removed" && diff.before && (
        <div className="divide-y divide-border/50">
          {Object.entries(diff.before)
            .sort(([a], [b]) => a.localeCompare(b))
            .map(([key, value]) => (
              <div key={key} className="px-4 py-2 text-xs font-mono">
                <span className="text-muted-foreground">{key}: </span>
                <span className="text-red-400 line-through">
                  {formatValue(value)}
                </span>
              </div>
            ))}
        </div>
      )}
    </div>
  );
}

export function StateDiffViewer({ diff }: Props) {
  if (diff.diffs.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-border p-6 text-center">
        <p className="text-sm text-muted-foreground">
          No differences between the selected state versions.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4 mt-4">
      {/* Summary */}
      <div className="flex items-center gap-4 text-sm font-mono">
        {diff.added > 0 && (
          <span className="text-green-400">+{diff.added} added</span>
        )}
        {diff.removed > 0 && (
          <span className="text-red-400">-{diff.removed} removed</span>
        )}
        {diff.changed > 0 && (
          <span className="text-yellow-400">~{diff.changed} changed</span>
        )}
        {diff.unchanged > 0 && (
          <span className="text-muted-foreground">
            {diff.unchanged} unchanged
          </span>
        )}
      </div>

      {diff.diffs.map((d, i) => (
        <DiffRow
          key={`${d.module}.${d.type}.${d.name}-${i}`}
          diff={d}
        />
      ))}
    </div>
  );
}
