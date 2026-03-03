package dsl

import "fmt"

var validNodeTypes = map[string]bool{
	"start": true,
	"end":   true,
	"agent": true,
	"http":  true,
	"echo":  true,
}

func Validate(w *Workflow) error {
	if w.ID == "" {
		return fmt.Errorf("workflow id is required")
	}
	nodeIDs := make(map[string]bool)
	startCount, endCount := 0, 0
	for _, n := range w.Nodes {
		if n.ID == "" {
			return fmt.Errorf("node id is required")
		}
		if !validNodeTypes[n.Type] {
			return fmt.Errorf("unknown node type: %s", n.Type)
		}
		nodeIDs[n.ID] = true
		if n.Type == "start" {
			startCount++
		}
		if n.Type == "end" {
			endCount++
		}
	}
	if startCount != 1 {
		return fmt.Errorf("workflow must have exactly one start node, got %d", startCount)
	}
	if endCount != 1 {
		return fmt.Errorf("workflow must have exactly one end node, got %d", endCount)
	}
	for _, e := range w.Edges {
		if !nodeIDs[e.Source] {
			return fmt.Errorf("edge source %q not found", e.Source)
		}
		if !nodeIDs[e.Target] {
			return fmt.Errorf("edge target %q not found", e.Target)
		}
	}
	// cycle detection via DFS
	adj := make(map[string][]string)
	for _, e := range w.Edges {
		adj[e.Source] = append(adj[e.Source], e.Target)
	}
	visited := make(map[string]int) // 0=unvisited, 1=in-progress, 2=done
	var dfs func(id string) error
	dfs = func(id string) error {
		if visited[id] == 1 {
			return fmt.Errorf("cycle detected at node %q", id)
		}
		if visited[id] == 2 {
			return nil
		}
		visited[id] = 1
		for _, next := range adj[id] {
			if err := dfs(next); err != nil {
				return err
			}
		}
		visited[id] = 2
		return nil
	}
	for _, n := range w.Nodes {
		if err := dfs(n.ID); err != nil {
			return err
		}
	}
	return nil
}
