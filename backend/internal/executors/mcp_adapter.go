package executors

import "context"

// MCPAdapter provides an interface for Model Context Protocol tool servers.
// This is a reserved interface for future MCP integration.
type MCPAdapter interface {
	// Connect connects to an MCP server
	Connect(ctx context.Context, serverURL string) error
	// ListTools returns tools available from the MCP server
	ListTools(ctx context.Context) ([]interface{}, error)
	// Close closes the connection
	Close() error
}
