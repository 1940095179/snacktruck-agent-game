# Cloudflare 部署（Go 后端）

## 架构
1. `backend-go` 用容器运行（Cloudflare Containers 或任意容器平台）
2. `cloudflare/worker-proxy.mjs` 作为统一 API 网关
3. 前端（Vercel）只需要配置一个 API 地址：Worker 域名

## A. 先跑 Go 容器（本地验证）
```bash
cd backend-go
docker build -t familyline-go-api:latest .
docker run --rm -p 8080:8080 familyline-go-api:latest
```
验证：
```bash
curl http://127.0.0.1:8080/api/health
```

## B. 部署 Worker 代理
1. 安装并登录 Wrangler
```bash
npm i -g wrangler
wrangler login
```
2. 修改 `wrangler.toml` 里的 `ORIGIN_URL`
3. 部署
```bash
cd backend-go/cloudflare
wrangler deploy
```

部署后会拿到 `*.workers.dev` 域名，把它填到前端：
```env
NEXT_PUBLIC_API_BASE=https://your-worker.workers.dev
```

## C. 前端部署到 Vercel
前端目录 `frontend/` 里设置环境变量后直接部署。

## 说明
- 这套方案不改 Go 业务代码。
- 生产建议：在 Worker 里加签名校验、IP 限流、日志上报。
