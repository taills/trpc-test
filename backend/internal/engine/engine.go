package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/taills/trpc-test/backend/internal/dsl"
	"github.com/taills/trpc-test/backend/internal/executors"
)

type NodeResult struct {
	NodeID    string                 `json:"node_id"`
	Status    string                 `json:"status"`
	Inputs    map[string]interface{} `json:"inputs"`
	Outputs   map[string]interface{} `json:"outputs"`
	Logs      []string               `json:"logs"`
	Error     string                 `json:"error,omitempty"`
	StartedAt time.Time              `json:"started_at"`
	EndedAt   time.Time              `json:"ended_at"`
}

type RunResult struct {
	RunID       string       `json:"run_id"`
	WorkflowID  string       `json:"workflow_id"`
	Status      string       `json:"status"`
	NodeResults []NodeResult `json:"node_results"`
	StartedAt   time.Time    `json:"started_at"`
	EndedAt     time.Time    `json:"ended_at"`
}

type ExecutorRegistry map[string]executors.Executor

func Run(ctx context.Context, w *dsl.Workflow, reg ExecutorRegistry) (*RunResult, error) {
	runID := uuid.New().String()
	result := &RunResult{
		RunID:      runID,
		WorkflowID: w.ID,
		Status:     "running",
		StartedAt:  time.Now(),
	}

	order, err := TopoSort(w)
	if err != nil {
		result.Status = "error"
		result.EndedAt = time.Now()
		return result, err
	}

	nodeMap := make(map[string]*dsl.Node)
	for i := range w.Nodes {
		nodeMap[w.Nodes[i].ID] = &w.Nodes[i]
	}

	// Build adjacency: predecessors map
	predecessors := make(map[string][]string)
	for _, e := range w.Edges {
		predecessors[e.Target] = append(predecessors[e.Target], e.Source)
	}

	nodeOutputs := make(map[string]map[string]interface{})

	for _, nodeID := range order {
		node := nodeMap[nodeID]
		exec, ok := reg[node.Type]
		if !ok {
			return result, fmt.Errorf("no executor for node type %q", node.Type)
		}

		// Merge inputs from predecessors
		inputData := make(map[string]interface{})
		if node.Type == "start" {
			for k, v := range w.Inputs {
				inputData[k] = v
			}
		} else {
			for _, predID := range predecessors[nodeID] {
				for k, v := range nodeOutputs[predID] {
					inputData[k] = v
				}
			}
		}

		nr := NodeResult{
			NodeID:    nodeID,
			Inputs:    inputData,
			StartedAt: time.Now(),
		}

		out, execErr := exec.Execute(ctx, executors.ExecuteInput{
			NodeID: nodeID,
			Config: node.Config,
			Data:   inputData,
		})

		nr.EndedAt = time.Now()
		nr.Logs = out.Logs
		nr.Outputs = out.Data

		if execErr != nil {
			nr.Status = "error"
			nr.Error = execErr.Error()
			result.NodeResults = append(result.NodeResults, nr)
			result.Status = "error"
			result.EndedAt = time.Now()
			return result, execErr
		}
		nr.Status = "success"
		nodeOutputs[nodeID] = out.Data
		result.NodeResults = append(result.NodeResults, nr)
	}

	result.Status = "success"
	result.EndedAt = time.Now()
	return result, nil
}
