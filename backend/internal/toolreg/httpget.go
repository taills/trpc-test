package toolreg

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func RegisterHTTPGetTool(r *Registry) {
	def := &ToolDef{
		Type: "function",
		Function: FuncSpec{
			Name:        "http_get",
			Description: "Fetch a URL via HTTP GET",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "The URL to fetch",
					},
				},
				"required": []string{"url"},
			},
		},
	}
	fn := func(ctx context.Context, argsJSON string) (string, error) {
		var input struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &input); err != nil {
			return "", fmt.Errorf("parsing http_get args: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, "GET", input.URL, nil)
		if err != nil {
			return "", err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("reading response body: %w", err)
		}
		result, err := json.Marshal(map[string]interface{}{
			"body":        string(body),
			"status_code": resp.StatusCode,
		})
		if err != nil {
			return "", fmt.Errorf("marshaling result: %w", err)
		}
		return string(result), nil
	}
	r.Register("http_get", def, fn)
}
