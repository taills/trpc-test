package executors

import "context"

type ExecuteInput struct {
	NodeID string
	Config map[string]interface{}
	Data   map[string]interface{}
}

type ExecuteOutput struct {
	Data map[string]interface{}
	Logs []string
}

type Executor interface {
	Execute(ctx context.Context, input ExecuteInput) (ExecuteOutput, error)
}
