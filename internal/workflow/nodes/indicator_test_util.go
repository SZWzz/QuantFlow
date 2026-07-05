package nodes

import (
	"context"
	"testing"

	"quantflow/internal/workflow"
)

type indicatorTestCase struct {
	name   string
	node   workflow.BaseNode
	inputs map[string]any
	params map[string]any
	check  func(t *testing.T, output map[string]any)
}

func runIndicatorTest(t *testing.T, tc indicatorTestCase) {
	t.Helper()
	out, err := tc.node.Execute(context.Background(), tc.inputs, tc.params, nil)
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", tc.name, err)
	}
	tc.check(t, out)
}
