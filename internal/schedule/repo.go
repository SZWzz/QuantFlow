package schedule

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Repo struct{ db *sql.DB }

func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

func (r *Repo) Create(task *Task) error {
	if task.ID == "" {
		task.ID = uuid.New().String()[:8]
	}
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	if task.TimeoutSec == 0 {
		task.TimeoutSec = 1800
	}
	_, err := r.db.Exec(
		`INSERT INTO schedule_tasks (id, name, cron_expr, workflow_id, enabled, timeout_sec, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.Name, task.CronExpr, task.WorkflowID, boolToInt(task.Enabled),
		task.TimeoutSec, task.CreatedAt.Format(time.RFC3339), task.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *Repo) Update(task *Task) error {
	task.UpdatedAt = time.Now()
	_, err := r.db.Exec(
		`UPDATE schedule_tasks SET name=?, cron_expr=?, workflow_id=?, enabled=?, timeout_sec=?, updated_at=? WHERE id=?`,
		task.Name, task.CronExpr, task.WorkflowID, boolToInt(task.Enabled),
		task.TimeoutSec, task.UpdatedAt.Format(time.RFC3339), task.ID,
	)
	return err
}

func (r *Repo) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM schedule_tasks WHERE id = ?", id)
	return err
}

func (r *Repo) List() ([]*Task, error) {
	rows, err := r.db.Query(
		"SELECT id, name, cron_expr, workflow_id, enabled, timeout_sec, created_at, updated_at FROM schedule_tasks ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []*Task
	for rows.Next() {
		task := &Task{}
		var ca, ua string
		var en int
		if err := rows.Scan(&task.ID, &task.Name, &task.CronExpr, &task.WorkflowID, &en, &task.TimeoutSec, &ca, &ua); err != nil {
			return nil, err
		}
		task.Enabled = en != 0
		task.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		task.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (r *Repo) RecordRun(id, status string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.Exec("UPDATE schedule_tasks SET last_run_at = ?, last_run_status = ? WHERE id = ?", now, status, id)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
