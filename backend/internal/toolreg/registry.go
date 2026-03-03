package toolreg

import (
	"context"
	"fmt"
)

// ToolDef is an OpenAI-compatible function tool definition
type ToolDef struct {
	Type     string   `json:"type"`
	Function FuncSpec `json:"function"`
}

type FuncSpec struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// ToolFunc is a function that executes a tool given JSON args and returns JSON result
type ToolFunc func(ctx context.Context, argsJSON string) (string, error)

type Registry struct {
	defs  map[string]*ToolDef
	funcs map[string]ToolFunc
}

func New() *Registry {
	return &Registry{
		defs:  make(map[string]*ToolDef),
		funcs: make(map[string]ToolFunc),
	}
}

func (r *Registry) Register(name string, def *ToolDef, fn ToolFunc) {
	r.defs[name] = def
	r.funcs[name] = fn
}

// GetDef returns OpenAI tool definition format
func (r *Registry) GetDef(name string) map[string]interface{} {
	def, ok := r.defs[name]
	if !ok {
		return nil
	}
	return map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        def.Function.Name,
			"description": def.Function.Description,
			"parameters":  def.Function.Parameters,
		},
	}
}

func (r *Registry) Execute(ctx context.Context, name, argsJSON string) (string, error) {
	fn, ok := r.funcs[name]
	if !ok {
		return "", fmt.Errorf("tool %q not found", name)
	}
	return fn(ctx, argsJSON)
}

func (r *Registry) List() []string {
	names := make([]string, 0, len(r.defs))
	for name := range r.defs {
		names = append(names, name)
	}
	return names
}
