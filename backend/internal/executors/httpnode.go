package executors

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type HTTPNodeExecutor struct{}

func (e *HTTPNodeExecutor) Execute(ctx context.Context, input ExecuteInput) (ExecuteOutput, error) {
	cfg := input.Config
	url := stringFromMap(input.Data, "url", "")
	if url == "" {
		url = stringFromMap(cfg, "url", "")
	}
	if url == "" {
		return ExecuteOutput{}, fmt.Errorf("url is required for http node")
	}
	method := strings.ToUpper(stringFromMap(cfg, "method", "GET"))

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return ExecuteOutput{}, fmt.Errorf("creating request: %w", err)
	}
	if hdrsRaw, ok := cfg["headers"]; ok {
		if hdrs, ok := hdrsRaw.(map[string]interface{}); ok {
			for k, v := range hdrs {
				req.Header.Set(k, fmt.Sprintf("%v", v))
			}
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ExecuteOutput{}, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ExecuteOutput{}, fmt.Errorf("reading response: %w", err)
	}
	return ExecuteOutput{
		Data: map[string]interface{}{
			"body":        string(body),
			"status_code": resp.StatusCode,
		},
		Logs: []string{fmt.Sprintf("%s %s → %d", method, url, resp.StatusCode)},
	}, nil
}
