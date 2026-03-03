# 工作流引擎 Copilot 指导文档

## 项目概述

这是一个**基于 DAG 的工作流引擎**，执行 JSON 定义的工作流，支持动态工具/执行器注册。工作流数据通过节点（start → agent/http/echo/... → end）按拓扑排序顺序流转。

**核心架构：**
- `internal/dsl/` — 工作流、节点、边的类型定义 + 验证器
- `internal/engine/` — DAG 执行器，包含拓扑排序（[engine.go](../backend/internal/engine/engine.go), [topo.go](../backend/internal/engine/topo.go)）
- `internal/executors/` — 节点实现（StartExecutor、AgentExecutor 等）
- `internal/toolreg/` — OpenAI 兼容的工具注册表，用于 LLM 代理
- `internal/registry/` — **动态注册系统**（参见 DynamicRegistry）
- `internal/api/` — REST 处理器，用于工作流 CRUD + 执行
- `internal/store/` — 线程安全的内存运行存储

## AI 使用规范

需要获取任何开发文档时，请使用context7 MCP服务/use context7

### ⚠️ 严格禁止

**绝对禁止滥用 tokens 生成以下内容：**
- ❌ 自动生成冗长的测试文件（除非明确要求）
- ❌ 自动生成详细的变更日志或文档
- ❌ 生成无实际价值的注释或说明文档
- ❌ 重复已有的代码注释或文档
- ❌ 在每次代码修改后生成总结性 markdown 文档

**正确做法：**
- ✅ 只在用户明确要求时生成测试或文档
- ✅ 保持代码简洁，注释精准有用
- ✅ 直接回答问题，避免冗余输出

### 高效使用原则

在协助开发时遵循以下 7 条上下文使用原则：

1. **按需查看** - 只读取实际需要的文件和行数，避免全量读取
2. **精准定位** - 善于使用工具快速定位，而非遍历
3. **复用知识** - 利用已有的 README 和架构说明，避免重复探索
4. **渐进理解** - 从关键文件（见下方列表）开始，逐步扩展
5. **验证假设** - 对不确定的部分查看具体实现，而非猜测
6. **批量操作** - 并行读取独立文件，减少往返次数
7. **简洁输出** - 直接给出答案或修改，避免冗长解释

## 关键模式

### 1. 动态注册（核心创新）
工作流触发**按需注册**工具/执行器。参见 [registry/dynamic.go](../backend/internal/registry/dynamic.go)：
```go
// 添加新工具：
1. 在 internal/toolreg/mytool.go 中创建工具，实现 RegisterMyTool(r *Registry)
2. 添加到 makeAvailableTools()： "my_tool": toolreg.RegisterMyTool

// 添加新执行器：
1. 实现 executors.Executor 接口
2. 在 makeAvailableExecutors() 中添加工厂函数
```

当 POST `/api/workflows` 时：
- 系统扫描工作流 JSON 中所需的节点类型和工具
- 验证所有依赖是否可用
- 自动注册组件到 execRegistry/toolRegistry
- 如果引用了未知工具/执行器则返回错误

### 2. 执行器模式
所有节点类型都实现 `executors.Executor` 接口：
```go
type Executor interface {
    Execute(ctx context.Context, input ExecuteInput) (ExecuteOutput, error)
}
```
- **StartExecutor**: 向下游传递 workflow.Inputs
- **EndExecutor**: 收集所有前驱节点的输出
- **AgentExecutor**: 运行 ReAct 循环，支持 LLM + 工具调用
- **HTTPExecutor**: 发起 HTTP 请求

参见 [executors/executor.go](../backend/internal/executors/executor.go) 的接口定义。

### 3. 工具注册表（OpenAI 格式）
代理节点的工具使用 OpenAI 函数调用模式。示例参见 [toolreg/echo.go](../backend/internal/toolreg/echo.go)：
```go
ToolDef{
    Type: "function",
    Function: FuncSpec{
        Name: "echo",
        Parameters: {...}, // JSON Schema 对象
    }
}
```
工具通过 `ToolFunc` 签名调用：`func(ctx, argsJSON string) (resultJSON string, error)`

### 4. ReAct 代理循环
[executors/agent.go](../backend/internal/executors/agent.go) 实现迭代式 LLM 推理：
1. 发送消息 + 工具定义给 LLM
2. 检查响应中的 `tool_calls`
3. 通过 ToolRegistry 执行工具
4. 将工具结果作为消息附加
5. 重复直到 LLM 返回最终答案（最多 5 次迭代）

通过 `OPENAI_BASE_URL` 支持 OpenAI 兼容的 API（默认为 OpenAI）。

### 5. DAG 执行流程
[engine/engine.go](../backend/internal/engine/engine.go) 的 Run() 函数：
1. 拓扑排序节点（[topo.go](../backend/internal/engine/topo.go)）
2. 按顺序执行每个节点：
   - 合并所有前驱节点的输出作为输入
   - 从 ExecutorRegistry 查找执行器
   - 执行节点，捕获输出/日志/错误
