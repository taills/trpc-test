# Workflow Demo — tRPC-Test

A **monolithic** workflow engine demo that lets you visually compose, save, trigger, and inspect DAG-based workflows.

## Architecture

```
frontend/          Single-file HTML/JS canvas UI
backend/
  cmd/server/      HTTP server entry-point
  internal/
    dsl/           Workflow JSON DSL types + validator
    engine/        DAG executor (topological sort + node dispatch)
    executors/     Node type executors
      start.go     Pass workflow inputs downstream
      end.go       Collect final outputs
      agent.go     ReAct LLM agent (OpenAI-compatible, with tool calling)
      httpnode.go  HTTP GET/POST request node
      mcp_adapter.go  Reserved MCP interface for future integration
    toolreg/       Built-in tool registry (echo, http_get)
    store/         Thread-safe in-memory run store (+ optional JSON files)
    api/           REST API handlers
examples/          Sample workflow JSON files
```

## Quick Start

### Prerequisites
- Go 1.21+

### Run

```bash
# From repo root
cd backend
go build -o ../workflow-server ./cmd/server
cd ..

# Start server (serves frontend on :8080)
FRONTEND_DIR=./frontend ./workflow-server

# Optional: configure an OpenAI-compatible LLM to enable the Agent node
export OPENAI_API_KEY=sk-...
export OPENAI_BASE_URL=https://api.openai.com/v1   # optional, defaults to OpenAI
export PORT=8080                                    # optional, default 8080
```

Open http://localhost:8080 in your browser.

### Running Tests

```bash
cd backend
go test ./...
```

## Demo Walkthrough

1. Open http://localhost:8080.
2. Click **Load Demo** to populate the editor with the sample `Start → Agent(ReAct+Tool) → End` workflow.
3. Click **Save Workflow** to register it with the backend.
4. Click **Run Workflow** to execute it and view per-node results in the right panel.

**Without an API key** the Agent node returns a mock response and logs `"LLM not configured, returning mock response"`.

**With an API key** the Agent runs a ReAct loop: it calls the LLM, detects `tool_calls`, dispatches to the tool registry (echo / http_get), appends tool results back to the conversation, and repeats until the LLM gives a final answer.

You can also load the demo workflow directly via the API:

```bash
curl -X POST http://localhost:8080/api/workflows \
  -H "Content-Type: application/json" \
  -d @examples/demo.json

curl -X POST http://localhost:8080/api/workflows/demo-workflow/run
```

## JSON DSL Structure

```json
{
  "id":   "string (unique workflow ID)",
  "name": "string",
  "nodes": [
    {
      "id":   "string (unique node ID)",
      "type": "start | end | agent | http | echo",
      "config": { /* type-specific config */ }
    }
  ],
  "edges": [
    { "source": "nodeID", "target": "nodeID" }
  ],
  "inputs": { "key": "value" }
}
```

### Node Types & Config

| Type    | Config fields | Description |
|---------|--------------|-------------|
| `start` | —            | Entry point; passes `inputs` downstream |
| `end`   | —            | Terminal node; collects upstream outputs |
| `agent` | `model`, `prompt`, `tools` (array of tool names), `memory` (bool), `llm_api_key`, `llm_base_url` | ReAct LLM agent with tool calling |
| `http`  | `url`, `method` (default GET), `headers` | HTTP request node |
| `echo`  | —            | Pass-through node |

### Built-in Tools

| Tool       | Description                          | Arguments         |
|------------|--------------------------------------|-------------------|
| `echo`     | Echoes a message back                | `{"message":"…"}` |
| `http_get` | Fetches a URL and returns the body   | `{"url":"…"}`     |

### MCP Integration (Reserved)

`internal/executors/mcp_adapter.go` defines the `MCPAdapter` interface for future Model Context Protocol tool servers. Implement this interface to connect an MCP server and inject its tools into the agent's tool registry at runtime.

## API Reference

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/workflows` | Upload/save a workflow JSON |
| `GET`  | `/api/workflows` | List all workflows |
| `GET`  | `/api/workflows/{id}` | Get a specific workflow |
| `POST` | `/api/workflows/{id}/run` | Trigger a workflow run |
| `GET`  | `/api/runs` | List all run results |
| `GET`  | `/api/runs/{run_id}` | Get a specific run result (with node-level I/O and logs) |

### Run Result Schema

```json
{
  "run_id": "uuid",
  "workflow_id": "string",
  "status": "success | error",
  "started_at": "RFC3339",
  "ended_at":   "RFC3339",
  "node_results": [
    {
      "node_id":    "string",
      "status":     "success | error",
      "inputs":     {},
      "outputs":    {},
      "logs":       ["…"],
      "error":      "string (if error)",
      "started_at": "RFC3339",
      "ended_at":   "RFC3339"
    }
  ]
}
```
