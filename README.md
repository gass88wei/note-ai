# AI 笔记助手

基于 Hybrid 搜索（BM25 + 向量 + RRF 融合）的桌面端个人知识库。支持 Markdown 笔记管理、增量索引、语义问答。

## 功能

- **笔记管理**：创建、编辑、删除 Markdown 笔记，支持分类和标签
- **AI 问答**：基于笔记内容的智能对话，引用来源并标注匹配方式
- **Hybrid 搜索**：BM25 关键词 + Qdrant 向量 + RRF 融合排序
- **分块索引**：笔记自动切分为小块，每块独立建向量，精准检索
- **增量更新**：SHA256 hash 检测变更，只处理变化的笔记
- **句子提取**：搜索结果只提取与查询最相关的句子，节省 LLM token

## 技术架构

```
用户提问
  │
  ├── BM25 关键词搜索（纯 Go 倒排索引）
  ├── Embedding → Qdrant HNSW 向量搜索
  └── RRF 融合排序
        │
        ▼
  分块检索 + 相邻块合并 + 句子提取
        │
        ▼
  LLM 回答（引用来源，标注匹配方式）
```

**技术栈**：

| 模块 | 技术 |
|------|------|
| 桌面框架 | Wails v2 (Go + Vue 3) |
| 向量搜索 | Qdrant (Rust, HNSW) |
| 关键词搜索 | 纯 Go BM25 倒排索引 |
| 融合排序 | RRF (Reciprocal Rank Fusion) |
| Embedding | LM Studio API |
| LLM 问答 | LM Studio / Ollama (OpenAI 兼容 API) |
| 数据库 | SQLite (pure Go) |
| 前端 | Vue 3 + TypeScript |

## 环境要求

| 工具 | 用途 | 必需？ |
|------|------|--------|
| [LM Studio](https://lmstudio.ai) | 加载 Embedding 模型 + LLM 模型 | **是** |
| Go 1.22+ | 从源码构建 | 仅源码构建 |
| Node.js 18+ | 从源码构建 | 仅源码构建 |
| [Wails CLI](https://wails.io) | 从源码构建 | 仅源码构建 |

**不需要 Python。** 纯 Go 实现，Qdrant 已打包在发行版中。

## 快速开始

### 方式一：下载便携版（推荐）

从 [Releases](https://github.com/yourname/note-ai/releases) 下载 `AI笔记助手.zip`，解压后直接运行。

### 方式二：从源码构建

```bash
# 1. 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 2. 克隆项目
git clone https://github.com/yourname/note-ai.git
cd note-ai

# 3. 构建
wails build
# 输出到 build/bin/AI笔记助手.exe
```

### 使用步骤

1. **打开 LM Studio**，加载两个模型：
   - Embedding 模型（如 `nomic-embed-text`、`embeddinggemma-300m`）
   - LLM 聊天模型（如 `qwen3.5-4b`、`llama3`）

2. **运行 `AI笔记助手.exe`**（Qdrant 会自动在后台启动）

3. **进入设置页面**，配置：
   - **LLM 问答配置**：API URL + 模型名称
   - **Embedding 配置**：API URL + 模型名称
   - 点「测试 LLM 连接」验证

4. **创建笔记** → 点「重建索引」

5. **开始 AI 对话** 🎉

## 项目结构

```
note-ai/
├── main.go                      # Wails 入口
├── app.go                       # Wails App 绑定
├── database.go                  # SQLite（notes, chunks, settings）
├── leann_service.go             # 搜索引擎（BM25 + Qdrant + RRF + 分块）
├── llm_client.go                # LLM API 客户端
├── note_service.go              # 笔记业务 + 增量索引
├── api_handler.go               # Wails API 层
│
├── internal/
│   ├── indexer/
│   │   ├── tokenizer.go         # 中文分词器
│   │   ├── bm25.go              # BM25 倒排索引
│   │   ├── hybrid.go            # RRF 融合排序
│   │   ├── chunker.go           # 笔记智能分块
│   │   ├── extractor.go         # 查询感知句子提取
│   │   └── embedder.go          # LM Studio Embedding API 客户端
│   └── vector/
│       └── qdrant.go            # Qdrant HTTP 客户端（自动启动）
│
├── frontend/
│   └── src/
│       ├── App.vue              # 主布局 + 状态指示器
│       ├── views/
│       │   ├── NotesView.vue    # 笔记管理
│       │   ├── ChatView.vue     # AI 对话
│       │   └── SettingsView.vue # 设置
│       └── stores/index.ts      # Pinia 状态管理
│
└── build/bin/
    ├── AI笔记助手.exe           # 主程序
    └── qdrant.exe               # Qdrant 向量搜索引擎（自动启动）
```

## 运行机制

### 搜索流程

```
用户提问 "男员工陪产假几天"
  │
  ├─ BM25: 倒排索引查找包含 "陪产假" 的分块
  ├─ Vector: embedding → Qdrant HNSW 搜索语义相似的分块
  └─ RRF: 根据查询长度动态调整 BM25/Vector 权重
        │
        ▼
  合并同笔记相邻分块 → 句子提取（只保留最相关的句子）
        │
        ▼
  传给 LLM: "[来源1 关键词] 公司考勤制度\n婚假：法定3天+公司福利7天=共10天"
        │
        ▼
  LLM 回答: "根据来源1，男员工陪产假为 15 天"
```

### 增量更新

```
首次启动 → 全量索引（存 SHA256 hash）
  │
之后修改笔记 → 比较 hash → 只处理变化的笔记
  │
新增 → 追加分块到 BM25 + Qdrant
修改 → 移除旧分块 → 追加新分块
删除 → 从 BM25 + Qdrant 移除
```

### 内存优化

- Chunk 文本存储在 SQLite，搜索时按需读取 top-K 条
- 内存中只保留 ID 映射（~5MB），不缓存全文
- 10,000 笔记内存占用约 60MB



## 常见问题

**Q: Embedding 连接失败？**
A: 确保 LM Studio 已加载 embedding 模型，并在设置中配置正确的 URL 和模型名。

**Q: LLM 回答太慢？**
A: 在 LM Studio 中增加 LLM 的 GPU offload 层数，Embedding 模型可改为 CPU 运行。

**Q: 搜索找不到笔记？**
A: 确保已点「重建索引」。新增/修改笔记会自动更新索引。

**Q: 需要安装 Python 吗？**
A: 不需要。纯 Go 实现，LM Studio 提供 Embedding 和 LLM 服务。

**Q: 支持哪些模型？**
A: 任何 OpenAI 兼容 API 的模型都支持（LM Studio、Ollama、vLLM 等）。

## License

MIT
