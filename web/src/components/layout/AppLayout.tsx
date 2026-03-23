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
  GitBranch,
  Settings,
} from "lucide-react";

function NavLink({
  href,
  active,
  icon: Icon,
  children,
}: {
  href: string;
  active: boolean;
  icon: React.ComponentType<{ className?: string }>;
  children: ReactNode;
}) {
  return (
    <Link
      href={href}
      className={`flex items-center gap-2.5 px-3 py-2 rounded-lg text-[13px] font-medium transition-all duration-150 ${
        active
          ? "text-foreground bg-primary/10 shadow-sm shadow-primary/5"
          : "text-muted-foreground hover:text-foreground hover:bg-accent/40"
      }`}
    >
      <Icon className={`w-4 h-4 ${active ? "text-primary" : ""}`} />
      {children}
    </Link>
  );
}

export function AppLayout({ children }: { children: ReactNode }) {
  const { user, logout } = useAuth();
  const location = useLocation();
  const path = location.split("?")[0];

  return (
    <div className="h-screen flex">
      {/* Sidebar */}
      <aside className="w-56 border-r border-border/60 bg-card/50 flex flex-col" aria-label="Main navigation">
        <div className="px-4 py-4 border-b border-border/60">
          <Link href="/" className="flex items-center gap-2.5">
            <div className="w-7 h-7 rounded-lg bg-primary/15 flex items-center justify-center">
              <Box className="w-3.5 h-3.5 text-primary" />
            </div>
            <span className="font-bold text-base tracking-tight">tofui</span>
          </Link>
        </div>

        <nav className="flex-1 p-2.5 space-y-0.5" aria-label="Primary">
          <NavLink
            href="/"
            active={path === "/" || path.startsWith("/workspaces")}
            icon={LayoutDashboard}
          >
            Workspaces
          </NavLink>
          <NavLink
            href="/pipelines"
            active={path.startsWith("/pipelines")}
            icon={GitBranch}
          >
            Pipelines
          </NavLink>
          <NavLink href="/teams" active={path === "/teams"} icon={Users}>
            Teams
          </NavLink>
          {(user?.role === "admin" || user?.role === "owner") && (
            <NavLink href="/users" active={path === "/users"} icon={UserCog}>
              Users
            </NavLink>
          )}
          {(user?.role === "admin" || user?.role === "owner") && (
            <NavLink
              href="/audit-logs"
              active={path === "/audit-logs"}
              icon={Shield}
            >
              Audit Logs
            </NavLink>
          )}
          {(user?.role === "admin" || user?.role === "owner") && (
            <NavLink
              href="/settings"
              active={path === "/settings"}
              icon={Settings}
            >
              Settings
            </NavLink>
          )}
        </nav>

        {user && (
          <div className="p-2.5 border-t border-border/60">
            <div className="flex items-center gap-2.5 px-2 py-1.5">
              {user.avatar_url ? (
                <img
                  src={user.avatar_url}
                  alt={user.name}
                  className="w-6 h-6 rounded-full ring-1 ring-border/60"
                />
              ) : (
                <div className="w-6 h-6 rounded-full bg-primary/15 flex items-center justify-center text-[10px] font-semibold text-primary">
                  {user.name[0]}
                </div>
              )}
              <div className="flex-1 min-w-0">
                <div className="text-xs font-medium break-words">{user.name}</div>
                <div className="text-[10px] text-muted-foreground break-words">
                  {user.email}
                </div>
              </div>
              <button
                onClick={logout}
                aria-label="Log out"
                className="p-1.5 rounded-md hover:bg-accent/40 transition-colors text-muted-foreground hover:text-foreground cursor-pointer"
              >
                <LogOut className="w-3 h-3" />
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
