# Gen-Code - AI-Powered Code Generation Service

一个基于大模型的代码生成服务，可以根据用户提示词自动生成代码项目并推送到GitHub。

## 功能特点

- 🤖 **智能代码生成**: 使用DeepSeek或OpenAI等大模型生成完整项目代码
- 📦 **自动创建仓库**: 自动在GitHub上创建仓库并推送代码
- 🔄 **实时状态推送**: 通过SSE实时查看生成进度
- 🌐 **多语言支持**: 支持Go、Python、JavaScript/TypeScript、Markdown等多种语言
- 🔌 **可扩展模型**: 支持多种LLM提供商（DeepSeek、OpenAI等）
- ⚡ **异步处理**: 立即响应请求，后台异步处理任务

## 系统架构

```
用户请求 → HTTP API → 任务管理器 → 代码生成器 → LLM → GitHub
                ↓
            SSE推送实时状态
```

## 快速开始

### 前置要求

- Go 1.21+
- GitHub Personal Access Token
- DeepSeek API Key 或 OpenAI API Key

### 安装

1. 克隆仓库：
```bash
git clone https://github.com/cosmos-link/gen-code.git
cd gen-code
```

2. 安装依赖：
```bash
go mod download
```

3. 配置环境变量：
```bash
cp .env.example .env
# 编辑 .env 文件，填入你的 API Keys
```

### 配置说明

在 `.env` 文件中配置以下环境变量：

```env
# GitHub Token（必填）
GITHUB_TOKEN=ghp_xxxxxxxxxxxx

# GitHub Owner（可选，不填则在当前用户下创建仓库）
GITHUB_OWNER=your_username_or_org

# DeepSeek API Key（使用DeepSeek时必填）
DEEPSEEK_API_KEY=sk-xxxxxxxxxxxx
DEEPSEEK_BASE_URL=https://api.deepseek.com

# OpenAI API Key（使用OpenAI时必填）
OPENAI_API_KEY=sk-xxxxxxxxxxxx
OPENAI_BASE_URL=https://api.openai.com/v1

# 默认使用的模型
DEFAULT_MODEL=deepseek
```

### 运行服务

```bash
go run cmd/server/main.go
```

服务将在 `http://localhost:8080` 上启动。

## API 使用

### 1. 创建代码生成任务

**POST** `/api/v1/generate`

```bash
curl -X POST http://localhost:8080/api/v1/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "创建一个Python Flask Web应用，包含用户认证和RESTful API",
    "repo_name": "my-flask-app",
    "model": "deepseek"
  }'
```

**响应:**
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "message": "Task created"
}
```

### 2. 查询任务状态

**GET** `/api/v1/task/:task_id`

```bash
curl http://localhost:8080/api/v1/task/550e8400-e29b-41d4-a716-446655440000
```

**响应:**
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "prompt": "创建一个Python Flask Web应用",
  "repo_name": "my-flask-app",
  "status": "completed",
  "message": "Successfully generated and pushed code!",
  "repo_url": "https://github.com/username/my-flask-app",
  "created_at": "2026-01-12T10:00:00Z",
  "updated_at": "2026-01-12T10:05:00Z"
}
```

### 3. 实时订阅状态（SSE）

**GET** `/api/v1/status/:task_id`

```bash
curl -N http://localhost:8080/api/v1/status/550e8400-e29b-41d4-a716-446655440000
```

**SSE事件流:**
```
event: status
data: {"status":"generating","message":"正在生成代码..."}

event: status
data: {"status":"creating_repo","message":"正在创建GitHub仓库..."}

event: status
data: {"status":"pushing","message":"正在推送代码..."}

event: status
data: {"status":"completed","message":"完成","repo_url":"https://github.com/user/my-flask-app"}
```

### 4. 健康检查

**GET** `/health`

```bash
curl http://localhost:8080/health
```

## 任务状态

| 状态 | 说明 |
|------|------|
| `pending` | 任务已创建，等待处理 |
| `generating` | 正在调用大模型生成代码 |
| `merging_files` | 正在拼接和处理文件 |
| `creating_repo` | 正在创建GitHub仓库 |
| `pushing` | 正在推送代码到仓库 |
| `completed` | 任务完成 |
| `failed` | 任务失败 |

## Web前端示例

### 使用JavaScript订阅SSE

