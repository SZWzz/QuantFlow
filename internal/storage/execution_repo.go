package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ExecutionRecord represents a persisted workflow execution history entry.
type ExecutionRecord struct {
	ID           int    `json:"id"`
	RunID        string `json:"run_id"`
	WorkflowID   string `json:"workflow_id"`
	WorkflowName string `json:"workflow_name"`
	WorkflowJSON string `json:"workflow_json"`
	Status       string `json:"status"`
	NodeCount    int    `json:"node_count"`
	NodeResults  string `json:"node_results"`
	StartedAt    string `json:"started_at"`
	FinishedAt   string `json:"finished_at"`
	TriggeredBy  string `json:"triggered_by"`
	Error        string `json:"error"`
	CreatedAt    string `json:"created_at"`
}

// ExecutionRepo provides CRUD operations for execution history.
type ExecutionRepo struct {
	db *sql.DB
}

// NewExecutionRepo creates a new execution history repository.
func NewExecutionRepo(db *sql.DB) *ExecutionRepo {
	return &ExecutionRepo{db: db}
}

// Save persists a new execution record.
func (r *ExecutionRepo) Save(runID, workflowID, workflowName, workflowJSON string, nodeCount int, nodeResults []byte, startedAt time.Time, triggeredBy string) error {
	_, err := r.db.Exec(
		`INSERT INTO executions (run_id, workflow_id, workflow_name, workflow_json, status, node_count, node_results, started_at, triggered_by)
		 VALUES (?, ?, ?, ?, 'running', ?, ?, ?, ?)`,
		runID, workflowID, workflowName, workflowJSON, nodeCount, string(nodeResults), startedAt.UTC().Format(time.RFC3339), triggeredBy,
	)
	return err
}

// Complete marks an execution as completed or failed.
func (r *ExecutionRepo) Complete(runID string, status string, nodeResults []byte, finishedAt time.Time, topErr string) error {
	_, err := r.db.Exec(
		`UPDATE executions SET status=?, node_results=?, finished_at=?, error=? WHERE run_id=?`,
		status, string(nodeResults), finishedAt.UTC().Format(time.RFC3339), topErr, runID,
	)
	return err
}

// List returns recent execution history entries, newest first.
func (r *ExecutionRepo) List(limit int) ([]ExecutionRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(
		`SELECT id, run_id, workflow_id, workflow_name, workflow_json, status, node_count, node_results, started_at, COALESCE(finished_at,''), triggered_by, COALESCE(error,''), created_at
		 FROM executions ORDER BY created_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list executions: %w", err)
	}
	defer rows.Close()
	return scanExecutions(rows)
}

// Get returns a single execution record by run ID.
func (r *ExecutionRepo) Get(runID string) (*ExecutionRecord, error) {
	row := r.db.QueryRow(
		`SELECT id, run_id, workflow_id, workflow_name, workflow_json, status, node_count, node_results, started_at, COALESCE(finished_at,''), triggered_by, COALESCE(error,''), created_at
		 FROM executions WHERE run_id=?`, runID,
	)
	rec := &ExecutionRecord{}
	err := row.Scan(&rec.ID, &rec.RunID, &rec.WorkflowID, &rec.WorkflowName, &rec.WorkflowJSON,
		&rec.Status, &rec.NodeCount, &rec.NodeResults, &rec.StartedAt, &rec.FinishedAt,
		&rec.TriggeredBy, &rec.Error, &rec.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get execution %s: %w", runID, err)
	}
	return rec, nil
}

// Delete removes old execution records before a given date.
// created_at uses SQLite's datetime('now') default ("YYYY-MM-DD HH:MM:SS"),
// so the cutoff must be formatted identically — RFC3339's 'T' separator would
// compare lexically greater than ' ' and match rows it should not.
func (r *ExecutionRepo) DeleteBefore(before time.Time) (int64, error) {
	res, err := r.db.Exec(`DELETE FROM executions WHERE created_at < ?`, before.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		return 0, fmt.Errorf("prune executions: %w", err)
	}
	return res.RowsAffected()
}

// scanExecutions scans all rows from a query.
func scanExecutions(rows *sql.Rows) ([]ExecutionRecord, error) {
	var records []ExecutionRecord
	for rows.Next() {
		var rec ExecutionRecord
		if err := rows.Scan(&rec.ID, &rec.RunID, &rec.WorkflowID, &rec.WorkflowName, &rec.WorkflowJSON,
			&rec.Status, &rec.NodeCount, &rec.NodeResults, &rec.StartedAt, &rec.FinishedAt,
			&rec.TriggeredBy, &rec.Error, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan execution row: %w", err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// Helper: marshal node results to JSON bytes for storage.
func MarshalNodeResults(results interface{}) ([]byte, error) {
	return json.Marshal(results)
}

// Helper: unmarshal node results from stored JSON.
func UnmarshalNodeResults(data string, v interface{}) error {
	if data == "" {
		return nil
	}
	return json.Unmarshal([]byte(data), v)
}
