package engine

import (
	"context"
	"testing"

	"github.com/taills/trpc-test/backend/internal/dsl"
	"github.com/taills/trpc-test/backend/internal/executors"
)

type passthroughExec struct{}

func (p *passthroughExec) Execute(_ context.Context, input executors.ExecuteInput) (executors.ExecuteOutput, error) {
	return executors.ExecuteOutput{Data: input.Data, Logs: []string{"passthrough"}}, nil
}

func TestSimpleStartEnd(t *testing.T) {
	w := &dsl.Workflow{
		ID: "test",
		Nodes: []dsl.Node{
			{ID: "start", Type: "start"},
			{ID: "end", Type: "end"},
		},
		Edges:  []dsl.Edge{{Source: "start", Target: "end"}},
		Inputs: map[string]interface{}{"key": "value"},
	}
	reg := NewExecutorRegistry()
	reg.Register("start", &passthroughExec{})
	reg.Register("end", &passthroughExec{})
	result, err := Run(context.Background(), w, reg)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "success" {
		t.Fatalf("expected success, got %s", result.Status)
	}
	if len(result.NodeResults) != 2 {
		t.Fatalf("expected 2 node results, got %d", len(result.NodeResults))
	}
}

func TestOutputPropagation(t *testing.T) {
	w := &dsl.Workflow{
		ID: "test",
		Nodes: []dsl.Node{
			{ID: "start", Type: "start"},
			{ID: "end", Type: "end"},
		},
		Edges:  []dsl.Edge{{Source: "start", Target: "end"}},
		Inputs: map[string]interface{}{"msg": "hello"},
	}
	reg := NewExecutorRegistry()
	reg.Register("start", &passthroughExec{})
	reg.Register("end", &passthroughExec{})
	result, err := Run(context.Background(), w, reg)
	if err != nil {
		t.Fatal(err)
	}
	endResult := result.NodeResults[1]
	if endResult.Inputs["msg"] != "hello" {
		t.Fatalf("expected msg=hello in end inputs, got %v", endResult.Inputs["msg"])
	}
}
