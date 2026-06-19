package ml

import (
	"encoding/json"
	"testing"
)

func TestMLModel_JSONSerialization(t *testing.T) {
	model := MLModel{
		ID:        "m1",
		Name:      "test",
		ModelType: ModelTypeXGBoost,
		Category:  CategoryPrediction,
		Status:    ModelStatusReady,
	}

	data, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}

	var restored MLModel
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}

	if restored.ID != "m1" || restored.ModelType != ModelTypeXGBoost {
		t.Errorf("round-trip failed: got ID=%s type=%s", restored.ID, restored.ModelType)
	}
}

func TestValidModelTypes(t *testing.T) {
	types := ValidModelTypes()
	if len(types) != 8 {
		t.Errorf("expected 8 valid types, got %d", len(types))
	}
}
