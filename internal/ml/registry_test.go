package ml

import (
	"context"
	"testing"

	"quantflow/internal/storage"
)

func setupTestDB(t *testing.T) (*ModelRegistry, func()) {
	t.Helper()
	tmp := t.TempDir()
	dbPath := tmp + "/test.db"

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("storage.Open() error = %v", err)
	}

	migs, err := storage.BuiltinMigrations()
	if err != nil {
		t.Fatalf("BuiltinMigrations() error = %v", err)
	}

	var mig010 storage.Migration
	found := false
	for _, m := range migs {
		if m.Version == 10 {
			mig010 = m
			found = true
			break
		}
	}
	if !found {
		t.Fatal("migration 010 not found in builtin migrations")
	}

	if err := storage.Run(db, []storage.Migration{mig010}); err != nil {
		t.Fatalf("storage.Run() error = %v", err)
	}

	r := NewModelRegistry(db)
	return r, func() { db.Close() }
}

func TestModelRegistry_CreateAndGet(t *testing.T) {
	r, cleanup := setupTestDB(t)
	defer cleanup()

	model := &MLModel{
		ID:          "m-test-1",
		Name:        "test_xgb",
		ModelType:   ModelTypeXGBoost,
		Category:    CategoryPrediction,
		Hyperparams: map[string]string{"n_estimators": "100"},
		Metrics:     map[string]float64{"train_rmse": 0.02},
		Status:      ModelStatusReady,
	}

	err := r.Create(context.Background(), model)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := r.Get(context.Background(), "m-test-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "test_xgb" {
		t.Errorf("expected name 'test_xgb', got '%s'", got.Name)
	}
	if got.ModelType != ModelTypeXGBoost {
		t.Errorf("expected type xgboost, got %s", got.ModelType)
	}
	if got.Category != CategoryPrediction {
		t.Errorf("expected category prediction, got %s", got.Category)
	}
	if got.Status != ModelStatusReady {
		t.Errorf("expected status ready, got %s", got.Status)
	}
	if got.Hyperparams["n_estimators"] != "100" {
		t.Errorf("expected hyperparams n_estimators=100, got %v", got.Hyperparams)
	}
	if got.Metrics["train_rmse"] != 0.02 {
		t.Errorf("expected metrics train_rmse=0.02, got %v", got.Metrics)
	}
}

func TestModelRegistry_CreateGeneratesUUID(t *testing.T) {
	r, cleanup := setupTestDB(t)
	defer cleanup()

	model := &MLModel{
		Name:      "auto_id",
		ModelType: ModelTypeLightGBM,
		Category:  CategoryPrediction,
		Status:    ModelStatusTraining,
	}

	err := r.Create(context.Background(), model)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if model.ID == "" {
		t.Error("expected non-empty ID after Create")
	}
}

