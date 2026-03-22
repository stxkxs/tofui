import type { ReactNode } from "react";
import { useAuth } from "@/hooks/useAuth";
import { useLocation } from "@/hooks/useNavigate";
import { Link } from "@/components/ui/link";
import {
  LayoutDashboard,
  LogOut,
  Box,
  Users,
  UserCog,
  Shield,
} from "lucide-react";

export function AppLayout({ children }: { children: ReactNode }) {
  const { user, logout } = useAuth();
  const location = useLocation();
  const path = location.split("?")[0];

  return (
    <div className="h-screen flex">
      {/* Sidebar */}
      <aside className="w-60 border-r border-border bg-card flex flex-col" aria-label="Main navigation">
        <div className="p-4 border-b border-border">
          <Link href="/" className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
              <Box className="w-4.5 h-4.5 text-primary-foreground" />
            </div>
            <span className="font-bold text-lg tracking-tight">tofui</span>
          </Link>
        </div>

        <nav className="flex-1 p-3 space-y-1" aria-label="Primary">
          <Link
            href="/"
            className={`flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
              path === "/" || path.startsWith("/workspaces")
                ? "text-foreground bg-accent/50"
                : "text-muted-foreground hover:text-foreground hover:bg-accent/30"
            }`}
          >
            <LayoutDashboard className="w-4 h-4" />
            Workspaces
          </Link>
          <Link
            href="/teams"
            className={`flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
              path === "/teams"
                ? "text-foreground bg-accent/50"
                : "text-muted-foreground hover:text-foreground hover:bg-accent/30"
            }`}
          >
            <Users className="w-4 h-4" />
            Teams
          </Link>
          {(user?.role === "admin" || user?.role === "owner") && (
            <Link
              href="/users"
              className={`flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                path === "/users"
                  ? "text-foreground bg-accent/50"
                  : "text-muted-foreground hover:text-foreground hover:bg-accent/30"
              }`}
            >
              <UserCog className="w-4 h-4" />
              Users
            </Link>
          )}
          {(user?.role === "admin" || user?.role === "owner") && (
            <Link
              href="/audit-logs"
              className={`flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                path === "/audit-logs"
                  ? "text-foreground bg-accent/50"
                  : "text-muted-foreground hover:text-foreground hover:bg-accent/30"
              }`}
            >
              <Shield className="w-4 h-4" />
              Audit Logs
            </Link>
          )}
        </nav>

        {user && (
          <div className="p-3 border-t border-border">
            <div className="flex items-center gap-2.5 px-2">
              {user.avatar_url ? (
                <img
                  src={user.avatar_url}
                  alt={user.name}
                  className="w-7 h-7 rounded-full"
                />
              ) : (
                <div className="w-7 h-7 rounded-full bg-primary/20 flex items-center justify-center text-xs font-medium">
                  {user.name[0]}
                </div>
              )}
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium truncate">{user.name}</div>
                <div className="text-xs text-muted-foreground truncate">
                  {user.email}
                </div>
              </div>
              <button
                onClick={logout}
                aria-label="Log out"
                className="p-1.5 rounded-md hover:bg-accent transition-colors text-muted-foreground hover:text-foreground cursor-pointer"
              >
                <LogOut className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        )}
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto" role="main">{children}</main>
    </div>
  );
}
