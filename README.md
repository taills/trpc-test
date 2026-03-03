# Workflow Demo — tRPC-Test

一个**单体式**工作流引擎示例，让你可以可视化地组合、保存、触发和检查基于 DAG 的工作流。

## 架构

```
frontend/          单文件 HTML/JS 画布界面
backend/
  cmd/server/      HTTP 服务器入口点
  internal/
    dsl/           Workflow JSON DSL 类型 + 验证器
    engine/        DAG 执行器（拓扑排序 + 节点调度）
    executors/     节点类型执行器
      start.go     向下游传递工作流输入
      end.go       收集最终输出
      agent.go     ReAct LLM 代理（兼容 OpenAI，支持工具调用）
      httpnode.go  HTTP GET/POST 请求节点
      mcp_adapter.go  为未来集成预留的 MCP 接口
    toolreg/       内置工具注册表 (echo, http_get)
    registry/      动态注册器（按需注册工具和执行器）
    store/         线程安全的内存运行存储（+ 可选 JSON 文件）
    api/           REST API 处理器
examples/          示例工作流 JSON 文件
```

## 快速开始

### 前置要求
- Go 1.21+

### 运行

```bash
# 从仓库根目录
cd backend
go build -o ../workflow-server ./cmd/server
cd ..

# 启动服务器（在 :8080 提供前端服务）
FRONTEND_DIR=./frontend ./workflow-server

# 可选：配置兼容 OpenAI 的 LLM 以启用 Agent 节点
export OPENAI_API_KEY=sk-...
export OPENAI_BASE_URL=https://api.openai.com/v1   # 可选，默认为 OpenAI
export PORT=8080                                    # 可选，默认 8080
```

在浏览器中打开 http://localhost:8080。

### 运行测试

```bash
cd backend
go test ./...
```

## 示例演示

1. 打开 http://localhost:8080。
2. 点击 **Load Demo** 加载示例 `Start → Agent(ReAct+Tool) → End` 工作流到编辑器。
3. 点击 **Save Workflow** 将工作流注册到后端。
4. 点击 **Run Workflow** 执行工作流，并在右侧面板查看每个节点的结果。

**没有 API key 时**，Agent 节点返回模拟响应并记录 `"LLM not configured, returning mock response"`。

**有 API key 时**，Agent 运行 ReAct 循环：调用 LLM，检测 `tool_calls`，分发到工具注册表（echo / http_get），将工具结果附加回对话，并重复直到 LLM 给出最终答案。

你也可以直接通过 API 加载示例工作流：

```bash
curl -X POST http://localhost:8080/api/workflows \
  -H "Content-Type: application/json" \
  -d @examples/demo.json

curl -X POST http://localhost:8080/api/workflows/demo-workflow/run
```

## 动态注册功能

系统支持根据 Workflow JSON 内容自动注册所需的工具（tools）和执行器（executors），无需在启动时预先注册所有组件。

### 工作原理

当通过 API 创建 workflow 时：

1. 系统解析 workflow JSON
2. 扫描所有节点类型，收集所需的执行器
3. 扫描 agent 节点的 `tools` 配置，收集所需的工具
4. 验证所有需要的组件都可用
5. 自动注册需要的组件
6. 如果有未知的工具或执行器，返回错误

### 可用组件

**可用工具**:
- `echo` - 回声工具
- `http_get` - HTTP GET 请求工具

**可用执行器**:
- `start` - 起始节点执行器
- `end` - 结束节点执行器
- `agent` - Agent 节点执行器（支持 ReAct + Tools）
- `http` - HTTP 节点执行器
- `echo` - Echo 节点执行器（透传）

### 优势

1. **按需加载** - 只注册 workflow 实际需要的组件
2. **错误检测** - 在创建 workflow 时就能发现缺失的依赖
3. **易于扩展** - 添加新工具或执行器只需在 `makeAvailableTools()` 或 `makeAvailableExecutors()` 中注册
4. **类型安全** - 编译时检查所有可用组件

### 添加新组件

**添加新工具**：

1. 在 `internal/toolreg` 中创建注册函数（如 `RegisterMyTool`）
2. 在 `registry/dynamic.go` 的 `makeAvailableTools()` 中添加：

```go
"my_tool": toolreg.RegisterMyTool,
```

**添加新执行器**：

1. 在 `internal/executors` 中创建执行器实现
2. 在 `registry/dynamic.go` 的 `makeAvailableExecutors()` 中添加：

```go
"my_type": func(toolReg *toolreg.Registry) executors.Executor {
    return &executors.MyExecutor{}
},
```

## JSON DSL 结构

```json
{
  "id":   "string (唯一的工作流 ID)",
  "name": "string",
  "nodes": [
    {
      "id":   "string (唯一的节点 ID)",
      "type": "start | end | agent | http | echo",
      "config": { /* 特定类型的配置 */ }
    }
  ],
  "edges": [
    { "source": "nodeID", "target": "nodeID" }
  ],
  "inputs": { "key": "value" }
}
```

### 节点类型和配置

| 类型    | 配置字段 | 描述 |
|---------|---------|------|
| `start` | —       | 入口点；向下游传递 `inputs` |
| `end`   | —       | 终端节点；收集上游输出 |
| `agent` | `model`, `prompt`, `tools`（工具名称数组）, `memory`（布尔值）, `llm_api_key`, `llm_base_url` | 支持工具调用的 ReAct LLM 代理 |
| `http`  | `url`, `method`（默认 GET）, `headers` | HTTP 请求节点 |
| `echo`  | —       | 透传节点 |

### 内置工具

| 工具       | 描述                          | 参数              |
|-----------|------------------------------|-------------------|
| `echo`     | 回显消息                     | `{"message":"…"}` |
| `http_get` | 获取 URL 并返回正文           | `{"url":"…"}`     |

### MCP 集成（预留）

`internal/executors/mcp_adapter.go` 定义了 `MCPAdapter` 接口，用于未来的模型上下文协议工具服务器。实现此接口以连接 MCP 服务器，并在运行时将其工具注入到代理的工具注册表中。

## API 参考

| 方法 | 路径 | 描述 |
|------|-----|------|
| `POST` | `/api/workflows` | 上传/保存工作流 JSON |
| `GET`  | `/api/workflows` | 列出所有工作流 |
| `GET`  | `/api/workflows/{id}` | 获取特定工作流 |
| `POST` | `/api/workflows/{id}/run` | 触发工作流运行 |
| `GET`  | `/api/runs` | 列出所有运行结果 |
| `GET`  | `/api/runs/{run_id}` | 获取特定运行结果（包含节点级输入/输出和日志） |

### 运行结果架构

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
      "error":      "string (如果有错误)",
      "started_at": "RFC3339",
      "ended_at":   "RFC3339"
    }
  ]
}
```

## 测试动态注册

```bash
# 启动服务器
cd backend
go run ./cmd/server

# 创建 workflow（会自动注册所需组件）
curl -X POST http://localhost:8080/api/workflows \
  -H "Content-Type: application/json" \
  -d @../examples/demo.json

# 运行 workflow
curl -X POST http://localhost:8080/api/workflows/demo-workflow/run

# 测试错误处理（使用未知工具）
curl -X POST http://localhost:8080/api/workflows \
  -H "Content-Type: application/json" \
  -d @../examples/invalid-tool.json
```

## 兼容性

- ✅ 完全向后兼容现有的 workflow 定义
- ✅ 不影响现有 API 接口
- ✅ 支持所有现有的节点类型和工具

