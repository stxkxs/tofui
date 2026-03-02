import type { RunStatus } from "@/api/types";
import { Badge, type BadgeProps } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import {
  CheckCircle2,
  XCircle,
  Clock,
  CircleDot,
  Ban,
  ShieldQuestion,
} from "lucide-react";

const statusConfig: Record<
  RunStatus,
  {
    label: string;
    variant: BadgeProps["variant"];
    icon: typeof CheckCircle2;
    spinning?: boolean;
  }
> = {
  pending: { label: "Pending", variant: "secondary", icon: Clock },
  queued: { label: "Queued", variant: "secondary", icon: Clock },
  planning: { label: "Planning", variant: "warning", icon: CircleDot, spinning: true },
  planned: { label: "Planned", variant: "default", icon: CheckCircle2 },
  awaiting_approval: { label: "Needs Approval", variant: "warning", icon: ShieldQuestion },
  applying: { label: "Applying", variant: "warning", icon: CircleDot, spinning: true },
  applied: { label: "Applied", variant: "success", icon: CheckCircle2 },
  errored: { label: "Errored", variant: "destructive", icon: XCircle },
  cancelled: { label: "Cancelled", variant: "secondary", icon: Ban },
  discarded: { label: "Discarded", variant: "secondary", icon: Ban },
};

export function RunStatusBadge({ status }: { status: RunStatus }) {
  const config = statusConfig[status] || statusConfig.pending;
  const Icon = config.icon;

  return (
    <Badge variant={config.variant} className="gap-1.5" aria-label={`Status: ${config.label}`}>
      {config.spinning ? (
        <Spinner className="w-3 h-3" aria-hidden="true" />
      ) : (
        <Icon className="w-3 h-3" aria-hidden="true" />
      )}
      {config.label}
    </Badge>
  );
}
