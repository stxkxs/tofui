import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatRelativeTime(date: string | Date): string {
  const now = new Date();
  const d = new Date(date);
  const seconds = Math.floor((now.getTime() - d.getTime()) / 1000);

  if (seconds < 60) return "just now";
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  return `${Math.floor(seconds / 86400)}d ago`;
}

export function getStatusColor(status: string): string {
  switch (status) {
    case "applied":
      return "text-success";
    case "planned":
      return "text-primary";
    case "planning":
    case "applying":
    case "queued":
      return "text-warning";
    case "errored":
      return "text-destructive";
    case "pending":
      return "text-muted-foreground";
    default:
      return "text-muted-foreground";
  }
}

export function getEnvironmentColor(env: string): string {
  switch (env) {
    case "production":
      return "bg-red-500/10 text-red-400 border-red-500/20";
    case "staging":
      return "bg-yellow-500/10 text-yellow-400 border-yellow-500/20";
    default:
      return "bg-blue-500/10 text-blue-400 border-blue-500/20";
  }
}
