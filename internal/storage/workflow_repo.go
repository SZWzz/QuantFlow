package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"quantflow/internal/workflow"
	"time"
)

// WorkflowMeta is a lightweight summary of a workflow, returned by List().
type WorkflowMeta struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	UpdatedAt   string `json:"updated_at"`
}

// WorkflowRepo persists and loads workflow definitions and execution results.
type WorkflowRepo struct {
	db *sql.DB
}

// NewWorkflowRepo creates a WorkflowRepo backed by the given database.
func NewWorkflowRepo(db *sql.DB) *WorkflowRepo {
	return &WorkflowRepo{db: db}
}

// Save persists a workflow definition. It upserts the workflow metadata in the
// workflows table and creates a new version entry in workflow_versions with the
// full graph JSON. The version number auto-increments per workflow.
func (r *WorkflowRepo) Save(wf *workflow.Workflow) error {
	graphJSON, err := json.Marshal(wf)
	if err != nil {
		return fmt.Errorf("marshal workflow: %w", err)
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	// Rollback after a successful Commit returns sql.ErrTxDone — safe to ignore
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(`INSERT INTO workflows (id, name, description, updated_at)
		VALUES (?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, description=excluded.description, updated_at=datetime('now')`,
		wf.ID, wf.Name, wf.Description)
	if err != nil {
		return fmt.Errorf("upsert workflow: %w", err)
	}

	var maxVersion int
	if err := tx.QueryRow("SELECT COALESCE(MAX(version), 0) FROM workflow_versions WHERE workflow_id = ?", wf.ID).Scan(&maxVersion); err != nil {
		return fmt.Errorf("get max version: %w", err)
	}
	nextVersion := maxVersion + 1

	_, err = tx.Exec(`INSERT INTO workflow_versions (workflow_id, version, graph_json) VALUES (?, ?, ?)`,
		wf.ID, nextVersion, string(graphJSON))
	if err != nil {
		return fmt.Errorf("insert version: %w", err)
	}

	return tx.Commit()
}

// Load retrieves a workflow definition by ID. If version is nil, the latest
// version is returned. If a specific version is requested and not found, an
// error is returned.
func (r *WorkflowRepo) Load(id string, version *int) (*workflow.Workflow, error) {
	var graphJSON string
	if version != nil {
		err := r.db.QueryRow("SELECT graph_json FROM workflow_versions WHERE workflow_id = ? AND version = ?", id, *version).Scan(&graphJSON)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow %q version %d not found", id, *version)
		}
		if err != nil {
			return nil, fmt.Errorf("load version: %w", err)
		}
	} else {
		err := r.db.QueryRow("SELECT graph_json FROM workflow_versions WHERE workflow_id = ? ORDER BY version DESC LIMIT 1", id).Scan(&graphJSON)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow %q not found", id)
		}
		if err != nil {
			return nil, fmt.Errorf("load latest: %w", err)
		}
	}

	var wf workflow.Workflow
	if err := json.Unmarshal([]byte(graphJSON), &wf); err != nil {
		return nil, fmt.Errorf("unmarshal workflow: %w", err)
	}
	return &wf, nil
}

// List returns metadata for all workflows, ordered by most recently updated first.
func (r *WorkflowRepo) List() ([]WorkflowMeta, error) {
	rows, err := r.db.Query("SELECT id, name, description, updated_at FROM workflows ORDER BY updated_at DESC")
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}
	defer rows.Close()

	var metas []WorkflowMeta
	for rows.Next() {
		var m WorkflowMeta
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan workflow: %w", err)
		}
		metas = append(metas, m)
	}
	return metas, rows.Err()
}

// SaveExecution records a workflow execution result in the execution_history table.
// The version argument identifies which workflow version was executed.
func (r *WorkflowRepo) SaveExecution(workflowID string, version int, status string, result *workflow.ExecutionResult) error {
	var resultJSON, errMsg string
	if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("marshal result: %w", err)
		}
		resultJSON = string(data)
		if result.Error != "" {
			errMsg = result.Error
		}
	}

	now := time.Now().Format(time.RFC3339)
	_, err := r.db.Exec(`INSERT INTO execution_history (workflow_id, version, started_at, finished_at, status, result_json, error_msg)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, workflowID, version, now, now, status, resultJSON, errMsg)
	if err != nil {
		return fmt.Errorf("save execution: %w", err)
	}
	return nil
}
