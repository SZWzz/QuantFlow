package storage

import (
	"testing"

	"quantflow/internal/workflow"
)

func setupRepo(t *testing.T) (*WorkflowRepo, func()) {
	t.Helper()
	tmp := t.TempDir()
	dbPath := tmp + "/test.db"

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	migrations, err := BuiltinMigrations()
	if err != nil {
		t.Fatalf("BuiltinMigrations() error = %v", err)
	}
	if err := Run(db, migrations); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	repo := NewWorkflowRepo(db)
	return repo, func() { db.Close() }
}

func TestWorkflowRepo_SaveAndLoad(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	wf := &workflow.Workflow{
		ID: "wf-test-1", Name: "Test Workflow",
		Nodes: []workflow.NodeInstance{{ID: "n1", NodeType: "sma", Params: map[string]any{"period": float64(10)}}},
		Edges: []workflow.Edge{},
	}

	if err := repo.Save(wf); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := repo.Load("wf-test-1", nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Name != "Test Workflow" {
		t.Errorf("Name = %q, want Test Workflow", loaded.Name)
	}
	if len(loaded.Nodes) != 1 {
		t.Errorf("len(Nodes) = %d, want 1", len(loaded.Nodes))
	}
}

func TestWorkflowRepo_SaveCreatesNewVersion(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	wf := &workflow.Workflow{ID: "wf-v", Name: "V1"}
	if err := repo.Save(wf); err != nil {
		t.Fatalf("Save() v1 error = %v", err)
	}

	wf.Name = "V2"
	if err := repo.Save(wf); err != nil {
		t.Fatalf("Save() v2 error = %v", err)
	}

	latest, err := repo.Load("wf-v", nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if latest.Name != "V2" {
		t.Errorf("latest Name = %q, want V2", latest.Name)
	}

	v1 := 1
	v1wf, err := repo.Load("wf-v", &v1)
	if err != nil {
		t.Fatalf("Load(v1) error = %v", err)
	}
	if v1wf.Name != "V1" {
		t.Errorf("v1 Name = %q, want V1", v1wf.Name)
	}
}

func TestWorkflowRepo_List(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	wf1 := &workflow.Workflow{ID: "wf-list-a", Name: "WF A"}
	wf2 := &workflow.Workflow{ID: "wf-list-b", Name: "WF B"}
	wf3 := &workflow.Workflow{ID: "wf-list-c", Name: "WF C"}
	repo.Save(wf1)
	repo.Save(wf2)
	repo.Save(wf3)

	metas, err := repo.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(metas) < 3 {
		t.Errorf("len(metas) = %d, want >= 3", len(metas))
	}
}

func TestWorkflowRepo_LoadNotFound(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()
	_, err := repo.Load("nonexistent", nil)
	if err == nil {
		t.Error("expected error for nonexistent workflow")
	}
}
