package ml

import "time"

// ModelType enumerates supported ML model types.
type ModelType string

const (
	ModelTypeXGBoost     ModelType = "xgboost"
	ModelTypeLightGBM    ModelType = "lightgbm"
	ModelTypeLSTM        ModelType = "lstm"
	ModelTypeTransformer ModelType = "transformer"
	ModelTypePPO         ModelType = "ppo"
	ModelTypeDQN         ModelType = "dqn"
	ModelTypeSAC         ModelType = "sac"
	ModelTypeGARCH       ModelType = "garch"
)

// ValidModelTypes returns all valid model type strings.
func ValidModelTypes() []string {
	return []string{
		string(ModelTypeXGBoost), string(ModelTypeLightGBM),
		string(ModelTypeLSTM), string(ModelTypeTransformer),
		string(ModelTypePPO), string(ModelTypeDQN), string(ModelTypeSAC),
		string(ModelTypeGARCH),
	}
}

// ModelCategory groups models by application domain.
type ModelCategory string

const (
	CategoryPrediction  ModelCategory = "prediction"
	CategoryAlphaMining ModelCategory = "alpha_mining"
	CategoryRL          ModelCategory = "rl"
	CategoryRisk        ModelCategory = "risk"
)

// ModelStatus tracks model lifecycle.
type ModelStatus string

const (
	ModelStatusTraining ModelStatus = "training"
	ModelStatusReady    ModelStatus = "ready"
	ModelStatusFailed   ModelStatus = "failed"
	ModelStatusArchived ModelStatus = "archived"
)

// MLModel is the core model entity.
type MLModel struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	ModelType   ModelType          `json:"model_type"`
	Category    ModelCategory      `json:"category"`
	Hyperparams map[string]string  `json:"hyperparams"`
	Metrics     map[string]float64 `json:"metrics"`
	FilePath    string             `json:"file_path"`
	FileBytes   []byte             `json:"-"`
	Status      ModelStatus        `json:"status"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// ModelFilter is used for listing models with optional filters.
type ModelFilter struct {
	ModelType *ModelType
	Category  *ModelCategory
	Status    *ModelStatus
	Search    string // matches name substring
	Limit     int
	Offset    int
}

// TrainingJob tracks an in-progress or completed training operation.
type TrainingJob struct {
	ID        string    `json:"id"`
	ModelID   string    `json:"model_id"`
	ModelType string    `json:"model_type"`
	Status    string    `json:"status"` // running/completed/failed
	Progress  float64   `json:"progress"`
	StartedAt time.Time `json:"started_at"`
}
