// API types matching the OpenAPI spec
// In production, these would be generated via openapi-typescript

export interface User {
  id: string;
  org_id: string;
  email: string;
  name: string;
  avatar_url?: string;
  role: "owner" | "admin" | "operator" | "viewer";
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Workspace {
  id: string;
  org_id: string;
  name: string;
  description?: string;
  repo_url: string;
  repo_branch: string;
  working_dir: string;
  tofu_version: string;
  environment: "development" | "staging" | "production";
  auto_apply: boolean;
  requires_approval: boolean;
  vcs_trigger_enabled: boolean;
  locked: boolean;
  locked_by?: string | null;
  current_run_id?: string | null;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export type RunStatus =
  | "pending"
  | "queued"
  | "planning"
  | "planned"
  | "awaiting_approval"
  | "applying"
  | "applied"
  | "errored"
  | "cancelled"
  | "discarded";

export type RunOperation = "plan" | "apply" | "destroy";

export interface Run {
  id: string;
  workspace_id: string;
  org_id: string;
  operation: RunOperation;
  status: RunStatus;
  plan_output?: string;
  plan_log_url?: string;
  apply_log_url?: string;
  resources_added?: number;
  resources_changed?: number;
  resources_deleted?: number;
  error_message?: string;
  commit_sha?: string;
  created_by: string;
  started_at?: string | null;
  finished_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface StateVersion {
  id: string;
  workspace_id: string;
  org_id: string;
  run_id: string;
  serial: number;
  state_url: string;
  resource_count: number;
  resource_summary: string;
  created_at: string;
}

export interface WorkspaceVariable {
  id: string;
  workspace_id: string;
  org_id: string;
  key: string;
  value: string;
  sensitive: boolean;
  category: "terraform" | "env";
  created_at: string;
  updated_at: string;
}

export interface Team {
  id: string;
  org_id: string;
  name: string;
  slug: string;
  created_at: string;
  updated_at: string;
}

export interface TeamMember {
  id: string;
  team_id: string;
  user_id: string;
  role: string;
  created_at: string;
  email: string;
  user_name: string;
  avatar_url: string;
}

export interface WorkspaceTeamAccess {
  id: string;
  workspace_id: string;
  team_id: string;
  role: string;
  created_at: string;
  team_name: string;
  team_slug: string;
}

export interface Approval {
  id: string;
  run_id: string;
  org_id: string;
  user_id: string;
  status: "approved" | "rejected";
  comment: string;
  created_at: string;
  user_name?: string;
  avatar_url?: string;
}

export interface AuditLog {
  id: string;
  org_id: string;
  user_id: string;
  action: string;
  entity_type: string;
  entity_id: string;
  before_data: unknown;
  after_data: unknown;
  ip_address: string;
  user_agent: string;
  created_at: string;
}

export interface ListResponse<T> {
  data: T[];
  total: number;
  page: number;
  per_page: number;
}

export interface CreateWorkspaceRequest {
  name: string;
  description?: string;
  repo_url: string;
  repo_branch?: string;
  working_dir?: string;
  tofu_version?: string;
  environment?: "development" | "staging" | "production";
  auto_apply?: boolean;
  requires_approval?: boolean;
  vcs_trigger_enabled?: boolean;
}

export interface UpdateWorkspaceRequest {
  name?: string;
  description?: string;
  repo_url?: string;
  repo_branch?: string;
  working_dir?: string;
  tofu_version?: string;
  environment?: "development" | "staging" | "production";
  auto_apply?: boolean;
  requires_approval?: boolean;
  vcs_trigger_enabled?: boolean;
}

export interface CreateRunRequest {
  operation?: RunOperation;
}

export interface CreateVariableRequest {
  key: string;
  value: string;
  sensitive: boolean;
  category: "terraform" | "env";
}

export interface ApprovalRequest {
  status: "approved" | "rejected";
  comment?: string;
}

export interface DiscoveredVariable {
  name: string;
  type?: string;
  description?: string;
  default?: string;
  required: boolean;
  configured: boolean;
}

export interface ErrorResponse {
  error: string;
  message?: string;
}

export interface UpdateRoleRequest {
  role: "owner" | "admin" | "operator" | "viewer";
}

// openapi-fetch compatible paths type
export interface paths {
  "/users": {
    get: {
      responses: {
        200: { content: { "application/json": User[] } };
      };
    };
  };
  "/users/{userId}/role": {
    put: {
      parameters: { path: { userId: string } };
      requestBody: {
        content: { "application/json": UpdateRoleRequest };
      };
      responses: {
        200: { content: { "application/json": User } };
      };
    };
  };
  "/audit-logs": {
    get: {
      parameters: { query?: { page?: number; per_page?: number } };
      responses: {
        200: { content: { "application/json": AuditLog[] } };
      };
    };
  };
  "/health": {
    get: {
      responses: {
        200: { content: { "application/json": { status: string } } };
      };
    };
  };
  "/auth/me": {
    get: {
      responses: {
        200: { content: { "application/json": User } };
        401: { content: { "application/json": ErrorResponse } };
      };
    };
  };
  "/teams": {
    get: {
      responses: {
        200: { content: { "application/json": Team[] } };
      };
    };
    post: {
      requestBody: { content: { "application/json": { name: string } } };
      responses: {
        201: { content: { "application/json": Team } };
      };
    };
  };
  "/teams/{teamId}": {
    get: {
      parameters: { path: { teamId: string } };
      responses: {
        200: { content: { "application/json": Team } };
      };
    };
    delete: {
      parameters: { path: { teamId: string } };
      responses: {
        204: { content: never };
      };
    };
  };
  "/teams/{teamId}/members": {
    get: {
      parameters: { path: { teamId: string } };
      responses: {
        200: { content: { "application/json": TeamMember[] } };
      };
    };
    post: {
      parameters: { path: { teamId: string } };
      requestBody: {
        content: {
          "application/json": { user_id: string; role: string };
        };
      };
      responses: {
        201: { content: { "application/json": TeamMember } };
      };
    };
  };
  "/teams/{teamId}/members/{userId}": {
    delete: {
      parameters: { path: { teamId: string; userId: string } };
      responses: {
        204: { content: never };
      };
    };
  };
  "/workspaces": {
    get: {
      parameters: { query?: { page?: number; per_page?: number } };
      responses: {
        200: { content: { "application/json": ListResponse<Workspace> } };
      };
    };
    post: {
      requestBody: { content: { "application/json": CreateWorkspaceRequest } };
      responses: {
        201: { content: { "application/json": Workspace } };
      };
    };
  };
  "/workspaces/{workspaceId}": {
    get: {
      parameters: { path: { workspaceId: string } };
      responses: {
        200: { content: { "application/json": Workspace } };
        404: { content: { "application/json": ErrorResponse } };
      };
    };
    put: {
      parameters: { path: { workspaceId: string } };
      requestBody: {
        content: { "application/json": UpdateWorkspaceRequest };
      };
      responses: {
        200: { content: { "application/json": Workspace } };
      };
    };
    delete: {
      parameters: { path: { workspaceId: string } };
      responses: {
        204: { content: never };
      };
    };
  };
  "/workspaces/{workspaceId}/lock": {
    post: {
      parameters: { path: { workspaceId: string } };
      responses: {
        200: { content: { "application/json": Workspace } };
      };
    };
  };
  "/workspaces/{workspaceId}/unlock": {
    post: {
      parameters: { path: { workspaceId: string } };
      responses: {
        200: { content: { "application/json": Workspace } };
      };
    };
  };
  "/workspaces/{workspaceId}/variables": {
    get: {
      parameters: { path: { workspaceId: string } };
      responses: {
        200: {
          content: { "application/json": WorkspaceVariable[] };
        };
      };
    };
    post: {
      parameters: { path: { workspaceId: string } };
      requestBody: {
        content: { "application/json": CreateVariableRequest };
      };
      responses: {
        201: { content: { "application/json": WorkspaceVariable } };
      };
    };
  };
  "/workspaces/{workspaceId}/variables/discover": {
    post: {
      parameters: { path: { workspaceId: string } };
      responses: {
        200: {
          content: { "application/json": DiscoveredVariable[] };
        };
      };
    };
  };
  "/workspaces/{workspaceId}/variables/{variableId}": {
    put: {
      parameters: { path: { workspaceId: string; variableId: string } };
      requestBody: {
        content: { "application/json": CreateVariableRequest };
      };
      responses: {
        200: { content: { "application/json": WorkspaceVariable } };
      };
    };
    delete: {
      parameters: { path: { workspaceId: string; variableId: string } };
      responses: {
        204: { content: never };
      };
    };
  };
  "/workspaces/{workspaceId}/state": {
    get: {
      parameters: { path: { workspaceId: string } };
      responses: {
        200: { content: { "application/json": StateVersion[] } };
      };
    };
  };
  "/workspaces/{workspaceId}/state/current": {
    get: {
      parameters: { path: { workspaceId: string } };
      responses: {
        200: { content: { "application/json": StateVersion } };
        404: { content: { "application/json": ErrorResponse } };
      };
    };
  };
  "/workspaces/{workspaceId}/state/{stateId}": {
    get: {
      parameters: { path: { workspaceId: string; stateId: string } };
      responses: {
        200: { content: { "application/json": StateVersion } };
      };
    };
  };
  "/workspaces/{workspaceId}/access": {
    get: {
      parameters: { path: { workspaceId: string } };
      responses: {
        200: { content: { "application/json": WorkspaceTeamAccess[] } };
      };
    };
    post: {
      parameters: { path: { workspaceId: string } };
      requestBody: {
        content: {
          "application/json": { team_id: string; role: string };
        };
      };
      responses: {
        201: { content: { "application/json": WorkspaceTeamAccess } };
      };
    };
  };
  "/workspaces/{workspaceId}/access/{teamId}": {
    delete: {
      parameters: { path: { workspaceId: string; teamId: string } };
      responses: {
        204: { content: never };
      };
    };
  };
  "/workspaces/{workspaceId}/runs": {
    get: {
      parameters: {
        path: { workspaceId: string };
        query?: { page?: number; per_page?: number };
      };
      responses: {
        200: { content: { "application/json": ListResponse<Run> } };
      };
    };
    post: {
      parameters: { path: { workspaceId: string } };
      requestBody: { content: { "application/json": CreateRunRequest } };
      responses: {
        201: { content: { "application/json": Run } };
      };
    };
  };
  "/workspaces/{workspaceId}/runs/{runId}": {
    get: {
      parameters: { path: { workspaceId: string; runId: string } };
      responses: {
        200: { content: { "application/json": Run } };
      };
    };
  };
  "/workspaces/{workspaceId}/runs/{runId}/cancel": {
    post: {
      parameters: { path: { workspaceId: string; runId: string } };
      responses: {
        200: { content: { "application/json": Run } };
      };
    };
  };
  "/workspaces/{workspaceId}/runs/{runId}/approvals": {
    get: {
      parameters: { path: { workspaceId: string; runId: string } };
      responses: {
        200: { content: { "application/json": Approval[] } };
      };
    };
    post: {
      parameters: { path: { workspaceId: string; runId: string } };
      requestBody: {
        content: { "application/json": ApprovalRequest };
      };
      responses: {
        201: { content: { "application/json": Approval } };
      };
    };
  };
}
