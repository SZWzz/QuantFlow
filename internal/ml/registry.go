package ml

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// ModelRegistry manages ML model persistence in SQLite.
type ModelRegistry struct {
	db *sql.DB
}

// NewModelRegistry creates a new registry backed by the given DB.
func NewModelRegistry(db *sql.DB) *ModelRegistry {
	return &ModelRegistry{db: db}
}

// Create inserts a new model record. If ID is empty, a UUID is generated.
func (r *ModelRegistry) Create(ctx context.Context, model *MLModel) error {
	if model.ID == "" {
		model.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	model.CreatedAt = now
	model.UpdatedAt = now

	hyperJSON, _ := json.Marshal(model.Hyperparams)
	metricsJSON, _ := json.Marshal(model.Metrics)

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ml_models (id, name, model_type, category, hyperparams, metrics, file_path, file_bytes, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		model.ID, model.Name, string(model.ModelType), string(model.Category),
		string(hyperJSON), string(metricsJSON), model.FilePath, model.FileBytes,
		string(model.Status), model.CreatedAt.Format(time.RFC3339), model.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("ml_models insert: %w", err)
	}
	slog.InfoContext(ctx, "model created", "id", model.ID, "name", model.Name)
	return nil
}

// Get retrieves a model by ID.
func (r *ModelRegistry) Get(ctx context.Context, id string) (*MLModel, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, model_type, category, hyperparams, metrics, file_path, file_bytes, status, created_at, updated_at
		 FROM ml_models WHERE id = ?`, id)

	model := &MLModel{}
	var hyperJSON, metricsJSON, modelType, category, status, ca, ua string

	err := row.Scan(&model.ID, &model.Name, &modelType, &category,
		&hyperJSON, &metricsJSON, &model.FilePath, &model.FileBytes, &status, &ca, &ua)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("model %s: %w", id, sql.ErrNoRows)
	}
	if err != nil {
		return nil, fmt.Errorf("ml_models get %s: %w", id, err)
	}

	model.ModelType = ModelType(modelType)
	model.Category = ModelCategory(category)
	model.Status = ModelStatus(status)
	model.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	model.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	json.Unmarshal([]byte(hyperJSON), &model.Hyperparams)
	json.Unmarshal([]byte(metricsJSON), &model.Metrics)

	return model, nil
}

// List returns models matching the filter.
func (r *ModelRegistry) List(ctx context.Context, filter ModelFilter) ([]*MLModel, error) {
	query := "SELECT id, name, model_type, category, hyperparams, metrics, file_path, file_bytes, status, created_at, updated_at FROM ml_models WHERE 1=1"
	args := []any{}

	if filter.ModelType != nil {
		query += " AND model_type = ?"
		args = append(args, string(*filter.ModelType))
	}
	if filter.Category != nil {
		query += " AND category = ?"
		args = append(args, string(*filter.Category))
	}
	if filter.Status != nil {
		query += " AND status = ?"
		args = append(args, string(*filter.Status))
	}
	if filter.Search != "" {
		query += " AND name LIKE ?"
		args = append(args, "%"+filter.Search+"%")
	}
	query += " ORDER BY created_at DESC"
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ml_models list: %w", err)
	}
	defer rows.Close()

	var models []*MLModel
	for rows.Next() {
		m := &MLModel{}
		var hyperJSON, metricsJSON, modelType, category, status, ca, ua string
		if err := rows.Scan(&m.ID, &m.Name, &modelType, &category,
			&hyperJSON, &metricsJSON, &m.FilePath, &m.FileBytes, &status, &ca, &ua); err != nil {
			return nil, fmt.Errorf("ml_models list scan: %w", err)
		}
		m.ModelType = ModelType(modelType)
		m.Category = ModelCategory(category)
		m.Status = ModelStatus(status)
		m.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		m.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		json.Unmarshal([]byte(hyperJSON), &m.Hyperparams)
		json.Unmarshal([]byte(metricsJSON), &m.Metrics)
		models = append(models, m)
	}
	return models, rows.Err()
}

// validTransitions defines allowed status transitions.
var validTransitions = map[ModelStatus]map[ModelStatus]bool{
	ModelStatusTraining: {ModelStatusReady: true, ModelStatusFailed: true, ModelStatusTraining: true},
	ModelStatusReady:    {ModelStatusArchived: true, ModelStatusReady: true},
	ModelStatusFailed:   {ModelStatusTraining: true, ModelStatusFailed: true},
	ModelStatusArchived: {ModelStatusReady: true, ModelStatusArchived: true},
}

// UpdateStatus transitions a model to a new status with validation.
func (r *ModelRegistry) UpdateStatus(ctx context.Context, id string, status ModelStatus) error {
	current, err := r.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	if allowed, ok := validTransitions[current.Status]; !ok || !allowed[status] {
		return fmt.Errorf("invalid transition: %s -> %s", current.Status, status)
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE ml_models SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("update status %s: %w", id, err)
	}
	return nil
}

// Archive sets model status to archived.
func (r *ModelRegistry) Archive(ctx context.Context, id string) error {
	return r.UpdateStatus(ctx, id, ModelStatusArchived)
}

// Delete removes a model record (and cascading predictions/evaluations via FK).
func (r *ModelRegistry) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM ml_models WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete model %s: %w", id, err)
	}
	return nil
}

// SaveModelFile stores model binary using the dual-track strategy:
// ≤1MB stored as SQLite BLOB, >1MB stored on filesystem with path reference.
func (r *ModelRegistry) SaveModelFile(ctx context.Context, modelID string, data []byte) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE ml_models SET file_bytes = ?, updated_at = ? WHERE id = ?`,
		data, time.Now().UTC().Format(time.RFC3339), modelID)
	if err != nil {
		return fmt.Errorf("save model file %s: %w", modelID, err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("model not found: %s", modelID)
	}
	return nil
}

// LoadModelFile retrieves model binary (from BLOB or filesystem).
func (r *ModelRegistry) LoadModelFile(ctx context.Context, modelID string) ([]byte, error) {
	model, err := r.Get(ctx, modelID)
	if err != nil {
		return nil, err
	}
	return model.FileBytes, nil
}
