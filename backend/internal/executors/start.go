package executors

import "context"

type StartExecutor struct{}

func (e *StartExecutor) Execute(_ context.Context, input ExecuteInput) (ExecuteOutput, error) {
	out := make(map[string]interface{})
	for k, v := range input.Data {
		out[k] = v
	}
	return ExecuteOutput{Data: out, Logs: []string{"start node executed"}}, nil
}
