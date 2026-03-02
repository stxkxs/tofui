import { useMemo } from "react";
import { cn } from "@/lib/utils";

interface Props {
  planOutput: string;
}

interface ResourceBlock {
  header: string;
  action: "create" | "update" | "destroy" | "read" | "replace" | "unknown";
  lines: string[];
}

function stripAnsi(str: string): string {
  return str.replace(/\x1b\[[0-9;]*m/g, "");
}

function parseAction(header: string): ResourceBlock["action"] {
  if (header.includes("will be created")) return "create";
  if (header.includes("will be updated") || header.includes("will be changed"))
    return "update";
  if (header.includes("will be destroyed")) return "destroy";
  if (header.includes("will be read")) return "read";
  if (header.includes("must be replaced")) return "replace";
  return "unknown";
}

function parsePlanOutput(raw: string): {
  preamble: string[];
  blocks: ResourceBlock[];
  summary: string[];
} {
  const clean = stripAnsi(raw);
  const lines = clean.split(/\r?\n/);

  const preamble: string[] = [];
  const blocks: ResourceBlock[] = [];
  const summary: string[] = [];

  let current: ResourceBlock | null = null;
  let inSummary = false;

  for (const line of lines) {
    if (inSummary) {
      summary.push(line);
      continue;
    }

    // Detect resource block headers: "# aws_instance.example will be created"
    if (/^\s*#\s+\S+\s+will be /.test(line) || /^\s*#\s+\S+\s+must be /.test(line)) {
      if (current) blocks.push(current);
      current = {
        header: line.trim(),
        action: parseAction(line),
        lines: [],
      };
      continue;
    }

    // Detect summary section
    if (/^Plan:/.test(line.trim()) || /^(No changes|Apply complete)/.test(line.trim())) {
      if (current) {
        blocks.push(current);
        current = null;
      }
      inSummary = true;
      summary.push(line);
      continue;
    }

    if (current) {
      current.lines.push(line);
    } else {
      preamble.push(line);
    }
  }

  if (current) blocks.push(current);

  return { preamble, blocks, summary };
}

const actionStyles: Record<ResourceBlock["action"], { border: string; bg: string; text: string; label: string }> = {
  create: { border: "border-green-500/30", bg: "bg-green-500/10", text: "text-green-400", label: "+" },
  update: { border: "border-yellow-500/30", bg: "bg-yellow-500/10", text: "text-yellow-400", label: "~" },
  destroy: { border: "border-red-500/30", bg: "bg-red-500/10", text: "text-red-400", label: "-" },
  replace: { border: "border-orange-500/30", bg: "bg-orange-500/10", text: "text-orange-400", label: "±" },
  read: { border: "border-blue-500/30", bg: "bg-blue-500/10", text: "text-blue-400", label: "≡" },
  unknown: { border: "border-border", bg: "bg-accent/10", text: "text-muted-foreground", label: "?" },
};

export function PlanDiffViewer({ planOutput }: Props) {
  const { preamble, blocks, summary } = useMemo(
    () => parsePlanOutput(planOutput),
    [planOutput]
  );

  if (!blocks.length) {
    return (
      <div className="p-6 font-mono text-sm text-muted-foreground whitespace-pre-wrap">
        {stripAnsi(planOutput)}
      </div>
    );
  }

  return (
    <div className="p-6 space-y-4">
      {/* Preamble */}
      {preamble.some((l) => l.trim()) && (
        <pre className="text-xs text-muted-foreground font-mono whitespace-pre-wrap">
          {preamble.join("\n").trim()}
        </pre>
      )}

      {/* Resource blocks */}
      {blocks.map((block, i) => {
        const style = actionStyles[block.action];
        return (
          <div
            key={i}
            className={cn("rounded-lg border overflow-hidden", style.border)}
          >
            <div
              className={cn(
                "px-4 py-2 flex items-center gap-2 text-sm font-medium font-mono",
                style.bg,
                style.text
              )}
            >
              <span className="text-base leading-none">{style.label}</span>
              <span>{block.header.replace(/^#\s*/, "")}</span>
            </div>
            {block.lines.some((l) => l.trim()) && (
              <pre className="px-4 py-3 text-xs font-mono text-foreground/80 overflow-x-auto whitespace-pre-wrap bg-card/50">
                {block.lines.join("\n").trim()}
              </pre>
            )}
          </div>
        );
      })}

      {/* Summary */}
      {summary.some((l) => l.trim()) && (
        <div className="rounded-lg border border-border bg-accent/20 px-4 py-3">
          <pre className="text-sm font-mono text-foreground whitespace-pre-wrap">
            {summary.join("\n").trim()}
          </pre>
        </div>
      )}
    </div>
  );
}
