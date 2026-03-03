package engine

import (
	"testing"
	"github.com/taills/trpc-test/backend/internal/dsl"
)

func TestLinearTopo(t *testing.T) {
	w := &dsl.Workflow{
		Nodes: []dsl.Node{
			{ID: "start"}, {ID: "a"}, {ID: "end"},
		},
		Edges: []dsl.Edge{
			{Source: "start", Target: "a"},
			{Source: "a", Target: "end"},
		},
	}
	order, err := TopoSort(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(order))
	}
}

func TestDiamondTopo(t *testing.T) {
	w := &dsl.Workflow{
		Nodes: []dsl.Node{
			{ID: "start"}, {ID: "a"}, {ID: "b"}, {ID: "end"},
		},
		Edges: []dsl.Edge{
			{Source: "start", Target: "a"},
			{Source: "start", Target: "b"},
			{Source: "a", Target: "end"},
			{Source: "b", Target: "end"},
		},
	}
	order, err := TopoSort(w)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(order))
	}
}

func TestCycleTopo(t *testing.T) {
	w := &dsl.Workflow{
		Nodes: []dsl.Node{
			{ID: "start"}, {ID: "a"}, {ID: "end"},
		},
		Edges: []dsl.Edge{
			{Source: "start", Target: "a"},
			{Source: "a", Target: "end"},
			{Source: "end", Target: "a"},
		},
	}
	_, err := TopoSort(w)
	if err == nil {
		t.Fatal("expected cycle error")
	}
}
