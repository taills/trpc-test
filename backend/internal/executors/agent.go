package executors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/taills/trpc-test/backend/internal/toolreg"
)

type AgentExecutor struct {
	ToolRegistry *toolreg.Registry
}

func (e *AgentExecutor) Execute(ctx context.Context, input ExecuteInput) (ExecuteOutput, error) {
	cfg := input.Config
	model := stringFromMap(cfg, "model", "gpt-4o-mini")
	systemPrompt := stringFromMap(cfg, "prompt", "You are a helpful assistant.")
	apiKey := stringFromMap(cfg, "llm_api_key", "")
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	baseURL := stringFromMap(cfg, "llm_base_url", "")
	if baseURL == "" {
		baseURL = os.Getenv("OPENAI_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	message := stringFromMap(input.Data, "message", "Hello")

	var logs []string

	if apiKey == "" {
		logs = append(logs, "LLM not configured, returning mock response")
		return ExecuteOutput{
			Data: map[string]interface{}{
				"response": fmt.Sprintf("Mock response for: %s", message),
				"message":  message,
			},
			Logs: logs,
		}, nil
	}

	// Get tools from registry
	var toolNames []string
	if toolsRaw, ok := cfg["tools"]; ok {
		if toolList, ok := toolsRaw.([]interface{}); ok {
			for _, t := range toolList {
				if name, ok := t.(string); ok {
					toolNames = append(toolNames, name)
				}
			}
		}
	}

	// Build OpenAI-compatible tool definitions
	toolDefs := e.buildToolDefs(toolNames)

	// ReAct loop
	messages := []map[string]interface{}{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": message},
	}

	maxIter := 5
	finalResponse := ""
	for i := 0; i < maxIter; i++ {
		logs = append(logs, fmt.Sprintf("LLM iteration %d", i+1))

		reqBody := map[string]interface{}{
			"model":    model,
			"messages": messages,
		}
		if len(toolDefs) > 0 {
			reqBody["tools"] = toolDefs
		}

		respData, err := callLLM(ctx, baseURL, apiKey, reqBody)
		if err != nil {
			return ExecuteOutput{Logs: logs}, fmt.Errorf("LLM call failed: %w", err)
		}

		choices, ok := respData["choices"].([]interface{})
		if !ok || len(choices) == 0 {
			return ExecuteOutput{Logs: logs}, fmt.Errorf("unexpected LLM response: missing choices")
		}
		choice, ok := choices[0].(map[string]interface{})
		if !ok {
			return ExecuteOutput{Logs: logs}, fmt.Errorf("unexpected LLM response: invalid choice")
		}
		msg, ok := choice["message"].(map[string]interface{})
		if !ok {
			return ExecuteOutput{Logs: logs}, fmt.Errorf("unexpected LLM response: missing message")
		}
		finishReason, _ := choice["finish_reason"].(string)

		messages = append(messages, msg)

		if finishReason == "tool_calls" || finishReason == "function_call" {
			toolCalls, ok := msg["tool_calls"].([]interface{})
			if !ok {
				break
			}
			for _, tc := range toolCalls {
				tcMap, ok := tc.(map[string]interface{})
				if !ok {
					continue
				}
				tcID, _ := tcMap["id"].(string)
				fn, ok := tcMap["function"].(map[string]interface{})
				if !ok {
					continue
				}
				fnName, _ := fn["name"].(string)
				fnArgs, _ := fn["arguments"].(string)

				logs = append(logs, fmt.Sprintf("Calling tool: %s args: %s", fnName, fnArgs))
				toolResult := e.executeTool(ctx, fnName, fnArgs)
				logs = append(logs, fmt.Sprintf("Tool result: %s", toolResult))

				messages = append(messages, map[string]interface{}{
					"role":         "tool",
					"tool_call_id": tcID,
					"content":      toolResult,
				})
			}
		} else {
			if content, ok := msg["content"].(string); ok {
				finalResponse = content
			}
			break
		}
	}

	return ExecuteOutput{
		Data: map[string]interface{}{
			"response": finalResponse,
			"message":  message,
		},
		Logs: logs,
	}, nil
}

func (e *AgentExecutor) buildToolDefs(names []string) []map[string]interface{} {
	var defs []map[string]interface{}
	if e.ToolRegistry == nil {
		return defs
	}
	for _, name := range names {
		t := e.ToolRegistry.GetDef(name)
		if t != nil {
			defs = append(defs, t)
		}
	}
	return defs
}

func (e *AgentExecutor) executeTool(ctx context.Context, name, argsJSON string) string {
	if e.ToolRegistry == nil {
		return "tool registry not available"
	}
	result, err := e.ToolRegistry.Execute(ctx, name, argsJSON)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return result
}

func callLLM(ctx context.Context, baseURL, apiKey string, body map[string]interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("LLM API returned %d: %s", resp.StatusCode, string(b))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func stringFromMap(m map[string]interface{}, key, def string) string {
	if m == nil {
		return def
	}
	v, ok := m[key]
	if !ok {
		return def
	}
	s, ok := v.(string)
	if !ok {
		return def
	}
	return s
}
