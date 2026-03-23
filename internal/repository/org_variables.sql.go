package repository

import "context"

const orgVarColumns = `id, org_id, key, value, sensitive, category, description, created_at, updated_at`

func scanOrgVariable(row interface{ Scan(...interface{}) error }) (OrgVariable, error) {
	var v OrgVariable
	err := row.Scan(&v.ID, &v.OrgID, &v.Key, &v.Value, &v.Sensitive, &v.Category, &v.Description, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}

func (q *Queries) ListOrgVariables(ctx context.Context, orgID string) ([]OrgVariable, error) {
	rows, err := q.db.Query(ctx,
		`SELECT `+orgVarColumns+` FROM org_variables WHERE org_id = $1 ORDER BY key`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vars []OrgVariable
	for rows.Next() {
		v, err := scanOrgVariable(rows)
		if err != nil {
			return nil, err
		}
		vars = append(vars, v)
	}
	if vars == nil {
		vars = []OrgVariable{}
	}
	return vars, rows.Err()
}

type GetOrgVariableParams struct {
	ID    string
	OrgID string
}

func (q *Queries) GetOrgVariable(ctx context.Context, arg GetOrgVariableParams) (OrgVariable, error) {
	row := q.db.QueryRow(ctx,
		`SELECT `+orgVarColumns+` FROM org_variables WHERE id = $1 AND org_id = $2`,
		arg.ID, arg.OrgID,
	)
	return scanOrgVariable(row)
}

type CreateOrgVariableParams struct {
	ID          string
	OrgID       string
	Key         string
	Value       string
	Sensitive   bool
	Category    string
	Description string
}

func (q *Queries) CreateOrgVariable(ctx context.Context, arg CreateOrgVariableParams) (OrgVariable, error) {
	row := q.db.QueryRow(ctx,
		`INSERT INTO org_variables (id, org_id, key, value, sensitive, category, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+orgVarColumns,
		arg.ID, arg.OrgID, arg.Key, arg.Value, arg.Sensitive, arg.Category, arg.Description,
	)
	return scanOrgVariable(row)
}

type UpdateOrgVariableParams struct {
	ID          string
	OrgID       string
	Value       string
	Sensitive   bool
	Description string
	Category    string
}

func (q *Queries) UpdateOrgVariable(ctx context.Context, arg UpdateOrgVariableParams) (OrgVariable, error) {
	row := q.db.QueryRow(ctx,
		`UPDATE org_variables SET value = $3, sensitive = $4, description = $5, category = COALESCE(NULLIF($6, ''), category), updated_at = NOW()
		WHERE id = $1 AND org_id = $2
		RETURNING `+orgVarColumns,
		arg.ID, arg.OrgID, arg.Value, arg.Sensitive, arg.Description, arg.Category,
	)
	return scanOrgVariable(row)
}

type DeleteOrgVariableParams struct {
	ID    string
	OrgID string
}

func (q *Queries) DeleteOrgVariable(ctx context.Context, arg DeleteOrgVariableParams) error {
	_, err := q.db.Exec(ctx,
		`DELETE FROM org_variables WHERE id = $1 AND org_id = $2`,
		arg.ID, arg.OrgID,
	)
	return err
}
