package executors

import "context"

type EndExecutor struct{}

func (e *EndExecutor) Execute(_ context.Context, input ExecuteInput) (ExecuteOutput, error) {
	out := make(map[string]interface{})
	for k, v := range input.Data {
		out[k] = v
	}
	return ExecuteOutput{Data: out, Logs: []string{"end node executed"}}, nil
}
