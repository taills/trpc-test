package registry

import (
	"fmt"

	"github.com/taills/trpc-test/backend/internal/dsl"
	"github.com/taills/trpc-test/backend/internal/engine"
	"github.com/taills/trpc-test/backend/internal/executors"
	"github.com/taills/trpc-test/backend/internal/toolreg"
)

// ToolRegistrar is a function that registers a tool
type ToolRegistrar func(*toolreg.Registry)

// ExecutorFactory creates a new executor instance
type ExecutorFactory func(*toolreg.Registry) executors.Executor

// DynamicRegistry manages available tools and executors
type DynamicRegistry struct {
	availableTools     map[string]ToolRegistrar
	availableExecutors map[string]ExecutorFactory
}

// NewDynamicRegistry creates a new dynamic registry with all available tools and executors
func NewDynamicRegistry() *DynamicRegistry {
	return &DynamicRegistry{
		availableTools:     makeAvailableTools(),
		availableExecutors: makeAvailableExecutors(),
	}
}

// makeAvailableTools returns a map of all available tool registrars
func makeAvailableTools() map[string]ToolRegistrar {
	return map[string]ToolRegistrar{
		"echo":     toolreg.RegisterEchoTool,
		"http_get": toolreg.RegisterHTTPGetTool,
	}
}

// makeAvailableExecutors returns a map of all available executor factories
func makeAvailableExecutors() map[string]ExecutorFactory {
	return map[string]ExecutorFactory{
		"start": func(toolReg *toolreg.Registry) executors.Executor {
			return &executors.StartExecutor{}
		},
		"end": func(toolReg *toolreg.Registry) executors.Executor {
			return &executors.EndExecutor{}
		},
		"agent": func(toolReg *toolreg.Registry) executors.Executor {
			return &executors.AgentExecutor{ToolRegistry: toolReg}
		},
		"http": func(toolReg *toolreg.Registry) executors.Executor {
			return &executors.HTTPNodeExecutor{}
		},
		"echo": func(toolReg *toolreg.Registry) executors.Executor {
			// echo node just passes through like start
			return &executors.StartExecutor{}
		},
	}
}

// RegisterFromWorkflow analyzes a workflow and dynamically registers required tools and executors
func (dr *DynamicRegistry) RegisterFromWorkflow(
	w *dsl.Workflow,
	toolReg *toolreg.Registry,
	execReg *engine.ExecutorRegistry,
) error {
	// Collect all required tools and node types
	requiredTools := make(map[string]bool)
	requiredExecutors := make(map[string]bool)

	for _, node := range w.Nodes {
		// Mark this node type as required
		requiredExecutors[node.Type] = true

		// If it's an agent node, collect required tools
		if node.Type == "agent" {
			if toolsInterface, ok := node.Config["tools"]; ok {
				if tools, ok := toolsInterface.([]interface{}); ok {
					for _, t := range tools {
						if toolName, ok := t.(string); ok {
							requiredTools[toolName] = true
						}
					}
				}
			}
		}
	}

	// Register required tools
	for toolName := range requiredTools {
		registrar, ok := dr.availableTools[toolName]
		if !ok {
			return fmt.Errorf("tool %q is required but not available", toolName)
		}
		registrar(toolReg)
	}

	// Register required executors
	for nodeType := range requiredExecutors {
		factory, ok := dr.availableExecutors[nodeType]
		if !ok {
			return fmt.Errorf("executor for node type %q is required but not available", nodeType)
		}
		executor := factory(toolReg)
		execReg.Register(nodeType, executor)
	}

	return nil
}

// RegisterAll registers all available tools and executors (for backward compatibility)
func (dr *DynamicRegistry) RegisterAll(
	toolReg *toolreg.Registry,
	execReg *engine.ExecutorRegistry,
) {
	// Register all tools
	for _, registrar := range dr.availableTools {
		registrar(toolReg)
	}

	// Register all executors
	for nodeType, factory := range dr.availableExecutors {
		executor := factory(toolReg)
		execReg.Register(nodeType, executor)
	}
}
