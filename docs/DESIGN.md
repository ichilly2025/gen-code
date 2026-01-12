# 代码生成服务设计文档

## 1. 项目概述

一个基于Go语言的后端服务，通过大模型（默认DeepSeek）根据用户提示词生成代码，自动创建GitHub仓库并推送代码。

## 2. 核心功能

### 2.1 输入参数
- `prompt`: 用户的代码生成提示词
- `repoName`: 目标GitHub仓库名称
- `model`: 可选，指定使用的大模型（默认deepseek）

### 2.2 主要流程
1. 接收用户请求，立即返回任务ID
2. 异步执行代码生成流程
3. 通过SSE推送实时状态更新
4. 调用大模型生成代码
5. 处理文件拼接（因token限制）
6. 创建GitHub仓库
7. 推送代码到仓库

## 3. 技术架构

### 3.1 技术栈
- **语言**: Go 1.21+
- **Web框架**: Gin
- **大模型集成**: OpenAI兼容API（DeepSeek, OpenAI, etc.）
- **Git操作**: go-git
- **任务队列**: 内存队列（可扩展为Redis）

### 3.2 项目结构
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
│   │   ├── generator.go        # 代码生成核心逻辑
│   │   └── file_merger.go      # 文件拼接处理
│   ├── task/
│   │   ├── manager.go          # 任务管理器
│   │   └── status.go           # 任务状态
│   └── config/
│       └── config.go           # 配置管理
├── pkg/
│   └── utils/
│       └── file.go             # 文件工具
├── go.mod
├── go.sum
├── .env.example
├── README.md
└── DESIGN.md
```

## 4. API设计

### 4.1 创建代码生成任务
**POST** `/api/v1/generate`

**Request Body:**
```json
{
  "prompt": "创建一个Python Flask Web应用，包含用户认证功能",
  "repo_name": "my-flask-app",
  "model": "deepseek",
  "github_org": "optional-org-name"
}
```

**Response:**
```json
{
  "task_id": "uuid-string",
  "status": "pending",
  "message": "Task created successfully"
}
```

### 4.2 SSE状态订阅
**GET** `/api/v1/status/:task_id`

**SSE Event Stream:**
```
event: status
data: {"status": "generating", "message": "正在生成代码..."}

event: status
data: {"status": "creating_repo", "message": "正在创建GitHub仓库..."}

event: status
data: {"status": "pushing", "message": "正在推送代码..."}

event: status
data: {"status": "completed", "message": "完成", "repo_url": "https://github.com/user/my-flask-app"}

event: error
data: {"status": "failed", "message": "错误信息"}
```

### 4.3 查询任务状态
**GET** `/api/v1/task/:task_id`

**Response:**
```json
{
  "task_id": "uuid-string",
  "status": "completed",
  "message": "Task completed successfully",
  "repo_url": "https://github.com/user/my-flask-app",
  "created_at": "2026-01-12T10:00:00Z",
  "updated_at": "2026-01-12T10:05:00Z"
}
```

## 5. 任务状态流转

```
pending → generating → merging_files → creating_repo → pushing → completed
                ↓           ↓              ↓             ↓
              failed      failed         failed       failed
```

**状态说明:**
- `pending`: 任务已创建，等待处理
- `generating`: 正在调用大模型生成代码
- `merging_files`: 正在拼接和处理文件
- `creating_repo`: 正在创建GitHub仓库
- `pushing`: 正在推送代码到仓库
- `completed`: 任务完成
- `failed`: 任务失败

## 6. 大模型集成

### 6.1 LLM接口设计
```go
type LLMClient interface {
    GenerateCode(ctx context.Context, prompt string) (*GeneratedProject, error)
    GenerateFile(ctx context.Context, prompt string, fileType string) (string, error)
}
```

### 6.2 支持的模型
- DeepSeek (默认)
- OpenAI GPT-4
- 可扩展其他OpenAI兼容API

### 6.3 文件拼接策略
由于token限制，大型项目需要分文件生成：

1. **第一次请求**: 生成项目结构和主要文件列表
2. **后续请求**: 逐个文件生成内容
3. **支持的文件类型**:
   - `.go` - Go源码
   - `.py` - Python源码
   - `.js` / `.ts` - JavaScript/TypeScript
   - `.md` - Markdown文档
   - `.json` / `.yaml` - 配置文件

**生成策略:**
```
Prompt → LLM → Project Structure → For each file → Generate Content → Merge
```

## 7. GitHub集成

### 7.1 GitHub API操作
- 创建仓库: `POST /user/repos`
- 创建文件: 通过git commit和push

### 7.2 认证方式
- Personal Access Token (PAT)
- 支持用户级和组织级仓库

### 7.3 Git操作流程
1. 初始化本地临时仓库
2. 添加生成的文件
3. 创建远程GitHub仓库
4. 推送代码到远程

## 8. 配置管理

### 8.1 环境变量
```env
# 服务配置
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# GitHub配置
GITHUB_TOKEN=your_github_token

# LLM配置
DEEPSEEK_API_KEY=your_deepseek_key
DEEPSEEK_BASE_URL=https://api.deepseek.com
OPENAI_API_KEY=your_openai_key
OPENAI_BASE_URL=https://api.openai.com/v1

# 默认模型
DEFAULT_MODEL=deepseek

# 任务配置
MAX_CONCURRENT_TASKS=5
TASK_TIMEOUT=600
TEMP_DIR=/tmp/gen-code
```

## 9. 错误处理

### 9.1 错误类型
- LLM调用失败
- GitHub API失败
- Git操作失败
- 网络超时
- Token额度不足

### 9.2 重试策略
- LLM调用: 3次重试，指数退避
- GitHub API: 3次重试
- Git操作: 不重试，直接失败

## 10. 安全考虑

1. **API认证**: 使用API Key认证
2. **Rate Limiting**: 限制每个用户的请求频率
3. **输入验证**: 验证prompt和repo名称
4. **Token安全**: 环境变量存储，不记录日志
5. **临时文件清理**: 任务完成后删除临时文件

## 11. 性能优化

1. **并发控制**: 限制同时执行的任务数
2. **缓存策略**: 缓存相似的prompt结果（可选）
3. **流式生成**: 支持流式返回大模型输出
4. **文件分块**: 大文件分块生成避免超时

## 12. 监控与日志

### 12.1 日志级别
- INFO: 任务创建、完成
- WARN: 重试、降级
- ERROR: 失败、异常

### 12.2 关键指标
- 任务成功率
- 平均处理时间
- LLM调用延迟
- GitHub API延迟

## 13. 实现计划

### Phase 1: 基础框架
1. 项目初始化和依赖管理
2. 配置管理
3. HTTP服务器和路由

### Phase 2: 核心功能
1. LLM客户端实现
2. GitHub客户端实现
3. 任务管理器

### Phase 3: 代码生成
1. 代码生成器核心逻辑
2. 文件拼接处理
3. 多语言支持

### Phase 4: SSE和状态管理
1. SSE实现
2. 任务状态流转
3. 错误处理

### Phase 5: 优化和部署
1. 性能优化
2. 测试
3. 部署文档