```javascript
// 创建任务
async function createTask() {
  const response = await fetch('http://localhost:8080/api/v1/generate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      prompt: '创建一个Go语言的REST API服务',
      repo_name: 'my-go-api',
      model: 'deepseek'
    })
  });
  
  const data = await response.json();
  return data.task_id;
}

// 订阅状态更新
function subscribeToStatus(taskId) {
  const eventSource = new EventSource(
    `http://localhost:8080/api/v1/status/${taskId}`
  );
  
  eventSource.addEventListener('status', (event) => {
    const data = JSON.parse(event.data);
    console.log('Status:', data.status, data.message);
    
    if (data.status === 'completed') {
      console.log('Repository URL:', data.repo_url);
      eventSource.close();
    } else if (data.status === 'failed') {
      console.error('Error:', data.error);
      eventSource.close();
    }
  });
  
  eventSource.onerror = (error) => {
    console.error('SSE Error:', error);
    eventSource.close();
  };
}

// 使用
const taskId = await createTask();
subscribeToStatus(taskId);
```

## 项目结构

```
gen-code/
├── cmd/
│   └── server/
│       └── main.go              # 服务入口
├── internal/
│   ├── api/
│   │   ├── handler.go          # HTTP处理器
│   │   └── sse.go              # SSE实现
│   ├── llm/
│   │   ├── client.go           # LLM客户端接口
│   │   ├── deepseek.go         # DeepSeek实现
│   │   └── openai.go           # OpenAI实现
│   ├── github/
│   │   └── client.go           # GitHub API客户端
│   ├── generator/
│   │   └── generator.go        # 代码生成核心逻辑
│   ├── task/
│   │   ├── manager.go          # 任务管理器
│   │   └── status.go           # 任务状态
│   └── config/
│       └── config.go           # 配置管理
├── go.mod
├── go.sum
├── .env.example
├── DESIGN.md                   # 设计文档
└── README.md
```

## 支持的文件类型

- `.go` - Go源代码
- `.py` - Python源代码
- `.js` / `.ts` - JavaScript/TypeScript
- `.md` - Markdown文档
- `.json` / `.yaml` - 配置文件

## 开发

### 构建

```bash
go build -o gen-code ./cmd/server
```

### 运行

```bash
./gen-code
```

### 测试

```bash
go test ./...
```

## 部署

### Docker部署（待实现）

```bash
docker build -t gen-code .
docker run -p 8080:8080 --env-file .env gen-code
```

### 环境变量配置

生产环境建议通过环境变量注入配置，而不是使用 `.env` 文件。

## 注意事项

1. **API Key安全**: 不要将API Key提交到版本控制系统
2. **GitHub Token权限**: 确保GitHub Token有创建仓库的权限
3. **GitHub Owner**: 如果不配置GITHUB_OWNER，则会在当前认证用户下创建仓库；如果配置，可以在指定用户或组织下创建
4. **临时文件**: 服务会在`./tmp`目录下生成临时文件，任务完成后自动清理；**如果任务失败，文件会保留在 `./tmp/<task-id>/` 目录中以便调试**
5. **速率限制**: 注意LLM和GitHub API的速率限制
6. **并发控制**: 通过 `MAX_CONCURRENT_TASKS` 控制并发任务数
7. **Prompt质量**: 查看 [最佳实践](BEST_PRACTICES.md) 了解如何编写高质量的prompt，避免常见问题

## 故障排查

### 常见问题

1. **GitHub Token无效**
   - 检查token是否过期
   - 确认token有 `repo` 权限
   - 如果配置了 `GITHUB_OWNER` 且遇到404错误，可能需要 `admin:org` 权限

2. **LLM API调用失败**
   - 检查API Key是否正确
   - 确认网络连接正常
   - 查看API余额是否充足
   - 如果遇到"JSON解析错误"，说明prompt可能太复杂，请简化描述（参考[最佳实践](BEST_PRACTICES.md)）

3. **推送失败**
   - 检查仓库是否已存在
   - 确认网络连接正常

4. **GitHub 404错误** (组织仓库)
   - 删除或注释 `.env` 中的 `GITHUB_OWNER` 配置
   - 或确保你是该组织成员且Token有正确权限

5. **调试失败任务**
   - 失败的任务文件会保留在 `./tmp/<task-id>/` 目录
   - 查看 [DEBUGGING.md](DEBUGGING.md) 了解详细的调试方法

📖 更多详细的故障排查步骤，请查看 [TROUBLESHOOTING.md](TROUBLESHOOTING.md)

## 贡献

欢迎提交Issue和Pull Request！

## 许可证

MIT License

## 联系方式

- GitHub: https://github.com/cosmos-link/gen-code
- Issues: https://github.com/cosmos-link/gen-code/issues

---

Made with ❤️ by Cosmos Link