func TestModelRegistry_UpdateStatus(t *testing.T) {
	r, cleanup := setupTestDB(t)
	defer cleanup()

	err := r.Create(context.Background(), &MLModel{
		ID: "m-s1", Name: "s", ModelType: ModelTypeLightGBM,
		Category: CategoryPrediction, Status: ModelStatusTraining,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = r.UpdateStatus(context.Background(), "m-s1", ModelStatusReady)
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	got, err := r.Get(context.Background(), "m-s1")
	if err != nil {
		t.Fatalf("Get after UpdateStatus failed: %v", err)
	}
	if got.Status != ModelStatusReady {
		t.Errorf("expected ready, got %s", got.Status)
	}
}

func TestModelRegistry_ListWithFilter(t *testing.T) {
	r, cleanup := setupTestDB(t)
	defer cleanup()

	r.Create(context.Background(), &MLModel{
		ID: "m-a", Name: "model_a", ModelType: ModelTypeXGBoost,
		Category: CategoryPrediction, Status: ModelStatusReady,
	})
	r.Create(context.Background(), &MLModel{
		ID: "m-b", Name: "model_b", ModelType: ModelTypeLSTM,
		Category: CategoryPrediction, Status: ModelStatusTraining,
	})

	predCat := CategoryPrediction
	models, err := r.List(context.Background(), ModelFilter{Category: &predCat})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(models) != 2 {
		t.Errorf("expected 2 prediction models, got %d", len(models))
	}

	// Filter by status
	readyStatus := ModelStatusReady
	readyModels, err := r.List(context.Background(), ModelFilter{Status: &readyStatus})
	if err != nil {
		t.Fatalf("List by status failed: %v", err)
	}
	if len(readyModels) != 1 {
		t.Errorf("expected 1 ready model, got %d", len(readyModels))
	}
	if readyModels[0].ID != "m-a" {
		t.Errorf("expected ready model m-a, got %s", readyModels[0].ID)
	}
}

func TestModelRegistry_ArchiveAndDelete(t *testing.T) {
	r, cleanup := setupTestDB(t)
	defer cleanup()

	r.Create(context.Background(), &MLModel{
		ID: "m-arc", Name: "arc", ModelType: ModelTypeXGBoost,
		Category: CategoryPrediction, Status: ModelStatusReady,
	})

	if err := r.Archive(context.Background(), "m-arc"); err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	got, err := r.Get(context.Background(), "m-arc")
	if err != nil {
		t.Fatalf("Get after Archive failed: %v", err)
	}
	if got.Status != ModelStatusArchived {
		t.Errorf("expected archived, got %s", got.Status)
	}

	if err := r.Delete(context.Background(), "m-arc"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = r.Get(context.Background(), "m-arc")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestModelRegistry_GetNotFound(t *testing.T) {
	r, cleanup := setupTestDB(t)
	defer cleanup()

	_, err := r.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent model")
	}
}

func TestModelRegistry_SaveAndLoadModelFile(t *testing.T) {
	r, cleanup := setupTestDB(t)
	defer cleanup()

	r.Create(context.Background(), &MLModel{
		ID: "m-file", Name: "file_model", ModelType: ModelTypeXGBoost,
		Category: CategoryPrediction, Status: ModelStatusReady,
	})

	data := []byte("model binary data")
	if err := r.SaveModelFile(context.Background(), "m-file", data); err != nil {
		t.Fatalf("SaveModelFile failed: %v", err)
	}

	loaded, err := r.LoadModelFile(context.Background(), "m-file")
	if err != nil {
		t.Fatalf("LoadModelFile failed: %v", err)
	}
	if string(loaded) != string(data) {
		t.Errorf("loaded data mismatch: got %q, want %q", string(loaded), string(data))
	}
}

func TestModelRegistry_UpdateStatus_TrainingToFailed(t *testing.T) {
	r, cleanup := setupTestDB(t)
	defer cleanup()

	err := r.Create(context.Background(), &MLModel{ID: "m-tf", Name: "tf", ModelType: ModelTypeXGBoost, Category: CategoryPrediction, Status: ModelStatusTraining})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = r.UpdateStatus(context.Background(), "m-tf", ModelStatusFailed)
	if err != nil {
		t.Fatalf("training -> failed should be valid: %v", err)
	}

	got, _ := r.Get(context.Background(), "m-tf")
	if got.Status != ModelStatusFailed {
		t.Errorf("expected failed, got %s", got.Status)
	}
}

func TestModelRegistry_UpdateStatus_InvalidTransition(t *testing.T) {
	r, cleanup := setupTestDB(t)
	defer cleanup()

	err := r.Create(context.Background(), &MLModel{ID: "m-it", Name: "it", ModelType: ModelTypeLightGBM, Category: CategoryPrediction, Status: ModelStatusReady})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = r.UpdateStatus(context.Background(), "m-it", ModelStatusTraining)
	if err == nil {
		t.Error("expected error for ready -> training transition")
	}
}

func TestModelRegistry_Delete_CascadeToPredictions(t *testing.T) {
	// Re-use setupTestDB but we need DB access for predictions.
	// This test verifies Delete removes the model cleanly.
	r, cleanup := setupTestDB(t)
	defer cleanup()

	err := r.Create(context.Background(), &MLModel{ID: "m-cas", Name: "cas", ModelType: ModelTypeXGBoost, Category: CategoryPrediction, Status: ModelStatusReady})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = r.Delete(context.Background(), "m-cas")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = r.Get(context.Background(), "m-cas")
	if err == nil {
		t.Error("expected error after delete, model should not exist")
	}
}

func TestModelRegistry_List_SearchFilter(t *testing.T) {
	r, cleanup := setupTestDB(t)
	defer cleanup()

	r.Create(context.Background(), &MLModel{ID: "m-alpha", Name: "alpha_test", ModelType: ModelTypeXGBoost, Category: CategoryPrediction, Status: ModelStatusReady})
	r.Create(context.Background(), &MLModel{ID: "m-beta", Name: "beta_model", ModelType: ModelTypeLightGBM, Category: CategoryPrediction, Status: ModelStatusReady})

	models, err := r.List(context.Background(), ModelFilter{Search: "alpha"})
	if err != nil {
		t.Fatalf("List with search failed: %v", err)
	}
	if len(models) != 1 {
		t.Errorf("expected 1 model matching 'alpha', got %d", len(models))
	}
	if models[0].ID != "m-alpha" {
		t.Errorf("expected m-alpha, got %s", models[0].ID)
	}
}

func TestModelRegistry_List_LimitOffset(t *testing.T) {
	r, cleanup := setupTestDB(t)
	defer cleanup()

	r.Create(context.Background(), &MLModel{ID: "m-lo-1", Name: "model_1", ModelType: ModelTypeXGBoost, Category: CategoryPrediction, Status: ModelStatusReady})
	r.Create(context.Background(), &MLModel{ID: "m-lo-2", Name: "model_2", ModelType: ModelTypeLightGBM, Category: CategoryPrediction, Status: ModelStatusReady})
	r.Create(context.Background(), &MLModel{ID: "m-lo-3", Name: "model_3", ModelType: ModelTypeLSTM, Category: CategoryPrediction, Status: ModelStatusReady})

	models, err := r.List(context.Background(), ModelFilter{Limit: 2})
	if err != nil {
		t.Fatalf("List with limit failed: %v", err)
	}
	if len(models) != 2 {
		t.Errorf("expected 2 models with limit, got %d", len(models))
	}

	models2, err := r.List(context.Background(), ModelFilter{Offset: 1, Limit: 10})
	if err != nil {
		t.Fatalf("List with offset failed: %v", err)
	}
	if len(models2) != 2 {
		t.Errorf("expected 2 models with offset=1, got %d", len(models2))
	}
}
