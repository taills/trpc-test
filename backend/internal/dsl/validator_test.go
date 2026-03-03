package dsl

import "testing"

func TestValidWorkflow(t *testing.T) {
	w := &Workflow{
		ID:   "test",
		Name: "Test",
		Nodes: []Node{
			{ID: "start", Type: "start"},
			{ID: "end", Type: "end"},
		},
		Edges: []Edge{{Source: "start", Target: "end"}},
	}
	if err := Validate(w); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
}

func TestMissingStart(t *testing.T) {
	w := &Workflow{
		ID:    "test",
		Nodes: []Node{{ID: "end", Type: "end"}},
	}
	if err := Validate(w); err == nil {
		t.Fatal("expected error for missing start")
	}
}

func TestMissingEnd(t *testing.T) {
	w := &Workflow{
		ID:    "test",
		Nodes: []Node{{ID: "start", Type: "start"}},
	}
	if err := Validate(w); err == nil {
		t.Fatal("expected error for missing end")
	}
}

func TestUnknownNodeType(t *testing.T) {
	w := &Workflow{
		ID: "test",
		Nodes: []Node{
			{ID: "start", Type: "start"},
			{ID: "bad", Type: "unknown"},
			{ID: "end", Type: "end"},
		},
	}
	if err := Validate(w); err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestCycleDetection(t *testing.T) {
	w := &Workflow{
		ID: "test",
		Nodes: []Node{
			{ID: "start", Type: "start"},
			{ID: "a", Type: "echo"},
			{ID: "end", Type: "end"},
		},
		Edges: []Edge{
			{Source: "start", Target: "a"},
			{Source: "a", Target: "end"},
			{Source: "end", Target: "a"},
		},
	}
	if err := Validate(w); err == nil {
		t.Fatal("expected error for cycle")
	}
}

func TestInvalidEdge(t *testing.T) {
	w := &Workflow{
		ID: "test",
		Nodes: []Node{
			{ID: "start", Type: "start"},
			{ID: "end", Type: "end"},
		},
		Edges: []Edge{{Source: "start", Target: "nonexistent"}},
	}
	if err := Validate(w); err == nil {
		t.Fatal("expected error for invalid edge")
	}
}
