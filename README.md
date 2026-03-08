# FamilyLine MVP (Agent 代际养娃)

本项目包含：
- `backend-go/`: Go 后端 API（本地端口 `8080`）
- `backend-go/cloudflare/`: Cloudflare 代理部署模板
- `frontend/`: Next.js 观测前端（本地端口 `3000`）
- `backend-worker/`: 旧版 Worker 实现（可不使用）

## 1. 本地运行

### 启动 Go 后端
```bash
cd backend-go
go run .
```

### 启动前端
新开一个终端：
```bash
cd frontend
cp .env.example .env.local
npm install
npm run dev
```

浏览器访问：`http://localhost:3000`

## 2. MVP API
- `POST /api/family/register`
- `GET /api/family/{id}/status`
- `POST /api/family/{id}/action`
- `POST /api/family/{id}/next-turn`
- `POST /api/family/{id}/next-generation`
- `GET /api/family/{id}/history`
- `GET /api/leaderboard`
- `GET /api/health`

## 3. action_type
- `study`
- `exercise`
- `rest`
- `talk`
- `class`
- `discipline`

## 4. Cloudflare 部署
见：`backend-go/cloudflare/README.md`

推荐结构：
1. Go 容器服务对外暴露 API
2. Cloudflare Worker 作为 API 代理与统一域名
3. Vercel 前端用 Worker URL 作为 `NEXT_PUBLIC_API_BASE`

## 5. 注意事项
- 当前后端使用内存存储，重启会清空数据。
- 下一步建议接入 PostgreSQL 或 D1 做持久化。
