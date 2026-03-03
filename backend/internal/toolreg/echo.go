package toolreg

import (
	"context"
	"encoding/json"
	"fmt"
)

func RegisterEchoTool(r *Registry) {
	def := &ToolDef{
		Type: "function",
		Function: FuncSpec{
			Name:        "echo",
			Description: "Echo back a message",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type":        "string",
						"description": "The message to echo back",
					},
				},
				"required": []string{"message"},
			},
		},
	}
	fn := func(_ context.Context, argsJSON string) (string, error) {
		var input struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &input); err != nil {
			return "", fmt.Errorf("parsing echo args: %w", err)
		}
		result, err := json.Marshal(map[string]string{"result": "Echo: " + input.Message})
		if err != nil {
			return "", fmt.Errorf("marshaling echo result: %w", err)
		}
		return string(result), nil
	}
	r.Register("echo", def, fn)
}
