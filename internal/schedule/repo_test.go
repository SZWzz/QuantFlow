package schedule

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupRepoDB(t *testing.T) *Repo {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`CREATE TABLE schedule_tasks (
		id TEXT PRIMARY KEY, name TEXT NOT NULL, cron_expr TEXT NOT NULL,
		workflow_id TEXT NOT NULL, enabled INTEGER DEFAULT 1,
		timeout_sec INTEGER DEFAULT 1800, last_run_at TEXT, last_run_status TEXT,
		created_at TEXT DEFAULT (datetime('now')),
		updated_at TEXT DEFAULT (datetime('now'))
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	return NewRepo(db)
}

func TestRepo_Create(t *testing.T) {
	r := setupRepoDB(t)
	task := &Task{Name: "test", CronExpr: "0 * * * *", WorkflowID: "wf-1", Enabled: true}
	if err := r.Create(task); err != nil {
		t.Fatal(err)
	}
	if task.ID == "" {
		t.Error("expected auto-generated ID")
	}
}

func TestRepo_List(t *testing.T) {
	r := setupRepoDB(t)
	r.Create(&Task{Name: "a", CronExpr: "* * * * * *", WorkflowID: "w1", Enabled: true})
	r.Create(&Task{Name: "b", CronExpr: "* * * * * *", WorkflowID: "w2", Enabled: false})
	tasks, err := r.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestRepo_Update(t *testing.T) {
	r := setupRepoDB(t)
	task := &Task{Name: "original", CronExpr: "0 * * * *", WorkflowID: "w1", Enabled: true}
	r.Create(task)
	task.Name = "updated"
	task.CronExpr = "*/5 * * * *"
	if err := r.Update(task); err != nil {
		t.Fatal(err)
	}
	tasks, _ := r.List()
	if len(tasks) != 1 || tasks[0].Name != "updated" || tasks[0].CronExpr != "*/5 * * * *" {
		t.Error("update did not persist")
	}
}

func TestRepo_Delete(t *testing.T) {
	r := setupRepoDB(t)
	r.Create(&Task{Name: "del", CronExpr: "* * * * * *", WorkflowID: "w1", Enabled: true})
	tasks, _ := r.List()
	if len(tasks) != 1 {
		t.Fatal("expected 1 before delete")
	}
	if err := r.Delete(tasks[0].ID); err != nil {
		t.Fatal(err)
	}
	tasks, _ = r.List()
	if len(tasks) != 0 {
		t.Error("expected empty after delete")
	}
}

func TestRepo_RecordRun(t *testing.T) {
	r := setupRepoDB(t)
	task := &Task{Name: "run", CronExpr: "* * * * * *", WorkflowID: "w1", Enabled: true}
	r.Create(task)
	if err := r.RecordRun(task.ID, "success"); err != nil {
		t.Fatal(err)
	}
	tasks, _ := r.List()
	if tasks[0].LastRunStatus != "success" {
		t.Errorf("LastRunStatus = %q, want 'success'", tasks[0].LastRunStatus)
	}
}
