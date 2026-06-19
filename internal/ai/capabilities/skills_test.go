package capabilities

import (
	"testing"

	"quantflow/internal/ai"
)

func TestRegisterSkillCapabilities_SearchSkills(t *testing.T) {
	reg := ai.NewCapabilityRegistry()
	RegisterSkillCapabilities(reg)

	cap := reg.GetCapability("search_skills")
	if cap == nil {
		t.Fatal("search_skills capability should be registered")
	}
	if cap.Name != "search_skills" {
		t.Errorf("name = %s, want search_skills", cap.Name)
	}
	if cap.Description == "" {
		t.Error("description should not be empty")
	}
}

func TestRegisterSkillCapabilities_ParametersNotEmpty(t *testing.T) {
	reg := ai.NewCapabilityRegistry()
	RegisterSkillCapabilities(reg)

	cap := reg.GetCapability("search_skills")
	if cap == nil {
		t.Fatal("search_skills not found")
	}
	if len(cap.Parameters) == 0 {
		t.Error("parameters schema should not be empty")
	}
}

func TestSkillCapabilities_HandlerNotNil(t *testing.T) {
	reg := ai.NewCapabilityRegistry()
	RegisterSkillCapabilities(reg)

	cap := reg.GetCapability("search_skills")
	if cap == nil {
		t.Fatal("search_skills not found")
	}
	if cap.Handler == nil {
		t.Fatal("handler should not be nil")
	}
}
