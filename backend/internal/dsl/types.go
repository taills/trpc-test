package dsl

// Workflow is the top-level workflow definition
type Workflow struct {
	ID     string                 `json:"id"`
	Name   string                 `json:"name"`
	Nodes  []Node                 `json:"nodes"`
	Edges  []Edge                 `json:"edges"`
	Inputs map[string]interface{} `json:"inputs,omitempty"`
}

// Node represents a single processing node
type Node struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"` // start, end, agent, http, echo
	Config map[string]interface{} `json:"config,omitempty"`
}

// Edge represents a directed connection between nodes
type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}