3. 返回包含每个节点详细信息的 RunResult

## 开发工作流

### 构建和运行
```bash
cd backend
go build -o ../workflow-server ./cmd/server
cd ..
FRONTEND_DIR=./frontend ./workflow-server  # 在 :8080 提供服务
```

### 测试
```bash
cd backend
go test ./...  # 运行所有测试
```

**测试模式：**
- 使用模拟执行器（参见 [engine_test.go](../backend/internal/engine/engine_test.go) 中的 `passthroughExec`）
- 验证器测试覆盖 DSL 约束（[dsl/validator_test.go](../backend/internal/dsl/validator_test.go)）
- 拓扑测试验证 DAG 排序（[engine/topo_test.go](../backend/internal/engine/topo_test.go)）

### API 使用
```bash
# 创建工作流（自动注册依赖）
curl -X POST http://localhost:8080/api/workflows \
  -H "Content-Type: application/json" \
  -d @examples/demo.json

# 运行工作流
curl -X POST http://localhost:8080/api/workflows/demo-workflow/run

# 获取运行详情（包含节点级别的输入/输出/日志）
curl http://localhost:8080/api/runs/{run_id}
```

## 约定

### 线程安全
- 所有注册表使用 `sync.RWMutex`（ExecutorRegistry、toolreg.Registry、Store）
- Register/Get 模式：Register 使用写锁，Get 使用读锁

### 错误处理
- 执行器返回 `ExecuteOutput`，其中 `.Logs[]` 用于诊断
- 节点失败时设置 `NodeResult.Status = "error"` 并填充 `.Error` 字段
- 工作流验证在执行**之前**进行（参见 [dsl/validator.go](../backend/internal/dsl/validator.go)）

### 配置
所有配置来自环境变量（无配置文件）：
- `OPENAI_API_KEY` — 代理节点必需
- `OPENAI_BASE_URL` — 自定义 LLM 端点（默认：OpenAI）
- `PORT` — 服务器端口（默认：8080）
- `FRONTEND_DIR` — 前端目录路径

节点级配置通过 JSON 中的 `node.config` 提供（如 agent 的 `model`、`prompt`、`tools[]`）。

### JSON DSL 结构
参见 [dsl/types.go](../backend/internal/dsl/types.go)：
```json
{
  "id": "unique-workflow-id",
  "nodes": [{"id": "node1", "type": "agent", "config": {...}}],
  "edges": [{"source": "node1", "target": "node2"}],
  "inputs": {"key": "value"}  // 传递给 start 节点
}
```

**必需节点：** 恰好一个 `start` 和一个 `end` 节点。  
**验证：** 无循环，所有边引用现有节点，已知节点类型。

## 文件组织

- **添加新节点类型** → `internal/executors/mynode.go` + 在 `registry/dynamic.go` 中注册
- **添加新工具** → `internal/toolreg/mytool.go` + 在 `makeAvailableTools()` 中注册
- **修改工作流 DSL** → 更新 `internal/dsl/types.go` + 验证器
- **修改执行逻辑** → `internal/engine/engine.go`
- **添加 API 端点** → `internal/api/handler.go`

## 常见任务

### 添加新工具
1. 创建 `internal/toolreg/newtool.go`：
   ```go
   func RegisterNewTool(r *Registry) {
       def := &ToolDef{...}  // OpenAI schema
       fn := func(ctx, argsJSON) (string, error) {...}
       r.Register("new_tool", def, fn)
   }
   ```
2. 添加到 `registry/dynamic.go`：`"new_tool": toolreg.RegisterNewTool`
3. 在工作流中引用：`"config": {"tools": ["new_tool"]}`

### 添加新节点类型
1. 在 `internal/executors/mynodetype.go` 中创建执行器，实现 `Executor`
2. 在 `registry/dynamic.go` 的 `makeAvailableExecutors()` 中添加工厂函数
3. 在工作流 JSON 中使用：`{"type": "mynodetype", "config": {...}}`

### 调试工作流运行
- 检查 `/api/runs/{run_id}` 查看节点级日志
- Agent 节点会记录 "LLM iteration N" 标记每次 ReAct 循环
- 缺少 API key → agent 返回模拟响应并记录日志
- 无效工具/执行器 → 工作流创建失败，并列出未知项

## 首次阅读的关键文件
1. [README.md](../README.md) — 架构概览
2. [internal/registry/dynamic.go](../backend/internal/registry/dynamic.go) — 注册系统
3. [internal/engine/engine.go](../backend/internal/engine/engine.go) — 执行流程
4. [internal/executors/agent.go](../backend/internal/executors/agent.go) — ReAct 实现
5. [examples/demo.json](../examples/demo.json) — 完整工作流示例
