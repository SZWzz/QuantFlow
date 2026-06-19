package schedule

import (
	"context"
	"time"
)

type Task struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	CronExpr      string     `json:"cron_expr"`
	WorkflowID    string     `json:"workflow_id"`
	Enabled       bool       `json:"enabled"`
	TimeoutSec    int        `json:"timeout_sec"`
	LastRunAt     *time.Time `json:"last_run_at"`
	LastRunStatus string     `json:"last_run_status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type WorkflowExecutor interface {
	Execute(ctx context.Context, workflowID string) (executionID string, err error)
}

type Notifier interface {
	SendTaskCompleted(taskName string, success bool, message string)
}
