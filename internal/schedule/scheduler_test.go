package schedule

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

type testExec struct{ lastWF string }

func (e *testExec) Execute(ctx context.Context, wfID string) (string, error) {
	e.lastWF = wfID
	return "exec-1", nil
}

type testNotifier struct {
	lastName string
	lastOk   bool
}

func (n *testNotifier) SendTaskCompleted(name string, ok bool, msg string) {
	n.lastName = name
	n.lastOk = ok
}

func TestScheduler_CreateAndList(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	db.Exec(`CREATE TABLE schedule_tasks (id TEXT PRIMARY KEY, name TEXT NOT NULL, cron_expr TEXT NOT NULL, workflow_id TEXT NOT NULL, enabled INTEGER DEFAULT 1, timeout_sec INTEGER DEFAULT 1800, last_run_at TEXT, last_run_status TEXT, created_at TEXT DEFAULT (datetime('now')), updated_at TEXT DEFAULT (datetime('now')))`)

	s := New(db, &testExec{}, &testNotifier{})
	err = s.CreateTask(&Task{Name: "Test", CronExpr: "0 */5 * * * *", WorkflowID: "wf-1", Enabled: false})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	tasks, _ := s.ListTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Name != "Test" {
		t.Errorf("name = %q", tasks[0].Name)
	}
}

func TestScheduler_AddJob(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	db.Exec(`CREATE TABLE schedule_tasks (id TEXT PRIMARY KEY, name TEXT NOT NULL, cron_expr TEXT NOT NULL, workflow_id TEXT NOT NULL, enabled INTEGER DEFAULT 1, timeout_sec INTEGER DEFAULT 1800, last_run_at TEXT, last_run_status TEXT, created_at TEXT DEFAULT (datetime('now')), updated_at TEXT DEFAULT (datetime('now')))`)

	s := New(db, &testExec{}, &testNotifier{})

	task := &Task{Name: "AddJob", CronExpr: "0 */5 * * * *", WorkflowID: "wf-add", Enabled: true}
	if err := s.CreateTask(task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	tasks, err := s.ListTasks()
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Name != "AddJob" {
		t.Errorf("name = %q, want 'AddJob'", tasks[0].Name)
	}
	if tasks[0].WorkflowID != "wf-add" {
		t.Errorf("workflow_id = %q, want 'wf-add'", tasks[0].WorkflowID)
	}
	if !tasks[0].Enabled {
		t.Error("expected task to be enabled")
	}

	// Verify the cron schedule is registered
	if len(s.cron.Entries()) != 1 {
		t.Errorf("expected 1 cron entry, got %d", len(s.cron.Entries()))
	}
}

func TestScheduler_RemoveJob(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	db.Exec(`CREATE TABLE schedule_tasks (id TEXT PRIMARY KEY, name TEXT NOT NULL, cron_expr TEXT NOT NULL, workflow_id TEXT NOT NULL, enabled INTEGER DEFAULT 1, timeout_sec INTEGER DEFAULT 1800, last_run_at TEXT, last_run_status TEXT, created_at TEXT DEFAULT (datetime('now')), updated_at TEXT DEFAULT (datetime('now')))`)

	s := New(db, &testExec{}, &testNotifier{})

	task := &Task{Name: "RemoveJob", CronExpr: "0 */5 * * * *", WorkflowID: "wf-rm", Enabled: true}
	if err := s.CreateTask(task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Confirm it was added
	tasks, _ := s.ListTasks()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task before remove, got %d", len(tasks))
	}

	// Remove the job
	if err := s.DeleteTask(task.ID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	// Verify gone from DB
	tasks, err = s.ListTasks()
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks after remove, got %d", len(tasks))
	}

	// Verify all cron entries cleared (removeAllEntries was called)
	if len(s.cron.Entries()) != 0 {
		t.Errorf("expected 0 cron entries, got %d", len(s.cron.Entries()))
	}
}

func TestScheduler_ListJobs(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	db.Exec(`CREATE TABLE schedule_tasks (id TEXT PRIMARY KEY, name TEXT NOT NULL, cron_expr TEXT NOT NULL, workflow_id TEXT NOT NULL, enabled INTEGER DEFAULT 1, timeout_sec INTEGER DEFAULT 1800, last_run_at TEXT, last_run_status TEXT, created_at TEXT DEFAULT (datetime('now')), updated_at TEXT DEFAULT (datetime('now')))`)

	s := New(db, &testExec{}, &testNotifier{})

	jobNames := []string{"Job1", "Job2", "Job3"}
	for _, name := range jobNames {
		task := &Task{Name: name, CronExpr: "0 */5 * * * *", WorkflowID: "wf-" + name, Enabled: false}
		if err := s.CreateTask(task); err != nil {
			t.Fatalf("CreateTask(%s): %v", name, err)
		}
	}

	allTasks, err := s.ListTasks()
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(allTasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(allTasks))
	}

	names := make(map[string]bool)
	for _, t := range allTasks {
		names[t.Name] = true
	}
	for _, n := range jobNames {
		if !names[n] {
			t.Errorf("missing task %q in list", n)
		}
	}
}

func TestScheduler_Delete(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	db.Exec(`CREATE TABLE schedule_tasks (id TEXT PRIMARY KEY, name TEXT NOT NULL, cron_expr TEXT NOT NULL, workflow_id TEXT NOT NULL, enabled INTEGER DEFAULT 1, timeout_sec INTEGER DEFAULT 1800, last_run_at TEXT, last_run_status TEXT, created_at TEXT DEFAULT (datetime('now')), updated_at TEXT DEFAULT (datetime('now')))`)

	s := New(db, &testExec{}, &testNotifier{})
	task := &Task{Name: "T", CronExpr: "* * * * * *", WorkflowID: "wf-2", Enabled: false}
	s.CreateTask(task)
	s.DeleteTask(task.ID)
	tasks, _ := s.ListTasks()
	if len(tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(tasks))
	}
}
