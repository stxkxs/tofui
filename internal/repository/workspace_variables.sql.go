package repository

import "context"

const varColumns = `id, workspace_id, org_id, key, value, sensitive, category, description, created_at, updated_at`

func scanVariable(row interface{ Scan(...interface{}) error }) (WorkspaceVariable, error) {
	var v WorkspaceVariable
	err := row.Scan(&v.ID, &v.WorkspaceID, &v.OrgID, &v.Key, &v.Value, &v.Sensitive, &v.Category, &v.Description, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}

type GetWorkspaceVariableParams struct {
	ID    string
	OrgID string
}

func (q *Queries) GetWorkspaceVariable(ctx context.Context, arg GetWorkspaceVariableParams) (WorkspaceVariable, error) {
	row := q.db.QueryRow(ctx,
		`SELECT `+varColumns+` FROM workspace_variables WHERE id = $1 AND org_id = $2`,
		arg.ID, arg.OrgID,
	)
	return scanVariable(row)
}

type ListWorkspaceVariablesParams struct {
	WorkspaceID string
	OrgID       string
}

func (q *Queries) ListWorkspaceVariables(ctx context.Context, arg ListWorkspaceVariablesParams) ([]WorkspaceVariable, error) {
	rows, err := q.db.Query(ctx,
		`SELECT `+varColumns+` FROM workspace_variables WHERE workspace_id = $1 AND org_id = $2 ORDER BY key`,
		arg.WorkspaceID, arg.OrgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vars []WorkspaceVariable
	for rows.Next() {
		v, err := scanVariable(rows)
		if err != nil {
			return nil, err
		}
		vars = append(vars, v)
	}
	if vars == nil {
		vars = []WorkspaceVariable{}
	}
	return vars, rows.Err()
}

type CreateWorkspaceVariableParams struct {
	ID          string
	WorkspaceID string
	OrgID       string
	Key         string
	Value       string
	Sensitive   bool
	Category    string
	Description string
}

func (q *Queries) CreateWorkspaceVariable(ctx context.Context, arg CreateWorkspaceVariableParams) (WorkspaceVariable, error) {
	row := q.db.QueryRow(ctx,
		`INSERT INTO workspace_variables (id, workspace_id, org_id, key, value, sensitive, category, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING `+varColumns,
		arg.ID, arg.WorkspaceID, arg.OrgID, arg.Key, arg.Value, arg.Sensitive, arg.Category, arg.Description,
	)
	return scanVariable(row)
}

type UpdateWorkspaceVariableParams struct {
	ID          string
	OrgID       string
	Value       string
	Sensitive   bool
	Description string
	Category    string
}

func (q *Queries) UpdateWorkspaceVariable(ctx context.Context, arg UpdateWorkspaceVariableParams) (WorkspaceVariable, error) {
	row := q.db.QueryRow(ctx,
		`UPDATE workspace_variables SET value = $3, sensitive = $4, description = $5, category = COALESCE(NULLIF($6, ''), category), updated_at = NOW()
		WHERE id = $1 AND org_id = $2
		RETURNING `+varColumns,
		arg.ID, arg.OrgID, arg.Value, arg.Sensitive, arg.Description, arg.Category,
	)
	return scanVariable(row)
}

type DeleteWorkspaceVariableParams struct {
	ID    string
	OrgID string
}

func (q *Queries) DeleteWorkspaceVariable(ctx context.Context, arg DeleteWorkspaceVariableParams) error {
	_, err := q.db.Exec(ctx,
		`DELETE FROM workspace_variables WHERE id = $1 AND org_id = $2`,
		arg.ID, arg.OrgID,
	)
	return err
}
