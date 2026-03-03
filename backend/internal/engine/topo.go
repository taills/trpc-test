package engine

import (
	"fmt"
	"github.com/taills/trpc-test/backend/internal/dsl"
)

// TopoSort returns node IDs in topological order
func TopoSort(w *dsl.Workflow) ([]string, error) {
	inDegree := make(map[string]int)
	adj := make(map[string][]string)
	for _, n := range w.Nodes {
		inDegree[n.ID] = 0
	}
	for _, e := range w.Edges {
		adj[e.Source] = append(adj[e.Source], e.Target)
		inDegree[e.Target]++
	}
	queue := []string{}
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	var result []string
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		result = append(result, cur)
		for _, next := range adj[cur] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	if len(result) != len(w.Nodes) {
		return nil, fmt.Errorf("cycle detected in workflow graph")
	}
	return result, nil
}
