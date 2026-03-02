import { useMemo } from "react";
import { ErrorBoundary } from "react-error-boundary";
import { AppLayout } from "@/components/layout/AppLayout";
import { ErrorFallback } from "@/components/ErrorFallback";
import { WorkspaceList } from "@/components/workspace/WorkspaceList";
import { WorkspaceDetail } from "@/components/workspace/WorkspaceDetail";
import { RunView } from "@/components/run/RunView";
import { LoginPage } from "@/routes/Login";
import { AuthCallbackPage } from "@/routes/AuthCallback";
import { TeamsPage } from "@/components/teams/TeamsPage";
import { UsersPage } from "@/components/users/UsersPage";
import { AuditLogPage } from "@/components/audit/AuditLogPage";
import { useAuth } from "@/hooks/useAuth";
import { Spinner } from "@/components/ui/spinner";
import { FileQuestion } from "lucide-react";

function useRoute() {
  return useMemo(() => {
    const path = window.location.pathname;

    if (path === "/login") return { page: "login" as const };
    if (path === "/auth/callback") return { page: "callback" as const };

    // /workspaces/:id/runs/:runId
    const runMatch = path.match(/^\/workspaces\/([^/]+)\/runs\/([^/]+)/);
    if (runMatch)
      return {
        page: "run" as const,
        workspaceId: runMatch[1],
        runId: runMatch[2],
      };

    // /workspaces/:id
    const wsMatch = path.match(/^\/workspaces\/([^/]+)/);
    if (wsMatch)
      return { page: "workspace" as const, workspaceId: wsMatch[1] };

    // /teams
    if (path === "/teams") return { page: "teams" as const };

    // /users
    if (path === "/users") return { page: "users" as const };

    // /audit-logs
    if (path === "/audit-logs") return { page: "audit-logs" as const };

    if (path === "/") return { page: "home" as const };
    return { page: "not-found" as const };
  }, []);
}

function NotFoundPage() {
  return (
    <div className="flex flex-col items-center justify-center py-20">
      <FileQuestion className="w-12 h-12 text-muted-foreground mb-4" />
      <h1 className="text-xl font-bold mb-2">Page not found</h1>
      <p className="text-sm text-muted-foreground mb-4">
        The page you're looking for doesn't exist.
      </p>
      <a
        href="/"
        className="text-sm text-primary hover:underline"
      >
        Back to dashboard
      </a>
    </div>
  );
}

export function App() {
  const route = useRoute();
  const { user, isLoading } = useAuth();

  // Public routes
  if (route.page === "login") return <LoginPage />;
  if (route.page === "callback") return <AuthCallbackPage />;

  // Auth loading
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <Spinner className="w-8 h-8" />
      </div>
    );
  }

  // Not logged in
  if (!user) {
    window.location.href = "/login";
    return null;
  }

  return (
    <AppLayout>
      <ErrorBoundary FallbackComponent={ErrorFallback} onReset={() => window.location.reload()}>
        {route.page === "home" && <WorkspaceList />}
        {route.page === "workspace" && (
          <WorkspaceDetail workspaceId={route.workspaceId!} />
        )}
        {route.page === "run" && (
          <RunView
            workspaceId={route.workspaceId!}
            runId={route.runId!}
          />
        )}
        {route.page === "teams" && <TeamsPage />}
        {route.page === "users" && <UsersPage />}
        {route.page === "audit-logs" && <AuditLogPage />}
        {route.page === "not-found" && <NotFoundPage />}
      </ErrorBoundary>
    </AppLayout>
  );
}
