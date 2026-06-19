package capabilities

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"quantflow/internal/ai"
)

// RegisterSkillCapabilities registers the search_skills capability.
// Scans local resources and Python skills directories for matching files.
func RegisterSkillCapabilities(reg *ai.CapabilityRegistry) {
	reg.Register(&ai.Capability{
		Name:        "search_skills",
		Description: "Search the trading skill knowledge base for domain expertise. Use this to get detailed information about trading strategies, risk management, technical analysis, etc.",
		Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {"type": "string", "description": "Search query, e.g. momentum, pairs trading, VaR"},
					"category": {"type": "string", "description": "Optional category filter: technical_analysis, fundamental_analysis, risk_management, market_microstructure, trading_strategies"}
				},
				"required": ["query"]
			}`),
		Handler: func(ctx context.Context, args json.RawMessage) (string, error) {
			var params struct {
				Query    string `json:"query"`
				Category string `json:"category"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("search_skills: %w", err)
			}

			// Scan local resources for matching skill/profile files.
			searchDirs := []string{"resources/agent-profiles", "python/skills"}
			var found []string
			for _, dir := range searchDirs {
				entries, err := os.ReadDir(dir)
				if err != nil {
					continue
				}
				for _, e := range entries {
					name := strings.ToLower(e.Name())
					q := strings.ToLower(params.Query)
					if strings.Contains(name, q) || strings.Contains(strings.ToLower(params.Category), q) {
						ext := filepath.Ext(e.Name())
						found = append(found, fmt.Sprintf("- %s [%s]", e.Name(), strings.TrimPrefix(ext, ".")))
					}
				}
			}

			if len(found) == 0 {
				cat := params.Category
				if cat == "" {
					cat = "all"
				}
				return fmt.Sprintf(
					"No matching skills found for %q in %s category. Available categories: technical_analysis, fundamental_analysis, risk_management, market_microstructure, trading_strategies. Try the resources/agent-profiles or python/skills directories.",
					params.Query, cat,
				), nil
			}

			return fmt.Sprintf("Found %d skill(s) matching %q:\n%s",
				len(found), params.Query, strings.Join(found, "\n")), nil
		},
	})
}
